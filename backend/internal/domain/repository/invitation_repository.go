package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// InvitationRepository は招待リポジトリのインターフェース
type InvitationRepository interface {
	// 基本CRUD
	Create(ctx context.Context, invitation *entity.Invitation) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Invitation, error)
	Update(ctx context.Context, invitation *entity.Invitation) error
	Delete(ctx context.Context, id uuid.UUID) error

	// トークン検索
	FindByToken(ctx context.Context, token string) (*entity.Invitation, error)

	// 検索
	FindPendingByGroupID(ctx context.Context, groupID uuid.UUID) ([]*entity.Invitation, error)
	FindPendingByEmail(ctx context.Context, email valueobject.Email) ([]*entity.Invitation, error)
	FindPendingByGroupAndEmail(ctx context.Context, groupID uuid.UUID, email valueobject.Email) (*entity.Invitation, error)

	// 一括操作
	DeleteByGroupID(ctx context.Context, groupID uuid.UUID) error
	ExpireOld(ctx context.Context) (int64, error)
}
