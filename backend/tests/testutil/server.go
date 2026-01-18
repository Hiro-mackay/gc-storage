package testutil

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/cache"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	infraRepo "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/handler"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/validator"
	authcmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/command"
	authqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/jwt"
)

// TestServer holds all test server dependencies
type TestServer struct {
	Echo         *echo.Echo
	Pool         *pgxpool.Pool
	Redis        *redis.Client
	TxManager    *database.TxManager
	JWTService   *jwt.JWTService
	SessionStore *cache.SessionStore
	JWTBlacklist *cache.JWTBlacklist
	RateLimiter  *cache.RateLimiter
	UserRepo     *infraRepo.UserRepository
	AuthHandler  *handler.AuthHandler
}

// NewTestServer creates a fully configured test server
func NewTestServer(t *testing.T) *TestServer {
	t.Helper()

	config := DefaultTestConfig()
	pool, redisClient := SetupTestEnvironment(t)

	// Transaction Manager
	txManager := database.NewTxManager(pool)

	// JWT Service
	jwtConfig := jwt.Config{
		SecretKey:          config.JWTSecretKey,
		Issuer:             "gc-storage-test",
		Audience:           []string{"gc-storage-api-test"},
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	jwtService := jwt.NewJWTService(jwtConfig)

	// Cache Services
	sessionStore := cache.NewSessionStore(redisClient, 7*24*time.Hour)
	jwtBlacklist := cache.NewJWTBlacklist(redisClient)
	rateLimiter := cache.NewRateLimiter(redisClient)

	// Repositories
	userRepo := infraRepo.NewUserRepository(txManager)

	// Commands
	registerCommand := authcmd.NewRegisterCommand(userRepo, nil, txManager)
	loginCommand := authcmd.NewLoginCommand(userRepo, sessionStore, jwtService)
	refreshTokenCommand := authcmd.NewRefreshTokenCommand(userRepo, sessionStore, jwtService, jwtBlacklist)
	logoutCommand := authcmd.NewLogoutCommand(sessionStore, jwtBlacklist)

	// Queries
	getUserQuery := authqry.NewGetUserQuery(userRepo)

	// Handlers
	authHandler := handler.NewAuthHandler(
		registerCommand,
		loginCommand,
		refreshTokenCommand,
		logoutCommand,
		getUserQuery,
	)

	// Middleware
	jwtAuthMiddleware := middleware.NewJWTAuthMiddleware(jwtService, jwtBlacklist)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimiter)

	// Echo instance
	e := echo.New()
	e.Validator = validator.NewCustomValidator()
	e.HTTPErrorHandler = middleware.CustomHTTPErrorHandler

	// Setup routes
	setupTestRoutes(e, authHandler, jwtAuthMiddleware, rateLimitMiddleware)

	return &TestServer{
		Echo:         e,
		Pool:         pool,
		Redis:        redisClient,
		TxManager:    txManager,
		JWTService:   jwtService,
		SessionStore: sessionStore,
		JWTBlacklist: jwtBlacklist,
		RateLimiter:  rateLimiter,
		UserRepo:     userRepo,
		AuthHandler:  authHandler,
	}
}

// setupTestRoutes configures routes for testing
func setupTestRoutes(
	e *echo.Echo,
	authHandler *handler.AuthHandler,
	jwtAuthMiddleware *middleware.JWTAuthMiddleware,
	rateLimitMiddleware *middleware.RateLimitMiddleware,
) {
	// API v1
	api := e.Group("/api/v1")

	// Auth routes (public)
	authGroup := api.Group("/auth")
	authGroup.POST("/register", authHandler.Register,
		rateLimitMiddleware.ByIP(middleware.RateLimitAuthSignup))
	authGroup.POST("/login", authHandler.Login,
		rateLimitMiddleware.ByIP(middleware.RateLimitAuthLogin))
	authGroup.POST("/refresh", authHandler.Refresh)

	// Auth routes (authenticated)
	authGroup.POST("/logout", authHandler.Logout, jwtAuthMiddleware.Authenticate())

	// User routes (authenticated)
	api.GET("/me", authHandler.Me, jwtAuthMiddleware.Authenticate())
}

// Cleanup cleans up test data
func (ts *TestServer) Cleanup(t *testing.T) {
	t.Helper()
	TruncateTables(t, ts.Pool, "sessions", "oauth_accounts", "users")
	FlushRedis(t, ts.Redis)
}
