package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// GrantRoleInput は権限付与の入力を定義します
type GrantRoleInput struct {
	ResourceType string
	ResourceID   uuid.UUID
	GranteeType  string
	GranteeID    uuid.UUID
	Role         string
	GrantedBy    uuid.UUID
}

// GrantRoleOutput は権限付与の出力を定義します
type GrantRoleOutput struct {
	Grant *authz.PermissionGrant
}

// GrantRoleCommand は権限付与コマンドです
type GrantRoleCommand struct {
	permissionGrantRepo authz.PermissionGrantRepository
	permissionResolver  authz.PermissionResolver
}

// NewGrantRoleCommand は新しいGrantRoleCommandを作成します
func NewGrantRoleCommand(
	permissionGrantRepo authz.PermissionGrantRepository,
	permissionResolver authz.PermissionResolver,
) *GrantRoleCommand {
	return &GrantRoleCommand{
		permissionGrantRepo: permissionGrantRepo,
		permissionResolver:  permissionResolver,
	}
}

// Execute は権限付与を実行します
func (c *GrantRoleCommand) Execute(ctx context.Context, input GrantRoleInput) (*GrantRoleOutput, error) {
	// 1. リソースタイプのバリデーション
	resourceType, err := authz.NewResourceType(input.ResourceType)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 2. 付与対象タイプのバリデーション
	granteeType, err := authz.NewGranteeType(input.GranteeType)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 3. ロールのバリデーション
	role, err := authz.NewRole(input.Role)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 4. Ownerロールは直接付与不可
	if role == authz.RoleOwner {
		return nil, apperror.NewValidationError("owner role cannot be granted directly. Use transfer ownership instead.", nil)
	}

	// 4.5. 自分自身への付与は不可
	if granteeType.IsUser() && input.GranteeID == input.GrantedBy {
		return nil, apperror.NewValidationError("cannot grant a role to yourself", nil)
	}

	// 5. 付与者が指定されたロールを付与可能か確認
	canGrant, err := c.permissionResolver.CanGrantRole(ctx, input.GrantedBy, resourceType, input.ResourceID, role)
	if err != nil {
		return nil, err
	}
	if !canGrant {
		return nil, apperror.NewForbiddenError("you do not have permission to grant this role")
	}

	// 6. 既存の同一権限付与をチェック
	existingGrant, err := c.permissionGrantRepo.FindByResourceGranteeAndRole(ctx, resourceType, input.ResourceID, granteeType, input.GranteeID, role)
	if err == nil && existingGrant != nil {
		return nil, apperror.NewConflictError("this role is already granted to the target")
	}

	// 7. 権限付与を作成
	grant := authz.NewPermissionGrant(resourceType, input.ResourceID, granteeType, input.GranteeID, role, input.GrantedBy)
	if err := c.permissionGrantRepo.Create(ctx, grant); err != nil {
		return nil, err
	}

	return &GrantRoleOutput{Grant: grant}, nil
}
