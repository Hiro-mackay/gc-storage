package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// CheckPermissionInput は権限確認の入力を定義します
type CheckPermissionInput struct {
	UserID       uuid.UUID
	ResourceType string
	ResourceID   uuid.UUID
	Permission   string
}

// CheckPermissionOutput は権限確認の出力を定義します
type CheckPermissionOutput struct {
	HasPermission bool
	EffectiveRole string
}

// CheckPermissionQuery は権限確認クエリです
type CheckPermissionQuery struct {
	permissionResolver authz.PermissionResolver
}

// NewCheckPermissionQuery は新しいCheckPermissionQueryを作成します
func NewCheckPermissionQuery(
	permissionResolver authz.PermissionResolver,
) *CheckPermissionQuery {
	return &CheckPermissionQuery{
		permissionResolver: permissionResolver,
	}
}

// Execute は権限確認を実行します
func (q *CheckPermissionQuery) Execute(ctx context.Context, input CheckPermissionInput) (*CheckPermissionOutput, error) {
	// 1. リソースタイプのバリデーション
	resourceType, err := authz.NewResourceType(input.ResourceType)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 2. 権限のバリデーション
	permission, err := authz.NewPermission(input.Permission)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 3. 権限を確認
	hasPermission, err := q.permissionResolver.HasPermission(ctx, input.UserID, resourceType, input.ResourceID, permission)
	if err != nil {
		return nil, err
	}

	// 4. 有効なロールを取得
	effectiveRole, err := q.permissionResolver.GetEffectiveRole(ctx, input.UserID, resourceType, input.ResourceID)
	if err != nil {
		return nil, err
	}

	return &CheckPermissionOutput{
		HasPermission: hasPermission,
		EffectiveRole: string(effectiveRole),
	}, nil
}
