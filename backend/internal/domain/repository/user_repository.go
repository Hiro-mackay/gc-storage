package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// UserRepository はユーザーリポジトリインターフェースを定義します
type UserRepository interface {
	// Create はユーザーを作成します
	Create(ctx context.Context, user *entity.User) error

	// Update はユーザーを更新します
	Update(ctx context.Context, user *entity.User) error

	// FindByID はIDでユーザーを検索します
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)

	// FindByEmail はメールアドレスでユーザーを検索します
	FindByEmail(ctx context.Context, email valueobject.Email) (*entity.User, error)

	// Exists はメールアドレスが存在するかを確認します
	Exists(ctx context.Context, email valueobject.Email) (bool, error)

	// Delete はユーザーを削除します
	Delete(ctx context.Context, id uuid.UUID) error

	// UpdateStorageUsed はストレージ使用量を更新します
	UpdateStorageUsed(ctx context.Context, id uuid.UUID, bytesUsed int64) error
}
