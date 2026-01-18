package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/cache"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/email"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/oauth"
	infraRepo "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/handler"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/presenter"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/server"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/validator"
	authcmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/command"
	authqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/jwt"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/logger"
)

func main() {
	// Logger setup
	if err := logger.Setup(logger.DefaultConfig()); err != nil {
		slog.Error("failed to setup logger", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Configuration
	config := loadConfig()

	// PostgreSQL
	slog.Info("connecting to PostgreSQL...")
	pgClient, err := database.NewPostgresClient(ctx, config.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer pgClient.Close()
	slog.Info("connected to PostgreSQL")

	// Redis
	slog.Info("connecting to Redis...")
	redisConfig := cache.DefaultConfig()
	redisConfig.URL = config.RedisURL
	redisClient, err := cache.NewRedisClient(redisConfig)
	if err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()
	slog.Info("connected to Redis")

	// Transaction Manager
	txManager := database.NewTxManager(pgClient.Pool())

	// JWT Service
	jwtConfig := jwt.Config{
		SecretKey:          config.JWTSecretKey,
		Issuer:             "gc-storage",
		Audience:           []string{"gc-storage-api"},
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	jwtService := jwt.NewJWTService(jwtConfig)

	// Cache Services
	sessionStore := cache.NewSessionStore(redisClient.Client(), 7*24*time.Hour)
	jwtBlacklist := cache.NewJWTBlacklist(redisClient.Client())
	rateLimiter := cache.NewRateLimiter(redisClient.Client())

	// Email Service
	smtpClient := email.NewSMTPClient(email.DefaultConfig())
	emailService := email.NewEmailService(smtpClient)

	// OAuth Client Factory
	oauthConfig := oauth.Config{
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", config.AppURL+"/auth/callback/google"),
		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		GitHubRedirectURL:  getEnv("GITHUB_REDIRECT_URL", config.AppURL+"/auth/callback/github"),
	}
	oauthFactory := oauth.NewClientFactory(oauthConfig)

	// Repositories
	userRepo := infraRepo.NewUserRepository(txManager)
	emailVerificationTokenRepo := infraRepo.NewEmailVerificationTokenRepository(txManager)
	passwordResetTokenRepo := infraRepo.NewPasswordResetTokenRepository(txManager)
	oauthAccountRepo := infraRepo.NewOAuthAccountRepository(txManager)

	// Commands
	registerCommand := authcmd.NewRegisterCommand(userRepo, emailVerificationTokenRepo, txManager, emailService, config.AppURL)
	loginCommand := authcmd.NewLoginCommand(userRepo, sessionStore, jwtService)
	refreshTokenCommand := authcmd.NewRefreshTokenCommand(userRepo, sessionStore, jwtService, jwtBlacklist)
	logoutCommand := authcmd.NewLogoutCommand(sessionStore, jwtBlacklist)
	verifyEmailCommand := authcmd.NewVerifyEmailCommand(userRepo, emailVerificationTokenRepo, txManager)
	resendEmailVerificationCommand := authcmd.NewResendEmailVerificationCommand(userRepo, emailVerificationTokenRepo, emailService, config.AppURL)
	forgotPasswordCommand := authcmd.NewForgotPasswordCommand(userRepo, passwordResetTokenRepo, emailService, config.AppURL)
	resetPasswordCommand := authcmd.NewResetPasswordCommand(userRepo, passwordResetTokenRepo, txManager)
	changePasswordCommand := authcmd.NewChangePasswordCommand(userRepo)
	oauthLoginCommand := authcmd.NewOAuthLoginCommand(userRepo, oauthAccountRepo, oauthFactory, txManager, sessionStore, jwtService)

	// Queries
	getUserQuery := authqry.NewGetUserQuery(userRepo)

	// Handlers
	healthHandler := handler.NewHealthHandler()
	healthHandler.RegisterChecker("postgres", pgClient)
	healthHandler.RegisterChecker("redis", redisClient)

	authHandler := handler.NewAuthHandler(
		registerCommand,
		loginCommand,
		refreshTokenCommand,
		logoutCommand,
		verifyEmailCommand,
		resendEmailVerificationCommand,
		forgotPasswordCommand,
		resetPasswordCommand,
		changePasswordCommand,
		oauthLoginCommand,
		getUserQuery,
	)

	// Middleware
	jwtAuthMiddleware := middleware.NewJWTAuthMiddleware(jwtService, jwtBlacklist)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimiter)

	// Server
	serverConfig := server.DefaultConfig()
	serverConfig.Port = config.ServerPort
	serverConfig.Debug = config.Debug
	srv := server.NewServer(serverConfig)

	e := srv.Echo()

	// Setup validator
	e.Validator = validator.NewCustomValidator()

	// Setup error handler
	e.HTTPErrorHandler = middleware.CustomHTTPErrorHandler

	// Global middleware
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.SecurityHeaders())
	e.Use(middleware.CORS())

	// Routes
	setupRoutes(e, healthHandler, authHandler, jwtAuthMiddleware, rateLimitMiddleware)

	// Start server
	slog.Info("starting server", "port", config.ServerPort)
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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
	slog.Info("server stopped")
}

type Config struct {
	ServerPort   int
	Debug        bool
	DatabaseURL  string
	RedisURL     string
	JWTSecretKey string
	AppURL       string
}

func loadConfig() Config {
	port := 8080
	if p := os.Getenv("SERVER_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}

	return Config{
		ServerPort:   port,
		Debug:        os.Getenv("DEBUG") == "true",
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/gc_storage?sslmode=disable"),
		RedisURL:     getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecretKey: getEnv("JWT_SECRET_KEY", "your-secret-key-change-in-production"),
		AppURL:       getEnv("APP_URL", "http://localhost:3000"),
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func setupRoutes(
	e *echo.Echo,
	healthHandler *handler.HealthHandler,
	authHandler *handler.AuthHandler,
	jwtAuthMiddleware *middleware.JWTAuthMiddleware,
	rateLimitMiddleware *middleware.RateLimitMiddleware,
) {
	// Health check
	e.GET("/health", healthHandler.Check)
	e.GET("/ready", healthHandler.Ready)

	// API v1
	api := e.Group("/api/v1")

	// Auth routes (public)
	authGroup := api.Group("/auth")
	authGroup.POST("/register", authHandler.Register,
		rateLimitMiddleware.ByIP(middleware.RateLimitAuthSignup))
	authGroup.POST("/login", authHandler.Login,
		rateLimitMiddleware.ByIP(middleware.RateLimitAuthLogin))
	authGroup.POST("/refresh", authHandler.Refresh)

	// OAuth routes (public)
	authGroup.POST("/oauth/:provider", authHandler.OAuthLogin,
		rateLimitMiddleware.ByIP(middleware.RateLimitAuthLogin))

	// Email verification routes (public)
	emailGroup := authGroup.Group("/email")
	emailGroup.POST("/verify", authHandler.VerifyEmail)
	emailGroup.POST("/resend", authHandler.ResendEmailVerification,
		rateLimitMiddleware.ByIP(middleware.RateLimitAuthSignup))

	// Password reset routes (public)
	passwordGroup := authGroup.Group("/password")
	passwordGroup.POST("/forgot", authHandler.ForgotPassword,
		rateLimitMiddleware.ByIP(middleware.RateLimitAuthLogin))
	passwordGroup.POST("/reset", authHandler.ResetPassword,
		rateLimitMiddleware.ByIP(middleware.RateLimitAuthLogin))

	// Password change route (authenticated)
	passwordGroup.POST("/change", authHandler.ChangePassword, jwtAuthMiddleware.Authenticate())

	// Auth routes (authenticated)
	authGroup.POST("/logout", authHandler.Logout, jwtAuthMiddleware.Authenticate())

	// User routes (authenticated)
	api.GET("/me", authHandler.Me, jwtAuthMiddleware.Authenticate())

	// Debug route
	api.GET("/", func(c echo.Context) error {
		return presenter.OK(c, map[string]string{
			"message": "GC Storage API v1",
		})
	})
}
