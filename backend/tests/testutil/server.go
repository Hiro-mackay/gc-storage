package testutil

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/cache"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/oauth"
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
	Echo              *echo.Echo
	Pool              *pgxpool.Pool
	Redis             *redis.Client
	TxManager         *database.TxManager
	JWTService        *jwt.JWTService
	SessionStore      *cache.SessionStore
	JWTBlacklist      *cache.JWTBlacklist
	RateLimiter       *cache.RateLimiter
	UserRepo          *infraRepo.UserRepository
	OAuthAccountRepo  *infraRepo.OAuthAccountRepository
	OAuthFactory      *oauth.ClientFactory
	MockGoogleClient  *oauth.MockOAuthClient
	MockGitHubClient  *oauth.MockOAuthClient
	AuthHandler       *handler.AuthHandler
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

	// Mock OAuth Clients
	mockGoogleClient := oauth.NewMockOAuthClient(valueobject.OAuthProviderGoogle)
	mockGitHubClient := oauth.NewMockOAuthClient(valueobject.OAuthProviderGitHub)

	// OAuth Factory with mock clients
	oauthFactory := oauth.NewClientFactory(oauth.Config{})
	oauthFactory.RegisterClient(valueobject.OAuthProviderGoogle, mockGoogleClient)
	oauthFactory.RegisterClient(valueobject.OAuthProviderGitHub, mockGitHubClient)

	// Repositories
	userRepo := infraRepo.NewUserRepository(txManager)
	oauthAccountRepo := infraRepo.NewOAuthAccountRepository(txManager)

	// Repositories (Email Verification)
	emailVerificationTokenRepo := infraRepo.NewEmailVerificationTokenRepository(txManager)

	// Repositories (Password Reset)
	passwordResetTokenRepo := infraRepo.NewPasswordResetTokenRepository(txManager)

	// Commands
	registerCommand := authcmd.NewRegisterCommand(userRepo, emailVerificationTokenRepo, txManager, nil, "http://localhost:3000")
	loginCommand := authcmd.NewLoginCommand(userRepo, sessionStore, jwtService)
	refreshTokenCommand := authcmd.NewRefreshTokenCommand(userRepo, sessionStore, jwtService, jwtBlacklist)
	logoutCommand := authcmd.NewLogoutCommand(sessionStore, jwtBlacklist)
	verifyEmailCommand := authcmd.NewVerifyEmailCommand(userRepo, emailVerificationTokenRepo, txManager)
	resendEmailVerificationCommand := authcmd.NewResendEmailVerificationCommand(userRepo, emailVerificationTokenRepo, nil, "http://localhost:3000")
	forgotPasswordCommand := authcmd.NewForgotPasswordCommand(userRepo, passwordResetTokenRepo, nil, "http://localhost:3000")
	resetPasswordCommand := authcmd.NewResetPasswordCommand(userRepo, passwordResetTokenRepo, txManager)
	changePasswordCommand := authcmd.NewChangePasswordCommand(userRepo)
	oauthLoginCommand := authcmd.NewOAuthLoginCommand(userRepo, oauthAccountRepo, oauthFactory, txManager, sessionStore, jwtService)

	// Queries
	getUserQuery := authqry.NewGetUserQuery(userRepo)

	// Handlers
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

	// Echo instance
	e := echo.New()
	e.Validator = validator.NewCustomValidator()
	e.HTTPErrorHandler = middleware.CustomHTTPErrorHandler

	// Setup routes
	setupTestRoutes(e, authHandler, jwtAuthMiddleware, rateLimitMiddleware)

	return &TestServer{
		Echo:              e,
		Pool:              pool,
		Redis:             redisClient,
		TxManager:         txManager,
		JWTService:        jwtService,
		SessionStore:      sessionStore,
		JWTBlacklist:      jwtBlacklist,
		RateLimiter:       rateLimiter,
		UserRepo:          userRepo,
		OAuthAccountRepo:  oauthAccountRepo,
		OAuthFactory:      oauthFactory,
		MockGoogleClient:  mockGoogleClient,
		MockGitHubClient:  mockGitHubClient,
		AuthHandler:       authHandler,
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
}

// Cleanup cleans up test data
func (ts *TestServer) Cleanup(t *testing.T) {
	t.Helper()
	TruncateTables(t, ts.Pool, "sessions", "oauth_accounts", "email_verification_tokens", "password_reset_tokens", "users")
	FlushRedis(t, ts.Redis)

	// Reset mock OAuth clients to default state
	ts.MockGoogleClient.SetUserInfo(&service.OAuthUserInfo{
		ProviderUserID: "google-user-123",
		Email:          "oauth-google@example.com",
		Name:           "Google User",
		AvatarURL:      "https://example.com/google-avatar.png",
	})
	ts.MockGoogleClient.SetExchangeError(nil)
	ts.MockGoogleClient.SetUserInfoError(nil)

	ts.MockGitHubClient.SetUserInfo(&service.OAuthUserInfo{
		ProviderUserID: "github-user-456",
		Email:          "oauth-github@example.com",
		Name:           "GitHub User",
		AvatarURL:      "https://example.com/github-avatar.png",
	})
	ts.MockGitHubClient.SetExchangeError(nil)
	ts.MockGitHubClient.SetUserInfoError(nil)
}
