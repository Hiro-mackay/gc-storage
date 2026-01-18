package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config はRedis接続設定を定義します
type Config struct {
	URL             string        // redis://[:password@]host:port/db
	MaxRetries      int           // 最大リトライ回数
	MinIdleConns    int           // 最小アイドル接続数
	MaxActiveConns  int           // 最大アクティブ接続数
	ConnMaxIdleTime time.Duration // アイドル接続の最大生存時間
	ConnMaxLifetime time.Duration // 接続の最大生存時間
	DialTimeout     time.Duration // 接続タイムアウト
	ReadTimeout     time.Duration // 読み取りタイムアウト
	WriteTimeout    time.Duration // 書き込みタイムアウト
}

// DefaultConfig はデフォルト設定を返します
func DefaultConfig() Config {
	return Config{
		MaxRetries:      3,
		MinIdleConns:    10,
		MaxActiveConns:  100,
		ConnMaxIdleTime: 5 * time.Minute,
		ConnMaxLifetime: 30 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
	}
}

// RedisClient はRedis操作を提供します
type RedisClient struct {
	client *redis.Client
	config Config
}

// NewRedisClient は新しいRedisClientを作成します
func NewRedisClient(cfg Config) (*RedisClient, error) {
	opt, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	// 設定をオーバーライド
	opt.MaxRetries = cfg.MaxRetries
	opt.MinIdleConns = cfg.MinIdleConns
	opt.MaxActiveConns = cfg.MaxActiveConns
	opt.ConnMaxIdleTime = cfg.ConnMaxIdleTime
	opt.ConnMaxLifetime = cfg.ConnMaxLifetime
	opt.DialTimeout = cfg.DialTimeout
	opt.ReadTimeout = cfg.ReadTimeout
	opt.WriteTimeout = cfg.WriteTimeout

	client := redis.NewClient(opt)

	// 接続確認
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &RedisClient{
		client: client,
		config: cfg,
	}, nil
}

// Client は内部のredis.Clientを返します
func (r *RedisClient) Client() *redis.Client {
	return r.client
}

// Close はRedis接続を閉じます
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Health はRedisの接続状態を確認します
func (r *RedisClient) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
