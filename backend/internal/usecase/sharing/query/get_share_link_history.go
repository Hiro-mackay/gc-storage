package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

const defaultHistoryLimit = 20
const maxHistoryLimit = 100

// GetShareLinkHistoryInput は共有リンクアクセス履歴取得の入力を定義します
type GetShareLinkHistoryInput struct {
	ShareLinkID uuid.UUID
	UserID      uuid.UUID
	Limit       int
	Offset      int
}

// GetShareLinkHistoryOutput は共有リンクアクセス履歴取得の出力を定義します
type GetShareLinkHistoryOutput struct {
	Accesses []*entity.ShareLinkAccess
	Total    int
}

// GetShareLinkHistoryQuery は共有リンクアクセス履歴取得クエリです
type GetShareLinkHistoryQuery struct {
	shareLinkRepo       repository.ShareLinkRepository
	shareLinkAccessRepo repository.ShareLinkAccessRepository
	permissionResolver  authz.PermissionResolver
}

// NewGetShareLinkHistoryQuery は新しいGetShareLinkHistoryQueryを作成します
func NewGetShareLinkHistoryQuery(
	shareLinkRepo repository.ShareLinkRepository,
	shareLinkAccessRepo repository.ShareLinkAccessRepository,
	permissionResolver authz.PermissionResolver,
) *GetShareLinkHistoryQuery {
	return &GetShareLinkHistoryQuery{
		shareLinkRepo:       shareLinkRepo,
		shareLinkAccessRepo: shareLinkAccessRepo,
		permissionResolver:  permissionResolver,
	}
}

// Execute は共有リンクアクセス履歴取得を実行します
func (q *GetShareLinkHistoryQuery) Execute(ctx context.Context, input GetShareLinkHistoryInput) (*GetShareLinkHistoryOutput, error) {
	// 1. 共有リンクを取得
	shareLink, err := q.shareLinkRepo.FindByID(ctx, input.ShareLinkID)
	if err != nil {
		return nil, err
	}

	// 2. 権限チェック（作成者でない場合）
	if !shareLink.IsCreatedBy(input.UserID) {
		var requiredPermission authz.Permission
		if shareLink.ResourceType == authz.ResourceTypeFile {
			requiredPermission = authz.PermFileShare
		} else {
			requiredPermission = authz.PermFolderShare
		}

		hasPermission, err := q.permissionResolver.HasPermission(ctx, input.UserID, shareLink.ResourceType, shareLink.ResourceID, requiredPermission)
		if err != nil {
			return nil, err
		}
		if !hasPermission {
			return nil, apperror.NewForbiddenError("you do not have permission to view this share link's history")
		}
	}

	// 3. ページネーション設定
	limit := input.Limit
	if limit <= 0 {
		limit = defaultHistoryLimit
	}
	if limit > maxHistoryLimit {
		limit = maxHistoryLimit
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	// 4. アクセス履歴を取得
	accesses, err := q.shareLinkAccessRepo.FindByShareLinkIDWithPagination(ctx, input.ShareLinkID, limit, offset)
	if err != nil {
		return nil, err
	}

	// 5. 総件数を取得
	total, err := q.shareLinkAccessRepo.CountByShareLinkID(ctx, input.ShareLinkID)
	if err != nil {
		return nil, err
	}

	return &GetShareLinkHistoryOutput{
		Accesses: accesses,
		Total:    total,
	}, nil
}
