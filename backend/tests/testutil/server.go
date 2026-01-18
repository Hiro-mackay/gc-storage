package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/cache"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/di"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/oauth"
	infraRepo "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/handler"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/router"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/validator"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/config"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/jwt"
)

// TestServer holds all test server dependencies
type TestServer struct {
	Echo             *echo.Echo
	Pool             *pgxpool.Pool
	Redis            *redis.Client
	Container        *di.Container
	TxManager        *database.TxManager
	JWTService       *jwt.JWTService
	SessionRepo      repository.SessionRepository
	JWTBlacklist     *cache.JWTBlacklist
	RateLimiter      *cache.RateLimiter
	UserRepo         *infraRepo.UserRepository
	OAuthAccountRepo *infraRepo.OAuthAccountRepository
	OAuthFactory     *oauth.ClientFactory
	MockGoogleClient *oauth.MockOAuthClient
	MockGitHubClient *oauth.MockOAuthClient
	AuthHandler      *handler.AuthHandler
}

// NewTestServer creates a fully configured test server
func NewTestServer(t *testing.T) *TestServer {
	t.Helper()

	testCfg := DefaultTestConfig()
	pool, redisClient := SetupTestEnvironment(t)

	// Create mock OAuth clients
	mockGoogleClient := oauth.NewMockOAuthClient(valueobject.OAuthProviderGoogle)
	mockGitHubClient := oauth.NewMockOAuthClient(valueobject.OAuthProviderGitHub)

	// Create OAuth factory with mock clients
	oauthFactory := oauth.NewClientFactory(oauth.Config{})
	oauthFactory.RegisterClient(valueobject.OAuthProviderGoogle, mockGoogleClient)
	oauthFactory.RegisterClient(valueobject.OAuthProviderGitHub, mockGitHubClient)

	// Create config for DI container
	cfg := &config.Config{
		JWT: config.JWTConfig{
			SecretKey:          testCfg.JWTSecretKey,
			Issuer:             "gc-storage-test",
			Audience:           []string{"gc-storage-api-test"},
			AccessTokenExpiry:  15 * time.Minute,
			RefreshTokenExpiry: 7 * 24 * time.Hour,
		},
		App: config.AppConfig{
			URL: "http://localhost:3000",
		},
	}

	// Create DI container with options
	container, err := di.NewContainerWithOptions(context.Background(), cfg, di.Options{
		PostgresPool: pool,
		RedisClient:  redisClient,
		EmailService: nil, // nil for tests (email sending is not needed)
		OAuthFactory: oauthFactory,
	})
	if err != nil {
		t.Fatalf("Failed to create DI container: %v", err)
	}

	// Initialize UseCases, Handlers, and Middlewares
	container.InitAuthUseCases()
	container.InitProfileUseCases()
	handlers := di.NewHandlersForTest(container)
	middlewares := di.NewMiddlewares(container)

	// Echo instance
	e := echo.New()
	e.Validator = validator.NewCustomValidator()
	e.HTTPErrorHandler = middleware.CustomHTTPErrorHandler

	// Setup routes
	router.NewRouter(e, handlers, middlewares).Setup()

	// Type assertions for repositories (they implement interfaces but we need concrete types)
	userRepo, _ := container.UserRepo.(*infraRepo.UserRepository)
	oauthAccountRepo, _ := container.OAuthAccountRepo.(*infraRepo.OAuthAccountRepository)

	return &TestServer{
		Echo:             e,
		Pool:             pool,
		Redis:            redisClient,
		Container:        container,
		TxManager:        container.TxManager,
		JWTService:       container.JWTService,
		SessionRepo:      container.SessionRepo,
		JWTBlacklist:     container.JWTBlacklist,
		RateLimiter:      container.RateLimiter,
		UserRepo:         userRepo,
		OAuthAccountRepo: oauthAccountRepo,
		OAuthFactory:     oauthFactory,
		MockGoogleClient: mockGoogleClient,
		MockGitHubClient: mockGitHubClient,
		AuthHandler:      handlers.Auth,
	}
}

// Cleanup cleans up test data
func (ts *TestServer) Cleanup(t *testing.T) {
	t.Helper()
	TruncateTables(t, ts.Pool, "sessions", "oauth_accounts", "email_verification_tokens", "password_reset_tokens", "user_profiles", "users")
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
