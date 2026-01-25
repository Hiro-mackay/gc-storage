package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ChangeRoleInput はロール変更の入力を定義します
type ChangeRoleInput struct {
	GroupID      uuid.UUID
	TargetUserID uuid.UUID
	NewRole      string
	ChangedBy    uuid.UUID
}

// ChangeRoleOutput はロール変更の出力を定義します
type ChangeRoleOutput struct {
	Membership *entity.Membership
}

// ChangeRoleCommand はロール変更コマンドです
type ChangeRoleCommand struct {
	groupRepo      repository.GroupRepository
	membershipRepo repository.MembershipRepository
}

// NewChangeRoleCommand は新しいChangeRoleCommandを作成します
func NewChangeRoleCommand(
	groupRepo repository.GroupRepository,
	membershipRepo repository.MembershipRepository,
) *ChangeRoleCommand {
	return &ChangeRoleCommand{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
	}
}

// Execute はロール変更を実行します
func (c *ChangeRoleCommand) Execute(ctx context.Context, input ChangeRoleInput) (*ChangeRoleOutput, error) {
	// 1. ロールのバリデーション
	newRole, err := valueobject.NewGroupRole(input.NewRole)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}
	// Ownerへの変更は所有権譲渡を使用
	if newRole.IsOwner() {
		return nil, apperror.NewValidationError("use transfer ownership to assign owner role", nil)
	}

	// 2. グループの存在確認
	_, err = c.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	// 3. 操作者の権限チェック
	changerMembership, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.ChangedBy)
	if err != nil {
		return nil, apperror.NewForbiddenError("you are not a member of this group")
	}
	if !changerMembership.CanManageMembers() {
		return nil, apperror.NewForbiddenError("you do not have permission to change roles")
	}

	// 4. 対象メンバーの取得
	targetMembership, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.TargetUserID)
	if err != nil {
		return nil, err
	}

	// 5. オーナーのロールは変更不可
	if targetMembership.IsOwner() {
		return nil, apperror.NewForbiddenError("cannot change owner's role")
	}

	// 6. 自分と同等以上のロールは変更不可
	if targetMembership.Role.Level() >= changerMembership.Role.Level() {
		return nil, apperror.NewForbiddenError("you cannot change role of a member with equal or higher role")
	}

	// 7. 新しいロールが自分より高い場合は不可
	if !changerMembership.CanChangeRoleTo(newRole) {
		return nil, apperror.NewForbiddenError("you cannot assign a role higher than your own")
	}

	// 8. ロールを変更
	targetMembership.ChangeRole(newRole)
	if err := c.membershipRepo.Update(ctx, targetMembership); err != nil {
		return nil, err
	}

	return &ChangeRoleOutput{Membership: targetMembership}, nil
}
