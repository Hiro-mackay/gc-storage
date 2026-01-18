package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimitResult はレート制限チェックの結果を表します
type RateLimitResult struct {
	Allowed   bool      // リクエストが許可されたか
	Remaining int       // 残りリクエスト数
	ResetAt   time.Time // リセット時刻
	RetryAt   time.Time // リトライ可能時刻（拒否された場合）
}

// RateLimitConfig はレート制限の設定を定義します
type RateLimitConfig struct {
	Type     string        // 制限タイプ（auth, api, upload等）
	Requests int           // ウィンドウ内の最大リクエスト数
	Window   time.Duration // ウィンドウサイズ
}

// 事前定義されたレート制限設定
var (
	RateLimitAuthLogin = RateLimitConfig{
		Type:     "auth:login",
		Requests: 10,
		Window:   time.Minute,
	}
	RateLimitAuthSignup = RateLimitConfig{
		Type:     "auth:signup",
		Requests: 5,
		Window:   time.Minute,
	}
	RateLimitAPIDefault = RateLimitConfig{
		Type:     "api:default",
		Requests: 1000,
		Window:   time.Minute,
	}
	RateLimitAPIUpload = RateLimitConfig{
		Type:     "api:upload",
		Requests: 100,
		Window:   time.Minute,
	}
	RateLimitAPIDownload = RateLimitConfig{
		Type:     "api:download",
		Requests: 500,
		Window:   time.Minute,
	}
	RateLimitAPISearch = RateLimitConfig{
		Type:     "api:search",
		Requests: 30,
		Window:   time.Minute,
	}
)

// RateLimiter はレート制限を提供します
type RateLimiter struct {
	client *redis.Client
}

// NewRateLimiter は新しいRateLimiterを作成します
func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// Sliding Window Counter アルゴリズムを使用したレート制限
// Luaスクリプトでアトミックに処理
var slidingWindowScript = redis.NewScript(`
    local key = KEYS[1]
    local now = tonumber(ARGV[1])
    local window = tonumber(ARGV[2])
    local limit = tonumber(ARGV[3])

    -- 古いエントリを削除
    redis.call('ZREMRANGEBYSCORE', key, 0, now - window * 1000)

    -- 現在のカウントを取得
    local count = redis.call('ZCARD', key)

    if count < limit then
        -- リクエストを記録
        redis.call('ZADD', key, now, now .. ':' .. math.random())
        redis.call('PEXPIRE', key, window * 1000)
        return {1, limit - count - 1, now + window * 1000}
    else
        -- 最も古いエントリの時刻を取得
        local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
        local retry_at = oldest[2] + window * 1000
        return {0, 0, retry_at}
    end
`)

// Allow はリクエストが許可されるかチェックします（Sliding Window Counter）
func (r *RateLimiter) Allow(ctx context.Context, identifier string, config RateLimitConfig) (*RateLimitResult, error) {
	key := fmt.Sprintf("%s:%s:%s", PrefixRateLimit, config.Type, identifier)
	now := time.Now().UnixMilli()
	windowMs := config.Window.Milliseconds()

	result, err := slidingWindowScript.Run(ctx, r.client, []string{key}, now, windowMs/1000, config.Requests).Slice()
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}

	allowed := result[0].(int64) == 1
	remaining := int(result[1].(int64))
	resetAtMs := result[2].(int64)

	return &RateLimitResult{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   time.UnixMilli(resetAtMs),
		RetryAt:   time.UnixMilli(resetAtMs),
	}, nil
}

// Fixed Window Counter（シンプル版）
var fixedWindowScript = redis.NewScript(`
    local key = KEYS[1]
    local limit = tonumber(ARGV[1])
    local window = tonumber(ARGV[2])

    local current = redis.call('INCR', key)
    if current == 1 then
        redis.call('EXPIRE', key, window)
    end

    if current <= limit then
        return {1, limit - current}
    else
        local ttl = redis.call('TTL', key)
        return {0, ttl}
    end
`)

// AllowFixedWindow はリクエストが許可されるかチェックします（Fixed Window）
func (r *RateLimiter) AllowFixedWindow(ctx context.Context, identifier string, config RateLimitConfig) (*RateLimitResult, error) {
	now := time.Now()
	windowStart := now.Truncate(config.Window)
	key := RateLimitKey(config.Type, identifier, windowStart.Unix())

	result, err := fixedWindowScript.Run(ctx, r.client, []string{key}, config.Requests, int(config.Window.Seconds())).Slice()
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}

	allowed := result[0].(int64) == 1
	secondValue := result[1].(int64)

	if allowed {
		return &RateLimitResult{
			Allowed:   true,
			Remaining: int(secondValue),
			ResetAt:   windowStart.Add(config.Window),
		}, nil
	}

	return &RateLimitResult{
		Allowed:   false,
		Remaining: 0,
		ResetAt:   windowStart.Add(config.Window),
		RetryAt:   now.Add(time.Duration(secondValue) * time.Second),
	}, nil
}

// Reset はレート制限をリセットします
func (r *RateLimiter) Reset(ctx context.Context, identifier string, config RateLimitConfig) error {
	// Sliding Window用のキーパターンで削除
	pattern := fmt.Sprintf("%s:%s:%s*", PrefixRateLimit, config.Type, identifier)

	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys: %w", err)
		}

		if len(keys) > 0 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
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
