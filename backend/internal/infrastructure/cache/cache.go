package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrCacheMiss はキャッシュミスを表すエラーです
var ErrCacheMiss = errors.New("cache miss")

// Cache は汎用キャッシュを提供します
type Cache struct {
	client     *redis.Client
	namespace  string
	defaultTTL time.Duration
}

// NewCache は新しいCacheを作成します
func NewCache(client *redis.Client, namespace string, defaultTTL time.Duration) *Cache {
	return &Cache{
		client:     client,
		namespace:  namespace,
		defaultTTL: defaultTTL,
	}
}

// Get はキャッシュから値を取得します
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
	cacheKey := CacheKey(c.namespace, key)

	data, err := c.client.Get(ctx, cacheKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheMiss
		}
		return fmt.Errorf("failed to get from cache: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal cache data: %w", err)
	}

	return nil
}

// Set はキャッシュに値を設定します
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl ...time.Duration) error {
	cacheKey := CacheKey(c.namespace, key)

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	expiry := c.defaultTTL
	if len(ttl) > 0 {
		expiry = ttl[0]
	}

	if err := c.client.Set(ctx, cacheKey, data, expiry).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// Delete はキャッシュから値を削除します
func (c *Cache) Delete(ctx context.Context, key string) error {
	cacheKey := CacheKey(c.namespace, key)
	return c.client.Del(ctx, cacheKey).Err()
}

// DeletePattern はパターンに一致するキャッシュを削除します
func (c *Cache) DeletePattern(ctx context.Context, pattern string) error {
	fullPattern := CacheKey(c.namespace, pattern)

	var cursor uint64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, fullPattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys: %w", err)
		}

		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("failed to delete keys: %w", err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// GetOrSet はキャッシュがなければ関数を実行して結果をキャッシュします
func (c *Cache) GetOrSet(ctx context.Context, key string, dest interface{}, ttl time.Duration, fn func() (interface{}, error)) error {
	// キャッシュを試行
	if err := c.Get(ctx, key, dest); err == nil {
		return nil
	}

	// キャッシュミス時は関数を実行
	value, err := fn()
	if err != nil {
		return err
	}

	// 結果をキャッシュ（エラーは無視）
	_ = c.Set(ctx, key, value, ttl)

	// destに値を設定
	data, _ := json.Marshal(value)
	return json.Unmarshal(data, dest)
}

// Exists はキーが存在するか確認します
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	cacheKey := CacheKey(c.namespace, key)
	result, err := c.client.Exists(ctx, cacheKey).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// TTL はキーの残り生存時間を取得します
func (c *Cache) TTL(ctx context.Context, key string) (time.Duration, error) {
	cacheKey := CacheKey(c.namespace, key)
	return c.client.TTL(ctx, cacheKey).Result()
}
