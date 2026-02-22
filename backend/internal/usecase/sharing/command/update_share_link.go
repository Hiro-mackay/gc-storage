package command

import (
	"context"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// UpdateShareLinkInput は共有リンク更新の入力を定義します
type UpdateShareLinkInput struct {
	ShareLinkID    uuid.UUID
	UpdatedBy      uuid.UUID
	Password       *string    // optional, set to clear or update password
	ExpiresAt      *time.Time // optional
	MaxAccessCount *int       // optional
}

// UpdateShareLinkOutput は共有リンク更新の出力を定義します
type UpdateShareLinkOutput struct {
	ShareLink *entity.ShareLink
}

// UpdateShareLinkCommand は共有リンク更新コマンドです
type UpdateShareLinkCommand struct {
	shareLinkRepo      repository.ShareLinkRepository
	permissionResolver authz.PermissionResolver
}

// NewUpdateShareLinkCommand は新しいUpdateShareLinkCommandを作成します
func NewUpdateShareLinkCommand(
	shareLinkRepo repository.ShareLinkRepository,
	permissionResolver authz.PermissionResolver,
) *UpdateShareLinkCommand {
	return &UpdateShareLinkCommand{
		shareLinkRepo:      shareLinkRepo,
		permissionResolver: permissionResolver,
	}
}

// Execute は共有リンク更新を実行します
func (c *UpdateShareLinkCommand) Execute(ctx context.Context, input UpdateShareLinkInput) (*UpdateShareLinkOutput, error) {
	// 1. 共有リンクを取得
	shareLink, err := c.shareLinkRepo.FindByID(ctx, input.ShareLinkID)
	if err != nil {
		return nil, err
	}

	// 2. アクティブ状態チェック
	if !shareLink.IsActive() {
		return nil, apperror.NewValidationError("share link is not active and cannot be updated", nil)
	}

	// 3. 権限チェック（作成者でない場合）
	if !shareLink.IsCreatedBy(input.UpdatedBy) {
		var requiredPermission authz.Permission
		if shareLink.ResourceType == authz.ResourceTypeFile {
			requiredPermission = authz.PermFileShare
		} else {
			requiredPermission = authz.PermFolderShare
		}

		hasPermission, err := c.permissionResolver.HasPermission(ctx, input.UpdatedBy, shareLink.ResourceType, shareLink.ResourceID, requiredPermission)
		if err != nil {
			return nil, err
		}
		if !hasPermission {
			return nil, apperror.NewForbiddenError("you do not have permission to update this share link")
		}
	}

	// 4. フィールド更新
	if input.ExpiresAt != nil {
		shareLink.UpdateExpiry(input.ExpiresAt)
	}
	if input.MaxAccessCount != nil {
		shareLink.UpdateMaxAccessCount(input.MaxAccessCount)
	}
	if input.Password != nil {
		if *input.Password == "" {
			shareLink.UpdatePassword("")
		} else {
			hash, err := bcrypt.GenerateFromPassword([]byte(*input.Password), 12)
			if err != nil {
				return nil, apperror.NewInternalError(err)
			}
			shareLink.UpdatePassword(string(hash))
		}
	}

	// 5. 保存
	if err := c.shareLinkRepo.Update(ctx, shareLink); err != nil {
		return nil, err
	}

	return &UpdateShareLinkOutput{ShareLink: shareLink}, nil
}
