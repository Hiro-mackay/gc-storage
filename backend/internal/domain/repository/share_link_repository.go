package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// ShareLinkRepository は共有リンクリポジトリのインターフェース
type ShareLinkRepository interface {
	// 基本CRUD
	Create(ctx context.Context, link *entity.ShareLink) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.ShareLink, error)
	Update(ctx context.Context, link *entity.ShareLink) error
	Delete(ctx context.Context, id uuid.UUID) error

	// トークン検索
	FindByToken(ctx context.Context, token valueobject.ShareToken) (*entity.ShareLink, error)

	// 検索
	FindByResource(ctx context.Context, resourceType authz.ResourceType, resourceID uuid.UUID) ([]*entity.ShareLink, error)
	FindByCreator(ctx context.Context, createdBy uuid.UUID) ([]*entity.ShareLink, error)
	FindActiveByResource(ctx context.Context, resourceType authz.ResourceType, resourceID uuid.UUID) ([]*entity.ShareLink, error)

	// 期限切れ処理
	FindExpired(ctx context.Context) ([]*entity.ShareLink, error)
	UpdateStatusBatch(ctx context.Context, ids []uuid.UUID, status valueobject.ShareLinkStatus) (int64, error)
}

// ShareLinkAccessRepository は共有リンクアクセスログリポジトリのインターフェース
type ShareLinkAccessRepository interface {
	// 基本CRUD
	Create(ctx context.Context, access *entity.ShareLinkAccess) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.ShareLinkAccess, error)

	// 検索
	FindByShareLinkID(ctx context.Context, shareLinkID uuid.UUID) ([]*entity.ShareLinkAccess, error)
	FindByShareLinkIDWithPagination(ctx context.Context, shareLinkID uuid.UUID, limit, offset int) ([]*entity.ShareLinkAccess, error)

	// カウント
	CountByShareLinkID(ctx context.Context, shareLinkID uuid.UUID) (int, error)

	// 一括削除
	DeleteByShareLinkID(ctx context.Context, shareLinkID uuid.UUID) error

	// 古いアクセスログの匿名化
	AnonymizeOldAccesses(ctx context.Context) (int64, error)
}
