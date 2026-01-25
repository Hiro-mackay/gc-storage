package di

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/cache"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/email"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/oauth"
	infraRepo "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/config"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/jwt"
)

// Container はアプリケーションの依存関係を保持するDIコンテナです
type Container struct {
	// Infrastructure
	PgClient    *database.PostgresClient
	RedisClient *cache.RedisClient
	TxManager   *database.TxManager

	// Services
	JWTService   *jwt.JWTService
	JWTBlacklist *cache.JWTBlacklist
	RateLimiter  *cache.RateLimiter
	EmailService service.EmailSender
	OAuthFactory service.OAuthClientFactory

	// Repositories
	UserRepo                   repository.UserRepository
	SessionRepo                repository.SessionRepository
	EmailVerificationTokenRepo repository.EmailVerificationTokenRepository
	PasswordResetTokenRepo     repository.PasswordResetTokenRepository
	OAuthAccountRepo           repository.OAuthAccountRepository
	UserProfileRepo            repository.UserProfileRepository

	// Auth UseCases
	Auth *AuthUseCases

	// Profile UseCases
	Profile *ProfileUseCases

	// Storage UseCases
	Storage *StorageUseCases

	// Storage Repositories (for tests and direct access)
	StorageRepos *StorageRepositories

	// Collaboration UseCases
	Collaboration *CollaborationUseCases

	// Collaboration Repositories
	CollabRepos *CollaborationRepositories

	// Authorization UseCases
	Authz *AuthzUseCases

	// Authorization Repositories
	AuthzRepos *AuthzRepositories

	// Permission Resolver
	PermissionResolver authz.PermissionResolver

	// Sharing UseCases
	Sharing *SharingUseCases

	// Sharing Repositories
	SharingRepos *SharingRepositories

	// config
	config *config.Config
}

// NewContainer は新しいContainerを作成します
func NewContainer(ctx context.Context, cfg *config.Config) (*Container, error) {
	return NewContainerWithOptions(ctx, cfg, Options{})
}

// NewContainerWithOptions はオプションを指定してContainerを作成します
func NewContainerWithOptions(ctx context.Context, cfg *config.Config, opts Options) (*Container, error) {
	c := &Container{
		config: cfg,
	}

	// PostgreSQL
	if opts.PostgresPool != nil {
		c.TxManager = database.NewTxManager(opts.PostgresPool)
	} else {
		slog.Info("connecting to PostgreSQL...")
		pgClient, err := database.NewPostgresClient(ctx, cfg.Database.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
		}
		c.PgClient = pgClient
		c.TxManager = database.NewTxManager(pgClient.Pool())
		slog.Info("connected to PostgreSQL")
	}

	// Redis
	if opts.RedisClient != nil {
		c.SessionRepo = cache.NewSessionStore(opts.RedisClient, 7*24*time.Hour)
		c.JWTBlacklist = cache.NewJWTBlacklist(opts.RedisClient)
		c.RateLimiter = cache.NewRateLimiter(opts.RedisClient)
	} else {
		slog.Info("connecting to Redis...")
		redisConfig := cache.DefaultConfig()
		redisConfig.URL = cfg.Redis.URL
		redisClient, err := cache.NewRedisClient(redisConfig)
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("failed to connect to Redis: %w", err)
		}
		c.RedisClient = redisClient
		c.SessionRepo = cache.NewSessionStore(redisClient.Client(), 7*24*time.Hour)
		c.JWTBlacklist = cache.NewJWTBlacklist(redisClient.Client())
		c.RateLimiter = cache.NewRateLimiter(redisClient.Client())
		slog.Info("connected to Redis")
	}

	// JWT Service
	jwtConfig := jwt.Config{
		SecretKey:          cfg.JWT.SecretKey,
		Issuer:             cfg.JWT.Issuer,
		Audience:           cfg.JWT.Audience,
		AccessTokenExpiry:  cfg.JWT.AccessTokenExpiry,
		RefreshTokenExpiry: cfg.JWT.RefreshTokenExpiry,
	}
	c.JWTService = jwt.NewJWTService(jwtConfig)

	// Email Service
	if opts.EmailService != nil {
		c.EmailService = opts.EmailService
	} else {
		smtpClient := email.NewSMTPClient(email.DefaultConfig())
		c.EmailService = email.NewEmailService(smtpClient)
	}

	// OAuth Factory
	if opts.OAuthFactory != nil {
		c.OAuthFactory = opts.OAuthFactory
	} else {
		oauthConfig := oauth.Config{
			GoogleClientID:     cfg.OAuth.GoogleClientID,
			GoogleClientSecret: cfg.OAuth.GoogleClientSecret,
			GoogleRedirectURL:  cfg.OAuth.GoogleRedirectURL,
			GitHubClientID:     cfg.OAuth.GitHubClientID,
			GitHubClientSecret: cfg.OAuth.GitHubClientSecret,
			GitHubRedirectURL:  cfg.OAuth.GitHubRedirectURL,
		}
		c.OAuthFactory = oauth.NewClientFactory(oauthConfig)
	}

	// Repositories
	c.UserRepo = infraRepo.NewUserRepository(c.TxManager)
	c.EmailVerificationTokenRepo = infraRepo.NewEmailVerificationTokenRepository(c.TxManager)
	c.PasswordResetTokenRepo = infraRepo.NewPasswordResetTokenRepository(c.TxManager)
	c.OAuthAccountRepo = infraRepo.NewOAuthAccountRepository(c.TxManager)
	c.UserProfileRepo = infraRepo.NewUserProfileRepository(c.TxManager)

	return c, nil
}

// InitAuthUseCases はAuth UseCasesを初期化します
func (c *Container) InitAuthUseCases() {
	c.Auth = NewAuthUseCases(c, c.config.App.URL)
}

// InitProfileUseCases はProfile UseCasesを初期化します
func (c *Container) InitProfileUseCases() {
	c.Profile = NewProfileUseCases(c)
}

// InitStorageUseCases はStorage UseCasesを初期化します
func (c *Container) InitStorageUseCases(storageService service.StorageService) {
	c.StorageRepos = NewStorageRepositories(c.TxManager)
	// AuthzRepos must be initialized before StorageUseCases for relationship management
	if c.AuthzRepos == nil {
		c.AuthzRepos = NewAuthzRepositories(c.TxManager)
	}
	// CollabRepos must be initialized for PermissionResolver
	if c.CollabRepos == nil {
		c.CollabRepos = NewCollaborationRepositories(c.TxManager)
	}
	// PermissionResolver must be initialized for StorageUseCases
	if c.PermissionResolver == nil {
		c.PermissionResolver = NewPermissionResolver(c.AuthzRepos, c.CollabRepos)
	}
	c.Storage = NewStorageUseCases(c.StorageRepos, c.AuthzRepos.RelationshipRepo, c.PermissionResolver, c.TxManager, storageService)
}

// InitCollaborationUseCases はCollaboration UseCasesを初期化します
func (c *Container) InitCollaborationUseCases() {
	c.CollabRepos = NewCollaborationRepositories(c.TxManager)
	c.Collaboration = NewCollaborationUseCases(c.CollabRepos, c.UserRepo, c.TxManager)
}

// InitAuthzUseCases はAuthorization UseCasesを初期化します
func (c *Container) InitAuthzUseCases() {
	c.AuthzRepos = NewAuthzRepositories(c.TxManager)
	c.PermissionResolver = NewPermissionResolver(c.AuthzRepos, c.CollabRepos)
	c.Authz = NewAuthzUseCases(c.AuthzRepos, c.PermissionResolver)
}

// InitSharingUseCases はSharing UseCasesを初期化します
func (c *Container) InitSharingUseCases() {
	c.SharingRepos = NewSharingRepositories(c.TxManager)
	c.Sharing = NewSharingUseCases(c.SharingRepos, c.PermissionResolver)
}

// Close はリソースをクリーンアップします
func (c *Container) Close() error {
	var errs []error

	if c.PgClient != nil {
		c.PgClient.Close()
	}

	if c.RedisClient != nil {
		if err := c.RedisClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Redis: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during close: %v", errs)
	}
	return nil
}

// Options はContainer作成時のオプションを定義します
type Options struct {
	PostgresPool *pgxpool.Pool
	RedisClient  *redis.Client
	EmailService service.EmailSender
	OAuthFactory service.OAuthClientFactory
}
