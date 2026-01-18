package cache

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
