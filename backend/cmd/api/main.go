package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/di"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/router"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/server"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/validator"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/config"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/logger"
)

// @title GC Storage API
// @version 1.0
// @description クラウドストレージシステム GC Storage の REST API
// @host localhost:8080
// @BasePath /api/v1
// @schemes http https
// @securityDefinitions.apikey SessionCookie
// @in cookie
// @name session_id
func main() {
	// Logger setup
	if err := logger.Setup(logger.DefaultConfig()); err != nil {
		slog.Error("failed to setup logger", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize DI Container
	container, err := di.NewContainer(ctx, cfg)
	if err != nil {
		slog.Error("failed to initialize container", "error", err)
		os.Exit(1)
	}
	defer container.Close()

	// Initialize UseCases, Handlers, and Middlewares
	container.InitAuthUseCases()
	container.InitProfileUseCases()
	handlers := di.NewHandlers(container)
	middlewares := di.NewMiddlewares(container)

	// Setup Server
	serverConfig := server.DefaultConfig()
	serverConfig.Port = cfg.Server.Port
	serverConfig.Debug = cfg.Server.Debug
	srv := server.NewServer(serverConfig)
	e := srv.Echo()

	// Setup validator and error handler
	e.Validator = validator.NewCustomValidator()
	e.HTTPErrorHandler = middleware.CustomHTTPErrorHandler

	// Global middleware
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.SecurityHeaders())
	e.Use(middleware.CORS())

	// Setup Router
	router.NewRouter(e, handlers, middlewares).Setup()

	// Start server
	slog.Info("starting server", "port", cfg.Server.Port)
	go func() {
		if err := srv.Start(); err != nil {
			slog.Error("server error", "error", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), srv.Config().ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
	slog.Info("server stopped")
}
