package cache

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
	PrefixCache     KeyPrefix = "cache"      // cache:{namespace}:{key}
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
