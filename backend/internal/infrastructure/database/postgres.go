package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBConfig はデータベース接続プールの設定
type DBConfig struct {
	MaxConns          int32         // 最大接続数
	MinConns          int32         // 最小接続数
	MaxConnLifetime   time.Duration // 接続の最大生存時間
	MaxConnIdleTime   time.Duration // アイドル接続の最大時間
	HealthCheckPeriod time.Duration // ヘルスチェック間隔
}

// DefaultDBConfig はデフォルトのDB設定を返す
func DefaultDBConfig() DBConfig {
	return DBConfig{
		MaxConns:          25,
		MinConns:          5,
		MaxConnLifetime:   1 * time.Hour,
		MaxConnIdleTime:   30 * time.Minute,
		HealthCheckPeriod: 1 * time.Minute,
	}
}

// PostgresClient はPostgreSQLへの接続を管理する
type PostgresClient struct {
	pool *pgxpool.Pool
}

// NewPostgresClient は新しいPostgresClientを作成する
func NewPostgresClient(ctx context.Context, databaseURL string) (*PostgresClient, error) {
	return NewPostgresClientWithConfig(ctx, databaseURL, DefaultDBConfig())
}

// NewPostgresClientWithConfig は設定を指定してPostgresClientを作成する
func NewPostgresClientWithConfig(ctx context.Context, databaseURL string, cfg DBConfig) (*PostgresClient, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// プール設定
	config.MaxConns = cfg.MaxConns
	config.MinConns = cfg.MinConns
	config.MaxConnLifetime = cfg.MaxConnLifetime
	config.MaxConnIdleTime = cfg.MaxConnIdleTime
	config.HealthCheckPeriod = cfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// 接続確認
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresClient{pool: pool}, nil
}

// Pool はコネクションプールを返す
func (c *PostgresClient) Pool() *pgxpool.Pool {
	return c.pool
}

// Close はコネクションプールを閉じる
func (c *PostgresClient) Close() {
	c.pool.Close()
}

// Health はデータベースのヘルスチェックを行う
func (c *PostgresClient) Health(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

// Stats はコネクションプールの統計情報を返す
func (c *PostgresClient) Stats() *pgxpool.Stat {
	return c.pool.Stat()
}
