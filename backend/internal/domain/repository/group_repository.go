package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// GroupRepository はグループリポジトリのインターフェース
type GroupRepository interface {
	// 基本CRUD
	Create(ctx context.Context, group *entity.Group) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Group, error)
	Update(ctx context.Context, group *entity.Group) error
	Delete(ctx context.Context, id uuid.UUID) error

	// 検索
	FindByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*entity.Group, error)
	FindByMemberID(ctx context.Context, userID uuid.UUID) ([]*entity.Group, error)
	FindActiveByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*entity.Group, error)

	// 存在チェック
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)
}
