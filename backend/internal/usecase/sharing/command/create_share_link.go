package command

import (
	"context"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// CreateShareLinkInput は共有リンク作成の入力を定義します
type CreateShareLinkInput struct {
	ResourceType   string
	ResourceID     uuid.UUID
	CreatedBy      uuid.UUID
	Permission     string
	Password       string     // optional
	ExpiresAt      *time.Time // optional
	MaxAccessCount *int       // optional
}

// CreateShareLinkOutput は共有リンク作成の出力を定義します
type CreateShareLinkOutput struct {
	ShareLink *entity.ShareLink
}

// CreateShareLinkCommand は共有リンク作成コマンドです
type CreateShareLinkCommand struct {
	shareLinkRepo      repository.ShareLinkRepository
	permissionResolver authz.PermissionResolver
}

// NewCreateShareLinkCommand は新しいCreateShareLinkCommandを作成します
func NewCreateShareLinkCommand(
	shareLinkRepo repository.ShareLinkRepository,
	permissionResolver authz.PermissionResolver,
) *CreateShareLinkCommand {
	return &CreateShareLinkCommand{
		shareLinkRepo:      shareLinkRepo,
		permissionResolver: permissionResolver,
	}
}

// Execute は共有リンク作成を実行します
func (c *CreateShareLinkCommand) Execute(ctx context.Context, input CreateShareLinkInput) (*CreateShareLinkOutput, error) {
	// 1. リソースタイプのバリデーション
	resourceType, err := authz.NewResourceType(input.ResourceType)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 2. 権限のバリデーション
	permission, err := valueobject.NewSharePermission(input.Permission)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 3. ユーザーが共有権限を持っているか確認
	var requiredPermission authz.Permission
	if resourceType == authz.ResourceTypeFile {
		requiredPermission = authz.PermFileShare
	} else {
		requiredPermission = authz.PermFolderShare
	}

	hasPermission, err := c.permissionResolver.HasPermission(ctx, input.CreatedBy, resourceType, input.ResourceID, requiredPermission)
	if err != nil {
		return nil, err
	}
	if !hasPermission {
		return nil, apperror.NewForbiddenError("you do not have permission to share this resource")
	}

	// 4. パスワードのハッシュ化（設定されている場合）
	var passwordHash string
	if input.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, apperror.NewInternalError(err)
		}
		passwordHash = string(hash)
	}

	// 5. 共有リンクを作成
	shareLink, err := entity.NewShareLink(
		resourceType,
		input.ResourceID,
		input.CreatedBy,
		permission,
		passwordHash,
		input.ExpiresAt,
		input.MaxAccessCount,
	)
	if err != nil {
		return nil, apperror.NewInternalError(err)
	}

	// 6. 保存
	if err := c.shareLinkRepo.Create(ctx, shareLink); err != nil {
		return nil, err
	}

	return &CreateShareLinkOutput{ShareLink: shareLink}, nil
}
