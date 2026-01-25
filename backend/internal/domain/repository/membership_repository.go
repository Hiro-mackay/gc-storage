package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// MembershipRepository はメンバーシップリポジトリのインターフェース
type MembershipRepository interface {
	// 基本CRUD
	Create(ctx context.Context, membership *entity.Membership) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Membership, error)
	Update(ctx context.Context, membership *entity.Membership) error
	Delete(ctx context.Context, id uuid.UUID) error

	// 検索
	FindByGroupID(ctx context.Context, groupID uuid.UUID) ([]*entity.Membership, error)
	FindByGroupIDWithUsers(ctx context.Context, groupID uuid.UUID) ([]*entity.MembershipWithUser, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Membership, error)
	FindByGroupAndUser(ctx context.Context, groupID, userID uuid.UUID) (*entity.Membership, error)

	// 存在・カウント
	Exists(ctx context.Context, groupID, userID uuid.UUID) (bool, error)
	CountByGroupID(ctx context.Context, groupID uuid.UUID) (int, error)

	// 一括操作
	DeleteByGroupID(ctx context.Context, groupID uuid.UUID) error
}
