# Redis インフラストラクチャ仕様書

## 概要

本ドキュメントでは、GC StorageにおけるRedis接続管理、セッションストア、レートリミッター、キャッシュの実装仕様を定義します。

**関連アーキテクチャ:**
- [BACKEND.md](../02-architecture/BACKEND.md) - Clean Architecture設計
- [SECURITY.md](../02-architecture/SECURITY.md) - 認証・レート制限設計

---

## 1. Redis接続管理

### 1.1 クライアント構成

```go
// backend/internal/infrastructure/persistence/redis/client.go

package redis

import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

// Config はRedis接続設定を定義します
type Config struct {
    URL             string        // redis://[:password@]host:port/db
    MaxRetries      int           // 最大リトライ回数（デフォルト: 3）
    MinIdleConns    int           // 最小アイドル接続数（デフォルト: 10）
    MaxActiveConns  int           // 最大アクティブ接続数（デフォルト: 100）
    ConnMaxIdleTime time.Duration // アイドル接続の最大生存時間（デフォルト: 5分）
    ConnMaxLifetime time.Duration // 接続の最大生存時間（デフォルト: 30分）
    DialTimeout     time.Duration // 接続タイムアウト（デフォルト: 5秒）
    ReadTimeout     time.Duration // 読み取りタイムアウト（デフォルト: 3秒）
    WriteTimeout    time.Duration // 書き込みタイムアウト（デフォルト: 3秒）
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

// HealthCheck はRedisの接続状態を確認します
func (r *RedisClient) HealthCheck(ctx context.Context) error {
    return r.client.Ping(ctx).Err()
}
```

### 1.2 環境変数設定

```bash
# .env.local
REDIS_URL=redis://localhost:6379/0

# 本番環境 (.env.sample)
REDIS_URL=redis://:password@redis-host:6379/0
```

### 1.3 ディレクトリ構成

```
backend/internal/infrastructure/persistence/redis/
├── client.go           # Redis接続管理
├── session_store.go    # セッションストア
├── rate_limiter.go     # レートリミッター
├── cache.go            # 汎用キャッシュ
├── jwt_blacklist.go    # JWTブラックリスト
└── keys.go             # キー命名規則
```

---

## 2. キー命名規則

### 2.1 キープレフィックス定義

```go
// backend/internal/infrastructure/persistence/redis/keys.go

package redis

import (
    "fmt"

    "github.com/google/uuid"
)

// KeyPrefix はRedisキーのプレフィックスを定義します
type KeyPrefix string

const (
    // セッション関連
    PrefixSession      KeyPrefix = "session"       // session:{session_id}
    PrefixUserSessions KeyPrefix = "user:sessions" // user:sessions:{user_id}

    // JWT関連
    PrefixJWTBlacklist KeyPrefix = "jwt:blacklist" // jwt:blacklist:{jti}

    // レート制限
    PrefixRateLimit KeyPrefix = "ratelimit" // ratelimit:{type}:{identifier}:{window}

    // キャッシュ
    PrefixCache     KeyPrefix = "cache"     // cache:{namespace}:{key}
    PrefixUserCache KeyPrefix = "cache:user" // cache:user:{user_id}
)

// SessionKey はセッションキーを生成します
func SessionKey(sessionID string) string {
    return fmt.Sprintf("%s:%s", PrefixSession, sessionID)
}

// UserSessionsKey はユーザーのセッション一覧キーを生成します
func UserSessionsKey(userID uuid.UUID) string {
    return fmt.Sprintf("%s:%s", PrefixUserSessions, userID.String())
}

// JWTBlacklistKey はJWTブラックリストキーを生成します
func JWTBlacklistKey(jti string) string {
    return fmt.Sprintf("%s:%s", PrefixJWTBlacklist, jti)
}

// RateLimitKey はレート制限キーを生成します
func RateLimitKey(limitType, identifier string, windowStart int64) string {
    return fmt.Sprintf("%s:%s:%s:%d", PrefixRateLimit, limitType, identifier, windowStart)
}

// CacheKey は汎用キャッシュキーを生成します
func CacheKey(namespace, key string) string {
    return fmt.Sprintf("%s:%s:%s", PrefixCache, namespace, key)
}

// UserCacheKey はユーザーキャッシュキーを生成します
func UserCacheKey(userID uuid.UUID, key string) string {
    return fmt.Sprintf("%s:%s:%s", PrefixUserCache, userID.String(), key)
}
```

### 2.2 キー命名ガイドライン

| パターン | 形式 | 例 |
|---------|------|-----|
| セッション | `session:{session_id}` | `session:abc123` |
| ユーザーセッション | `user:sessions:{user_id}` | `user:sessions:550e8400-...` |
| JWTブラックリスト | `jwt:blacklist:{jti}` | `jwt:blacklist:def456` |
| レート制限 | `ratelimit:{type}:{id}:{window}` | `ratelimit:api:user123:1705311300` |
| キャッシュ | `cache:{namespace}:{key}` | `cache:files:metadata:abc` |

---

## 3. セッションストア

### 3.1 セッションデータ構造

```go
// backend/internal/domain/entity/session.go（参考）

type Session struct {
    ID           string    `json:"id"`
    UserID       uuid.UUID `json:"user_id"`
    RefreshToken string    `json:"refresh_token"`
    UserAgent    string    `json:"user_agent"`
    IPAddress    string    `json:"ip_address"`
    CreatedAt    time.Time `json:"created_at"`
    ExpiresAt    time.Time `json:"expires_at"`
    LastUsedAt   time.Time `json:"last_used_at"`
}
```

### 3.2 セッションストア実装

```go
// backend/internal/infrastructure/persistence/redis/session_store.go

package redis

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
)

// SessionStore はセッションの永続化を提供します
type SessionStore struct {
    client *redis.Client
    ttl    time.Duration // デフォルト: 7日（リフレッシュトークン有効期限）
}

// NewSessionStore は新しいSessionStoreを作成します
func NewSessionStore(client *redis.Client, ttl time.Duration) *SessionStore {
    return &SessionStore{
        client: client,
        ttl:    ttl,
    }
}

// Save はセッションを保存します
func (s *SessionStore) Save(ctx context.Context, session *entity.Session) error {
    data, err := json.Marshal(session)
    if err != nil {
        return fmt.Errorf("failed to marshal session: %w", err)
    }

    key := SessionKey(session.ID)
    ttl := time.Until(session.ExpiresAt)
    if ttl <= 0 {
        ttl = s.ttl
    }

    // パイプラインで複数操作をアトミックに実行
    pipe := s.client.TxPipeline()

    // セッションデータを保存
    pipe.Set(ctx, key, data, ttl)

    // ユーザーのセッション一覧に追加（TTLなしのSet）
    userSessionsKey := UserSessionsKey(session.UserID)
    pipe.SAdd(ctx, userSessionsKey, session.ID)
    pipe.Expire(ctx, userSessionsKey, 30*24*time.Hour) // 30日

    _, err = pipe.Exec(ctx)
    if err != nil {
        return fmt.Errorf("failed to save session: %w", err)
    }

    return nil
}

// FindByID はセッションIDでセッションを取得します
func (s *SessionStore) FindByID(ctx context.Context, sessionID string) (*entity.Session, error) {
    key := SessionKey(sessionID)

    data, err := s.client.Get(ctx, key).Bytes()
    if err != nil {
        if err == redis.Nil {
            return nil, repository.ErrSessionNotFound
        }
        return nil, fmt.Errorf("failed to get session: %w", err)
    }

    var session entity.Session
    if err := json.Unmarshal(data, &session); err != nil {
        return nil, fmt.Errorf("failed to unmarshal session: %w", err)
    }

    return &session, nil
}

// FindByUserID はユーザーIDで全セッションを取得します
func (s *SessionStore) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Session, error) {
    userSessionsKey := UserSessionsKey(userID)

    sessionIDs, err := s.client.SMembers(ctx, userSessionsKey).Result()
    if err != nil {
        return nil, fmt.Errorf("failed to get user sessions: %w", err)
    }

    if len(sessionIDs) == 0 {
        return []*entity.Session{}, nil
    }

    // パイプラインで一括取得
    pipe := s.client.Pipeline()
    cmds := make([]*redis.StringCmd, len(sessionIDs))
    for i, sessionID := range sessionIDs {
        cmds[i] = pipe.Get(ctx, SessionKey(sessionID))
    }

    _, err = pipe.Exec(ctx)
    if err != nil && err != redis.Nil {
        return nil, fmt.Errorf("failed to get sessions: %w", err)
    }

    sessions := make([]*entity.Session, 0, len(sessionIDs))
    expiredIDs := make([]string, 0)

    for i, cmd := range cmds {
        data, err := cmd.Bytes()
        if err != nil {
            if err == redis.Nil {
                // 期限切れセッションをマーク
                expiredIDs = append(expiredIDs, sessionIDs[i])
                continue
            }
            continue
        }

        var session entity.Session
        if err := json.Unmarshal(data, &session); err != nil {
            continue
        }
        sessions = append(sessions, &session)
    }

    // 期限切れセッションをユーザーセッション一覧から削除
    if len(expiredIDs) > 0 {
        go func() {
            bgCtx := context.Background()
            s.client.SRem(bgCtx, userSessionsKey, expiredIDs)
        }()
    }

    return sessions, nil
}

// UpdateLastUsed はセッションの最終使用時刻を更新します
func (s *SessionStore) UpdateLastUsed(ctx context.Context, sessionID string) error {
    session, err := s.FindByID(ctx, sessionID)
    if err != nil {
        return err
    }

    session.LastUsedAt = time.Now()
    return s.Save(ctx, session)
}

// Delete はセッションを削除します
func (s *SessionStore) Delete(ctx context.Context, sessionID string) error {
    session, err := s.FindByID(ctx, sessionID)
    if err != nil {
        if err == repository.ErrSessionNotFound {
            return nil // すでに削除済み
        }
        return err
    }

    pipe := s.client.TxPipeline()
    pipe.Del(ctx, SessionKey(sessionID))
    pipe.SRem(ctx, UserSessionsKey(session.UserID), sessionID)

    _, err = pipe.Exec(ctx)
    return err
}

// DeleteByUserID はユーザーの全セッションを削除します（ログアウトオール）
func (s *SessionStore) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
    userSessionsKey := UserSessionsKey(userID)

    sessionIDs, err := s.client.SMembers(ctx, userSessionsKey).Result()
    if err != nil {
        return fmt.Errorf("failed to get user sessions: %w", err)
    }

    if len(sessionIDs) == 0 {
        return nil
    }

    pipe := s.client.TxPipeline()

    // 全セッションキーを削除
    for _, sessionID := range sessionIDs {
        pipe.Del(ctx, SessionKey(sessionID))
    }

    // ユーザーセッション一覧を削除
    pipe.Del(ctx, userSessionsKey)

    _, err = pipe.Exec(ctx)
    return err
}

// Verify interface compliance
var _ repository.SessionRepository = (*SessionStore)(nil)
```

### 3.3 セッションリポジトリインターフェース

```go
// backend/internal/domain/repository/session.go

package repository

import (
    "context"
    "errors"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

var ErrSessionNotFound = errors.New("session not found")

// SessionRepository はセッション永続化のインターフェースを定義します
type SessionRepository interface {
    Save(ctx context.Context, session *entity.Session) error
    FindByID(ctx context.Context, sessionID string) (*entity.Session, error)
    FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Session, error)
    UpdateLastUsed(ctx context.Context, sessionID string) error
    Delete(ctx context.Context, sessionID string) error
    DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}
```

---

## 4. JWTブラックリスト

### 4.1 ブラックリスト実装

```go
// backend/internal/infrastructure/persistence/redis/jwt_blacklist.go

package redis

import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

// JWTBlacklist はJWTトークンのブラックリストを管理します
type JWTBlacklist struct {
    client *redis.Client
}

// NewJWTBlacklist は新しいJWTBlacklistを作成します
func NewJWTBlacklist(client *redis.Client) *JWTBlacklist {
    return &JWTBlacklist{client: client}
}

// Add はトークンをブラックリストに追加します
// jti: JWT ID (unique identifier)
// expiry: トークンの有効期限（ブラックリストエントリのTTLとして使用）
func (b *JWTBlacklist) Add(ctx context.Context, jti string, expiry time.Time) error {
    key := JWTBlacklistKey(jti)

    // TTLはトークンの残り有効期限
    ttl := time.Until(expiry)
    if ttl <= 0 {
        return nil // すでに期限切れ
    }

    err := b.client.Set(ctx, key, "1", ttl).Err()
    if err != nil {
        return fmt.Errorf("failed to add to blacklist: %w", err)
    }

    return nil
}

// IsBlacklisted はトークンがブラックリストに存在するか確認します
func (b *JWTBlacklist) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
    key := JWTBlacklistKey(jti)

    exists, err := b.client.Exists(ctx, key).Result()
    if err != nil {
        return false, fmt.Errorf("failed to check blacklist: %w", err)
    }

    return exists > 0, nil
}

// Remove はトークンをブラックリストから削除します（通常は不要、TTLで自動削除）
func (b *JWTBlacklist) Remove(ctx context.Context, jti string) error {
    key := JWTBlacklistKey(jti)
    return b.client.Del(ctx, key).Err()
}
```

### 4.2 ブラックリストサービスインターフェース

```go
// backend/internal/domain/service/jwt_blacklist.go

package service

import (
    "context"
    "time"
)

// JWTBlacklistService はJWTブラックリストのインターフェースを定義します
type JWTBlacklistService interface {
    Add(ctx context.Context, jti string, expiry time.Time) error
    IsBlacklisted(ctx context.Context, jti string) (bool, error)
}
```

---

## 5. レートリミッター

### 5.1 Token Bucket / Sliding Window 実装

```go
// backend/internal/infrastructure/persistence/redis/rate_limiter.go

package redis

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
```

### 5.2 レートリミッターミドルウェア

```go
// backend/internal/interface/middleware/rate_limit.go

package middleware

import (
    "fmt"
    "net/http"
    "strconv"

    "github.com/labstack/echo/v4"

    redisinfra "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/persistence/redis"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// RateLimitMiddleware はレート制限ミドルウェアを提供します
type RateLimitMiddleware struct {
    limiter *redisinfra.RateLimiter
}

// NewRateLimitMiddleware は新しいRateLimitMiddlewareを作成します
func NewRateLimitMiddleware(limiter *redisinfra.RateLimiter) *RateLimitMiddleware {
    return &RateLimitMiddleware{limiter: limiter}
}

// Limit は指定されたレート制限を適用するミドルウェアを返します
func (m *RateLimitMiddleware) Limit(config redisinfra.RateLimitConfig, identifierFunc func(echo.Context) string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            identifier := identifierFunc(c)

            result, err := m.limiter.Allow(c.Request().Context(), identifier, config)
            if err != nil {
                // レート制限チェックに失敗した場合は許可（フェイルオープン）
                return next(c)
            }

            // レート制限ヘッダーを設定
            c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(config.Requests))
            c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
            c.Response().Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

            if !result.Allowed {
                c.Response().Header().Set("Retry-After", strconv.FormatInt(int64(result.RetryAt.Sub(result.ResetAt).Seconds()), 10))
                return apperror.NewTooManyRequestsError(
                    fmt.Sprintf("rate limit exceeded, retry after %s", result.RetryAt.Format("15:04:05")),
                )
            }

            return next(c)
        }
    }
}

// ByIP はIPアドレスでレート制限を行うミドルウェアを返します
func (m *RateLimitMiddleware) ByIP(config redisinfra.RateLimitConfig) echo.MiddlewareFunc {
    return m.Limit(config, func(c echo.Context) string {
        return c.RealIP()
    })
}

// ByUser はユーザーIDでレート制限を行うミドルウェアを返します
func (m *RateLimitMiddleware) ByUser(config redisinfra.RateLimitConfig) echo.MiddlewareFunc {
    return m.Limit(config, func(c echo.Context) string {
        userID := c.Get("user_id")
        if userID == nil {
            return c.RealIP() // 未認証の場合はIPで制限
        }
        return fmt.Sprintf("user:%s", userID)
    })
}
```

---

## 6. 汎用キャッシュ

### 6.1 キャッシュ実装

```go
// backend/internal/infrastructure/persistence/redis/cache.go

package redis

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

// Cache は汎用キャッシュを提供します
type Cache struct {
    client    *redis.Client
    namespace string
    defaultTTL time.Duration
}

// NewCache は新しいCacheを作成します
func NewCache(client *redis.Client, namespace string, defaultTTL time.Duration) *Cache {
    return &Cache{
        client:    client,
        namespace: namespace,
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

    // 結果をキャッシュ
    if err := c.Set(ctx, key, value, ttl); err != nil {
        // キャッシュ失敗はログのみ
    }

    // destに値を設定
    data, _ := json.Marshal(value)
    return json.Unmarshal(data, dest)
}

// ErrCacheMiss はキャッシュミスを表すエラーです
var ErrCacheMiss = fmt.Errorf("cache miss")
```

### 6.2 ユーザーキャッシュ例

```go
// backend/internal/infrastructure/persistence/redis/user_cache.go

package redis

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// UserCache はユーザー情報のキャッシュを提供します
type UserCache struct {
    cache *Cache
}

// NewUserCache は新しいUserCacheを作成します
func NewUserCache(client *redis.Client) *UserCache {
    return &UserCache{
        cache: NewCache(client, "user", 15*time.Minute),
    }
}

// GetUser はユーザー情報をキャッシュから取得します
func (c *UserCache) GetUser(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
    var user entity.User
    if err := c.cache.Get(ctx, userID.String(), &user); err != nil {
        return nil, err
    }
    return &user, nil
}

// SetUser はユーザー情報をキャッシュに設定します
func (c *UserCache) SetUser(ctx context.Context, user *entity.User) error {
    return c.cache.Set(ctx, user.ID.String(), user)
}

// InvalidateUser はユーザーキャッシュを無効化します
func (c *UserCache) InvalidateUser(ctx context.Context, userID uuid.UUID) error {
    return c.cache.Delete(ctx, userID.String())
}
```

---

## 7. エラーハンドリング

### 7.1 Redisエラーの分類

```go
// pkg/apperror/redis.go

package apperror

import (
    "errors"
    "fmt"

    "github.com/redis/go-redis/v9"
)

// Redis関連エラー
var (
    ErrRedisConnection = errors.New("redis connection error")
    ErrRedisTimeout    = errors.New("redis timeout error")
    ErrRedisOperation  = errors.New("redis operation error")
)

// WrapRedisError はRedisエラーをアプリケーションエラーにラップします
func WrapRedisError(err error) error {
    if err == nil {
        return nil
    }

    // 接続エラー
    if errors.Is(err, redis.ErrClosed) {
        return fmt.Errorf("%w: %v", ErrRedisConnection, err)
    }

    // タイムアウト
    if isTimeout(err) {
        return fmt.Errorf("%w: %v", ErrRedisTimeout, err)
    }

    return fmt.Errorf("%w: %v", ErrRedisOperation, err)
}

func isTimeout(err error) bool {
    type timeout interface {
        Timeout() bool
    }
    if t, ok := err.(timeout); ok {
        return t.Timeout()
    }
    return false
}
```

### 7.2 サーキットブレーカー

```go
// backend/internal/infrastructure/persistence/redis/circuit_breaker.go

package redis

import (
    "context"
    "sync"
    "time"
)

// CircuitState はサーキットブレーカーの状態を表します
type CircuitState int

const (
    StateClosed CircuitState = iota
    StateOpen
    StateHalfOpen
)

// CircuitBreaker はRedis操作のサーキットブレーカーを提供します
type CircuitBreaker struct {
    mu              sync.RWMutex
    state           CircuitState
    failures        int
    lastFailure     time.Time
    threshold       int           // 失敗閾値
    timeout         time.Duration // オープン状態のタイムアウト
    halfOpenTimeout time.Duration // ハーフオープン状態のタイムアウト
}

// NewCircuitBreaker は新しいCircuitBreakerを作成します
func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
    return &CircuitBreaker{
        state:           StateClosed,
        threshold:       threshold,
        timeout:         timeout,
        halfOpenTimeout: timeout / 2,
    }
}

// Execute はサーキットブレーカーを通じて操作を実行します
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
    cb.mu.RLock()
    state := cb.state
    cb.mu.RUnlock()

    switch state {
    case StateOpen:
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.mu.Lock()
            cb.state = StateHalfOpen
            cb.mu.Unlock()
            return cb.execute(fn)
        }
        return ErrCircuitOpen
    case StateHalfOpen:
        return cb.execute(fn)
    default:
        return cb.execute(fn)
    }
}

func (cb *CircuitBreaker) execute(fn func() error) error {
    err := fn()

    cb.mu.Lock()
    defer cb.mu.Unlock()

    if err != nil {
        cb.failures++
        cb.lastFailure = time.Now()

        if cb.failures >= cb.threshold {
            cb.state = StateOpen
        }
        return err
    }

    // 成功時はリセット
    cb.failures = 0
    cb.state = StateClosed
    return nil
}

// ErrCircuitOpen はサーキットがオープン状態であることを示します
var ErrCircuitOpen = fmt.Errorf("circuit breaker is open")
```

---

## 8. 初期化とDI

### 8.1 Redis依存関係の初期化

```go
// backend/internal/infrastructure/di/redis.go

package di

import (
    "time"

    "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/persistence/redis"
)

// RedisComponents はRedis関連の依存関係を保持します
type RedisComponents struct {
    Client       *redis.RedisClient
    SessionStore *redis.SessionStore
    JWTBlacklist *redis.JWTBlacklist
    RateLimiter  *redis.RateLimiter
    UserCache    *redis.UserCache
}

// NewRedisComponents はRedis関連の依存関係を初期化します
func NewRedisComponents(redisURL string) (*RedisComponents, error) {
    cfg := redis.DefaultConfig()
    cfg.URL = redisURL

    client, err := redis.NewRedisClient(cfg)
    if err != nil {
        return nil, err
    }

    return &RedisComponents{
        Client:       client,
        SessionStore: redis.NewSessionStore(client.Client(), 7*24*time.Hour),
        JWTBlacklist: redis.NewJWTBlacklist(client.Client()),
        RateLimiter:  redis.NewRateLimiter(client.Client()),
        UserCache:    redis.NewUserCache(client.Client()),
    }, nil
}

// Close は全てのRedis接続を閉じます
func (c *RedisComponents) Close() error {
    return c.Client.Close()
}
```

---

## 9. テストヘルパー

### 9.1 Redis Testcontainer

```go
// backend/internal/infrastructure/persistence/redis/testhelper/redis.go

package testhelper

import (
    "context"
    "testing"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

// RedisContainer はテスト用Redisコンテナを管理します
type RedisContainer struct {
    Container testcontainers.Container
    Client    *redis.Client
    URL       string
}

// NewRedisContainer はテスト用Redisコンテナを起動します
func NewRedisContainer(t *testing.T) *RedisContainer {
    t.Helper()

    ctx := context.Background()

    req := testcontainers.ContainerRequest{
        Image:        "redis:7-alpine",
        ExposedPorts: []string{"6379/tcp"},
        WaitingFor:   wait.ForLog("Ready to accept connections"),
    }

    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    if err != nil {
        t.Fatalf("failed to start redis container: %v", err)
    }

    host, err := container.Host(ctx)
    if err != nil {
        t.Fatalf("failed to get container host: %v", err)
    }

    port, err := container.MappedPort(ctx, "6379")
    if err != nil {
        t.Fatalf("failed to get container port: %v", err)
    }

    url := fmt.Sprintf("redis://%s:%s", host, port.Port())

    client := redis.NewClient(&redis.Options{
        Addr: fmt.Sprintf("%s:%s", host, port.Port()),
    })

    // 接続確認
    ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    if err := client.Ping(ctxTimeout).Err(); err != nil {
        t.Fatalf("failed to connect to redis: %v", err)
    }

    t.Cleanup(func() {
        client.Close()
        container.Terminate(ctx)
    })

    return &RedisContainer{
        Container: container,
        Client:    client,
        URL:       url,
    }
}

// FlushAll はすべてのデータを削除します
func (c *RedisContainer) FlushAll(ctx context.Context) error {
    return c.Client.FlushAll(ctx).Err()
}
```

### 9.2 セッションストアテスト例

```go
// backend/internal/infrastructure/persistence/redis/session_store_test.go

package redis_test

import (
    "context"
    "testing"
    "time"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    redisinfra "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/persistence/redis"
    "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/persistence/redis/testhelper"
)

func TestSessionStore_Save_and_FindByID(t *testing.T) {
    container := testhelper.NewRedisContainer(t)
    store := redisinfra.NewSessionStore(container.Client, 7*24*time.Hour)
    ctx := context.Background()

    session := &entity.Session{
        ID:           "test-session-id",
        UserID:       uuid.New(),
        RefreshToken: "test-refresh-token",
        UserAgent:    "Mozilla/5.0",
        IPAddress:    "192.168.1.1",
        CreatedAt:    time.Now(),
        ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
        LastUsedAt:   time.Now(),
    }

    // Save
    err := store.Save(ctx, session)
    require.NoError(t, err)

    // FindByID
    found, err := store.FindByID(ctx, session.ID)
    require.NoError(t, err)
    assert.Equal(t, session.ID, found.ID)
    assert.Equal(t, session.UserID, found.UserID)
    assert.Equal(t, session.RefreshToken, found.RefreshToken)
}

func TestSessionStore_DeleteByUserID(t *testing.T) {
    container := testhelper.NewRedisContainer(t)
    store := redisinfra.NewSessionStore(container.Client, 7*24*time.Hour)
    ctx := context.Background()

    userID := uuid.New()

    // 複数セッションを作成
    for i := 0; i < 3; i++ {
        session := &entity.Session{
            ID:        fmt.Sprintf("session-%d", i),
            UserID:    userID,
            ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
        }
        require.NoError(t, store.Save(ctx, session))
    }

    // 全セッション削除
    err := store.DeleteByUserID(ctx, userID)
    require.NoError(t, err)

    // 確認
    sessions, err := store.FindByUserID(ctx, userID)
    require.NoError(t, err)
    assert.Empty(t, sessions)
}
```

---

## 10. 受け入れ基準

### 10.1 機能要件

| 項目 | 基準 |
|------|------|
| セッション保存 | TTL付きでセッションを保存できる |
| セッション取得 | セッションIDで取得できる |
| ユーザーセッション一覧 | ユーザーIDで全セッションを取得できる |
| セッション削除 | 単一セッション・全セッション削除ができる |
| JWTブラックリスト | トークンの無効化・確認ができる |
| レート制限 | Sliding Window / Fixed Window で制限できる |
| キャッシュ | Get/Set/Delete/パターン削除ができる |

### 10.2 非機能要件

| 項目 | 基準 |
|------|------|
| 接続タイムアウト | 5秒以内 |
| 操作タイムアウト | 3秒以内 |
| 接続プール | min 10, max 100 |
| エラー時フェイルオーバー | レート制限はフェイルオープン |
| テストカバレッジ | 80%以上 |

### 10.3 チェックリスト

- [ ] Redis接続が確立できる
- [ ] セッションの CRUD 操作が正常動作する
- [ ] ユーザーの全セッション一括削除が動作する
- [ ] JWTブラックリストの追加・確認が動作する
- [ ] レート制限が正しく機能する
- [ ] キャッシュの Get/Set/Delete が動作する
- [ ] パイプライン・Luaスクリプトが正常動作する
- [ ] 接続プールが適切に管理される
- [ ] エラーハンドリングが適切である
- [ ] テストコンテナでのテストが通過する

---

## 関連ドキュメント

- [infra-database.md](./infra-database.md) - PostgreSQL基盤仕様
- [auth-identity.md](./auth-identity.md) - 認証仕様（セッション管理の利用元）
- [SECURITY.md](../02-architecture/SECURITY.md) - セキュリティ設計
