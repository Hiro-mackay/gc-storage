// Package testutil provides utilities for integration testing
package testutil

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/cache"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
)

var (
	testDBPool     *pgxpool.Pool
	testRedis      *redis.Client
	setupOnce      sync.Once
	teardownOnce   sync.Once
)

// TestConfig holds test environment configuration
type TestConfig struct {
	DatabaseURL  string
	RedisURL     string
	JWTSecretKey string
}

// DefaultTestConfig returns default test configuration
func DefaultTestConfig() TestConfig {
	return TestConfig{
		DatabaseURL:  getEnv("TEST_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/gc_storage_test?sslmode=disable"),
		RedisURL:     getEnv("TEST_REDIS_URL", "redis://localhost:6379/1"),
		JWTSecretKey: "test-secret-key-for-integration-tests",
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// SetupTestEnvironment initializes the test database and Redis connections
func SetupTestEnvironment(t *testing.T) (*pgxpool.Pool, *redis.Client) {
	t.Helper()

	config := DefaultTestConfig()

	setupOnce.Do(func() {
		ctx := context.Background()

		// Setup PostgreSQL
		pool, err := pgxpool.New(ctx, config.DatabaseURL)
		if err != nil {
			log.Fatalf("Failed to connect to test database: %v", err)
		}

		// Verify connection
		if err := pool.Ping(ctx); err != nil {
			log.Fatalf("Failed to ping test database: %v", err)
		}

		testDBPool = pool

		// Setup Redis
		opt, err := redis.ParseURL(config.RedisURL)
		if err != nil {
			log.Fatalf("Failed to parse Redis URL: %v", err)
		}
		testRedis = redis.NewClient(opt)

		// Verify Redis connection
		if err := testRedis.Ping(ctx).Err(); err != nil {
			log.Fatalf("Failed to ping Redis: %v", err)
		}
	})

	return testDBPool, testRedis
}

// CleanupTestEnvironment closes test connections
func CleanupTestEnvironment() {
	teardownOnce.Do(func() {
		if testDBPool != nil {
			testDBPool.Close()
		}
		if testRedis != nil {
			testRedis.Close()
		}
	})
}

// TruncateTables clears specified tables for test isolation
func TruncateTables(t *testing.T, pool *pgxpool.Pool, tables ...string) {
	t.Helper()
	ctx := context.Background()

	for _, table := range tables {
		_, err := pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Fatalf("Failed to truncate table %s: %v", table, err)
		}
	}
}

// FlushRedis clears Redis test database
func FlushRedis(t *testing.T, client *redis.Client) {
	t.Helper()
	ctx := context.Background()

	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("Failed to flush Redis: %v", err)
	}
}

// NewTestTxManager creates a TxManager for testing
func NewTestTxManager(pool *pgxpool.Pool) *database.TxManager {
	return database.NewTxManager(pool)
}

// NewTestSessionStore creates a SessionStore for testing
func NewTestSessionStore(client *redis.Client) *cache.SessionStore {
	return cache.NewSessionStore(client, 7*24*time.Hour)
}

// NewTestJWTBlacklist creates a JWTBlacklist for testing
func NewTestJWTBlacklist(client *redis.Client) *cache.JWTBlacklist {
	return cache.NewJWTBlacklist(client)
}

// NewTestRateLimiter creates a RateLimiter for testing
func NewTestRateLimiter(client *redis.Client) *cache.RateLimiter {
	return cache.NewRateLimiter(client)
}
