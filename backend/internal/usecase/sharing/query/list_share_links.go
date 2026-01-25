package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ListShareLinksInput は共有リンク一覧取得の入力を定義します
type ListShareLinksInput struct {
	ResourceType string
	ResourceID   uuid.UUID
	UserID       uuid.UUID
}

// ListShareLinksOutput は共有リンク一覧取得の出力を定義します
type ListShareLinksOutput struct {
	ShareLinks []*entity.ShareLink
}

// ListShareLinksQuery は共有リンク一覧取得クエリです
type ListShareLinksQuery struct {
	shareLinkRepo      repository.ShareLinkRepository
	permissionResolver authz.PermissionResolver
}

// NewListShareLinksQuery は新しいListShareLinksQueryを作成します
func NewListShareLinksQuery(
	shareLinkRepo repository.ShareLinkRepository,
	permissionResolver authz.PermissionResolver,
) *ListShareLinksQuery {
	return &ListShareLinksQuery{
		shareLinkRepo:      shareLinkRepo,
		permissionResolver: permissionResolver,
	}
}

// Execute は共有リンク一覧取得を実行します
func (q *ListShareLinksQuery) Execute(ctx context.Context, input ListShareLinksInput) (*ListShareLinksOutput, error) {
	// 1. リソースタイプのバリデーション
	resourceType, err := authz.NewResourceType(input.ResourceType)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 2. ユーザーが共有権限を持っているか確認
	var requiredPermission authz.Permission
	if resourceType == authz.ResourceTypeFile {
		requiredPermission = authz.PermFileShare
	} else {
		requiredPermission = authz.PermFolderShare
	}

	hasPermission, err := q.permissionResolver.HasPermission(ctx, input.UserID, resourceType, input.ResourceID, requiredPermission)
	if err != nil {
		return nil, err
	}
	if !hasPermission {
		return nil, apperror.NewForbiddenError("you do not have permission to view share links for this resource")
	}

	// 3. 共有リンク一覧を取得
	shareLinks, err := q.shareLinkRepo.FindByResource(ctx, resourceType, input.ResourceID)
	if err != nil {
		return nil, err
	}

	return &ListShareLinksOutput{ShareLinks: shareLinks}, nil
}
