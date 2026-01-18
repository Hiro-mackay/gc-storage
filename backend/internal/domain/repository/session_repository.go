package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// SessionRepository はセッションリポジトリインターフェースを定義します
type SessionRepository interface {
	// Save はセッションを保存します
	Save(ctx context.Context, session *entity.Session) error

	// FindByID はIDでセッションを検索します
	FindByID(ctx context.Context, sessionID string) (*entity.Session, error)

	// FindByUserID はユーザーIDでセッション一覧を取得します
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Session, error)

	// Delete はセッションを削除します
	Delete(ctx context.Context, sessionID string) error

	// DeleteByUserID はユーザーの全セッションを削除します
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}
