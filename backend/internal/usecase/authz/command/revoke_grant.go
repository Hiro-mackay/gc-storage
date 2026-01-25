package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// RevokeGrantInput は権限取り消しの入力を定義します
type RevokeGrantInput struct {
	GrantID   uuid.UUID
	RevokedBy uuid.UUID
}

// RevokeGrantOutput は権限取り消しの出力を定義します
type RevokeGrantOutput struct {
	RevokedGrantID uuid.UUID
}

// RevokeGrantCommand は権限取り消しコマンドです
type RevokeGrantCommand struct {
	permissionGrantRepo authz.PermissionGrantRepository
	permissionResolver  authz.PermissionResolver
}

// NewRevokeGrantCommand は新しいRevokeGrantCommandを作成します
func NewRevokeGrantCommand(
	permissionGrantRepo authz.PermissionGrantRepository,
	permissionResolver authz.PermissionResolver,
) *RevokeGrantCommand {
	return &RevokeGrantCommand{
		permissionGrantRepo: permissionGrantRepo,
		permissionResolver:  permissionResolver,
	}
}

// Execute は権限取り消しを実行します
func (c *RevokeGrantCommand) Execute(ctx context.Context, input RevokeGrantInput) (*RevokeGrantOutput, error) {
	// 1. 権限付与を取得
	grant, err := c.permissionGrantRepo.FindByID(ctx, input.GrantID)
	if err != nil {
		return nil, err
	}

	// 2. 取り消し者が取り消し可能か確認
	// 取り消し者がこのロールを付与可能な権限を持っているか確認
	canGrant, err := c.permissionResolver.CanGrantRole(ctx, input.RevokedBy, grant.ResourceType, grant.ResourceID, grant.Role)
	if err != nil {
		return nil, err
	}
	if !canGrant {
		return nil, apperror.NewForbiddenError("you do not have permission to revoke this grant")
	}

	// 3. 権限付与を削除
	if err := c.permissionGrantRepo.Delete(ctx, input.GrantID); err != nil {
		return nil, err
	}

	return &RevokeGrantOutput{RevokedGrantID: input.GrantID}, nil
}
