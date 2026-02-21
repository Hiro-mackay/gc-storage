package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/di"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/storage"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/worker"
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

	// Initialize MinIO Storage
	slog.Info("connecting to MinIO...")
	minioClient, err := storage.NewMinIOClient(storage.Config{
		Endpoint:        cfg.Storage.Endpoint,
		AccessKeyID:     cfg.Storage.AccessKeyID,
		SecretAccessKey: cfg.Storage.SecretAccessKey,
		BucketName:      cfg.Storage.BucketName,
		UseSSL:          cfg.Storage.UseSSL,
		Region:          "us-east-1",
	})
	if err != nil {
		slog.Error("failed to initialize MinIO client", "error", err)
		os.Exit(1)
	}
	if err := minioClient.EnsureBucket(ctx); err != nil {
		slog.Error("failed to ensure MinIO bucket", "error", err)
		os.Exit(1)
	}
	storageService := storage.NewStorageServiceAdapter(storage.NewStorageService(minioClient))
	slog.Info("connected to MinIO", "endpoint", cfg.Storage.Endpoint, "bucket", cfg.Storage.BucketName)

	// Initialize UseCases, Handlers, and Middlewares
	container.InitAuthUseCases()
	container.InitProfileUseCases()
	container.InitCollaborationUseCases()
	container.InitAuthzUseCases()
	container.InitStorageUseCases(storageService)
	container.InitSharingUseCases()
	container.InitAuditService()
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
	e.Use(middleware.SecurityHeadersWithConfig(middleware.SecurityHeadersConfig{
		EnableHSTS:    cfg.Security.EnableHSTS,
		HSTSMaxAge:    31536000, // 1年
		CSPDirectives: "default-src 'self'",
	}))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     cfg.Security.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Request-ID", middleware.CSRFHeaderName},
		AllowCredentials: true,
		MaxAge:           86400,
	}))
	e.Use(middleware.CSRF())
	if middlewares.Audit != nil {
		e.Use(middlewares.Audit.Inject())
	}

	// Setup Router
	router.NewRouter(e, handlers, middlewares).Setup()

	// Start background workers
	workerMgr := worker.NewManager()
	workerMgr.Register(worker.NewHealthCheckJob(func(ctx context.Context) error {
		return container.PgClient.Pool().Ping(ctx)
	}))
	workerMgr.Start()

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
	workerMgr.Shutdown(10 * time.Second)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), srv.Config().ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
	slog.Info("server stopped")
}
