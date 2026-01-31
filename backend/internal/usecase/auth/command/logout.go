package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
)

// LogoutCommand はログアウトコマンドです
type LogoutCommand struct {
	sessionRepo repository.SessionRepository
}

// NewLogoutCommand は新しいLogoutCommandを作成します
func NewLogoutCommand(sessionRepo repository.SessionRepository) *LogoutCommand {
	return &LogoutCommand{
		sessionRepo: sessionRepo,
	}
}

// Execute はログアウトを実行します
func (c *LogoutCommand) Execute(ctx context.Context, sessionID string) error {
	return c.sessionRepo.Delete(ctx, sessionID)
}

// ExecuteAll は全セッションからログアウトを実行します
func (c *LogoutCommand) ExecuteAll(ctx context.Context, userID uuid.UUID) error {
	return c.sessionRepo.DeleteByUserID(ctx, userID)
}
