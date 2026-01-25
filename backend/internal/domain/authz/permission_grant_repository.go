package authz

import (
	"context"

	"github.com/google/uuid"
)

// PermissionGrantRepository は権限付与リポジトリのインターフェース
type PermissionGrantRepository interface {
	// 基本CRUD
	Create(ctx context.Context, grant *PermissionGrant) error
	FindByID(ctx context.Context, id uuid.UUID) (*PermissionGrant, error)
	Delete(ctx context.Context, id uuid.UUID) error

	// リソースでの検索
	FindByResource(ctx context.Context, resourceType ResourceType, resourceID uuid.UUID) ([]*PermissionGrant, error)
	FindByResourceAndGrantee(ctx context.Context, resourceType ResourceType, resourceID uuid.UUID, granteeType GranteeType, granteeID uuid.UUID) ([]*PermissionGrant, error)
	FindByResourceGranteeAndRole(ctx context.Context, resourceType ResourceType, resourceID uuid.UUID, granteeType GranteeType, granteeID uuid.UUID, role Role) (*PermissionGrant, error)

	// 付与対象での検索
	FindByGrantee(ctx context.Context, granteeType GranteeType, granteeID uuid.UUID) ([]*PermissionGrant, error)

	// 一括削除
	DeleteByResource(ctx context.Context, resourceType ResourceType, resourceID uuid.UUID) error
	DeleteByGrantee(ctx context.Context, granteeType GranteeType, granteeID uuid.UUID) error
	DeleteByResourceAndGrantee(ctx context.Context, resourceType ResourceType, resourceID uuid.UUID, granteeType GranteeType, granteeID uuid.UUID) error
}
