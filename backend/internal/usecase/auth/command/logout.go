package command

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/cache"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/jwt"
)

// LogoutCommand はログアウトコマンドです
type LogoutCommand struct {
	sessionStore *cache.SessionStore
	jwtBlacklist *cache.JWTBlacklist
}

// NewLogoutCommand は新しいLogoutCommandを作成します
func NewLogoutCommand(
	sessionStore *cache.SessionStore,
	jwtBlacklist *cache.JWTBlacklist,
) *LogoutCommand {
	return &LogoutCommand{
		sessionStore: sessionStore,
		jwtBlacklist: jwtBlacklist,
	}
}

// Execute はログアウトを実行します
func (c *LogoutCommand) Execute(ctx context.Context, sessionID string, accessTokenClaims *jwt.AccessTokenClaims) error {
	// 1. セッションを削除
	if err := c.sessionStore.Delete(ctx, sessionID); err != nil {
		// エラーでも続行
	}

	// 2. アクセストークンをブラックリストに追加
	if accessTokenClaims != nil && c.jwtBlacklist != nil {
		if accessTokenClaims.ExpiresAt != nil {
			c.jwtBlacklist.Add(ctx, accessTokenClaims.ID, accessTokenClaims.ExpiresAt.Time)
		} else {
			// 有効期限がない場合は15分後に設定
			c.jwtBlacklist.Add(ctx, accessTokenClaims.ID, time.Now().Add(15*time.Minute))
		}
	}

	return nil
}

// ExecuteAll は全セッションからログアウトを実行します
func (c *LogoutCommand) ExecuteAll(ctx context.Context, userID uuid.UUID) error {
	return c.sessionStore.DeleteByUserID(ctx, userID)
}
