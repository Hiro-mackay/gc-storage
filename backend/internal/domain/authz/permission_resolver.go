package authz

import (
	"context"

	"github.com/google/uuid"
)

// PermissionResolver は権限解決を行うサービスインターフェース
type PermissionResolver interface {
	// HasPermission はユーザーがリソースに対して指定された権限を持つかを判定します
	HasPermission(ctx context.Context, userID uuid.UUID, resourceType ResourceType, resourceID uuid.UUID, permission Permission) (bool, error)

	// CollectPermissions はユーザーがリソースに対して持つ権限を全て取得します
	CollectPermissions(ctx context.Context, userID uuid.UUID, resourceType ResourceType, resourceID uuid.UUID) (*PermissionSet, error)

	// GetEffectiveRole はユーザーがリソースに対して持つ最も高いロールを取得します
	GetEffectiveRole(ctx context.Context, userID uuid.UUID, resourceType ResourceType, resourceID uuid.UUID) (Role, error)

	// IsOwner はユーザーがリソースのオーナーかを判定します
	IsOwner(ctx context.Context, userID uuid.UUID, resourceType ResourceType, resourceID uuid.UUID) (bool, error)

	// CanGrantRole はユーザーがリソースに対して指定されたロールを付与可能かを判定します
	CanGrantRole(ctx context.Context, userID uuid.UUID, resourceType ResourceType, resourceID uuid.UUID, targetRole Role) (bool, error)
}
