package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ListGrantsInput は権限一覧取得の入力を定義します
type ListGrantsInput struct {
	ResourceType string
	ResourceID   uuid.UUID
	UserID       uuid.UUID
}

// ListGrantsOutput は権限一覧取得の出力を定義します
type ListGrantsOutput struct {
	Grants []*authz.PermissionGrant
}

// ListGrantsQuery は権限一覧取得クエリです
type ListGrantsQuery struct {
	permissionGrantRepo authz.PermissionGrantRepository
	permissionResolver  authz.PermissionResolver
}

// NewListGrantsQuery は新しいListGrantsQueryを作成します
func NewListGrantsQuery(
	permissionGrantRepo authz.PermissionGrantRepository,
	permissionResolver authz.PermissionResolver,
) *ListGrantsQuery {
	return &ListGrantsQuery{
		permissionGrantRepo: permissionGrantRepo,
		permissionResolver:  permissionResolver,
	}
}

// Execute は権限一覧取得を実行します
func (q *ListGrantsQuery) Execute(ctx context.Context, input ListGrantsInput) (*ListGrantsOutput, error) {
	// 1. リソースタイプのバリデーション
	resourceType, err := authz.NewResourceType(input.ResourceType)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 2. ユーザーがこのリソースの権限を閲覧できるか確認
	hasPermission, err := q.permissionResolver.HasPermission(ctx, input.UserID, resourceType, input.ResourceID, authz.PermPermissionRead)
	if err != nil {
		return nil, err
	}
	if !hasPermission {
		return nil, apperror.NewForbiddenError("you do not have permission to view grants for this resource")
	}

	// 3. 権限一覧を取得
	grants, err := q.permissionGrantRepo.FindByResource(ctx, resourceType, input.ResourceID)
	if err != nil {
		return nil, err
	}

	return &ListGrantsOutput{Grants: grants}, nil
}
