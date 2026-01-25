package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// RevokeShareLinkInput は共有リンク無効化の入力を定義します
type RevokeShareLinkInput struct {
	ShareLinkID uuid.UUID
	RevokedBy   uuid.UUID
}

// RevokeShareLinkOutput は共有リンク無効化の出力を定義します
type RevokeShareLinkOutput struct {
	RevokedShareLinkID uuid.UUID
}

// RevokeShareLinkCommand は共有リンク無効化コマンドです
type RevokeShareLinkCommand struct {
	shareLinkRepo      repository.ShareLinkRepository
	permissionResolver authz.PermissionResolver
}

// NewRevokeShareLinkCommand は新しいRevokeShareLinkCommandを作成します
func NewRevokeShareLinkCommand(
	shareLinkRepo repository.ShareLinkRepository,
	permissionResolver authz.PermissionResolver,
) *RevokeShareLinkCommand {
	return &RevokeShareLinkCommand{
		shareLinkRepo:      shareLinkRepo,
		permissionResolver: permissionResolver,
	}
}

// Execute は共有リンク無効化を実行します
func (c *RevokeShareLinkCommand) Execute(ctx context.Context, input RevokeShareLinkInput) (*RevokeShareLinkOutput, error) {
	// 1. 共有リンクを取得
	shareLink, err := c.shareLinkRepo.FindByID(ctx, input.ShareLinkID)
	if err != nil {
		return nil, err
	}

	// 2. 既に無効化されているかチェック
	if !shareLink.IsActive() {
		return nil, apperror.NewValidationError("share link is already revoked or expired", nil)
	}

	// 3. 作成者または共有権限を持つユーザーのみ無効化可能
	if !shareLink.IsCreatedBy(input.RevokedBy) {
		var requiredPermission authz.Permission
		if shareLink.ResourceType == authz.ResourceTypeFile {
			requiredPermission = authz.PermFileShare
		} else {
			requiredPermission = authz.PermFolderShare
		}

		hasPermission, err := c.permissionResolver.HasPermission(ctx, input.RevokedBy, shareLink.ResourceType, shareLink.ResourceID, requiredPermission)
		if err != nil {
			return nil, err
		}
		if !hasPermission {
			return nil, apperror.NewForbiddenError("you do not have permission to revoke this share link")
		}
	}

	// 4. 共有リンクを無効化
	shareLink.Revoke()
	if err := c.shareLinkRepo.Update(ctx, shareLink); err != nil {
		return nil, err
	}

	return &RevokeShareLinkOutput{RevokedShareLinkID: input.ShareLinkID}, nil
}
