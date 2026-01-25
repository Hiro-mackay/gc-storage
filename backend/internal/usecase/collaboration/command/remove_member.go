package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// RemoveMemberInput はメンバー削除の入力を定義します
type RemoveMemberInput struct {
	GroupID     uuid.UUID
	TargetUserID uuid.UUID
	RemovedBy   uuid.UUID
}

// RemoveMemberOutput はメンバー削除の出力を定義します
type RemoveMemberOutput struct {
	RemovedUserID uuid.UUID
}

// RemoveMemberCommand はメンバー削除コマンドです
type RemoveMemberCommand struct {
	groupRepo      repository.GroupRepository
	membershipRepo repository.MembershipRepository
}

// NewRemoveMemberCommand は新しいRemoveMemberCommandを作成します
func NewRemoveMemberCommand(
	groupRepo repository.GroupRepository,
	membershipRepo repository.MembershipRepository,
) *RemoveMemberCommand {
	return &RemoveMemberCommand{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
	}
}

// Execute はメンバー削除を実行します
func (c *RemoveMemberCommand) Execute(ctx context.Context, input RemoveMemberInput) (*RemoveMemberOutput, error) {
	// 1. グループの存在確認
	_, err := c.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	// 2. 操作者の権限チェック
	removerMembership, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.RemovedBy)
	if err != nil {
		return nil, apperror.NewForbiddenError("you are not a member of this group")
	}
	if !removerMembership.CanManageMembers() {
		return nil, apperror.NewForbiddenError("you do not have permission to remove members")
	}

	// 3. 対象メンバーの取得
	targetMembership, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.TargetUserID)
	if err != nil {
		return nil, err
	}

	// 4. オーナーは削除不可
	if targetMembership.IsOwner() {
		return nil, apperror.NewForbiddenError("cannot remove the group owner")
	}

	// 5. 自分より高いロールは削除不可
	if targetMembership.Role.Level() >= removerMembership.Role.Level() {
		return nil, apperror.NewForbiddenError("you cannot remove a member with equal or higher role")
	}

	// 6. メンバーシップを削除
	if err := c.membershipRepo.Delete(ctx, targetMembership.ID); err != nil {
		return nil, err
	}

	return &RemoveMemberOutput{RemovedUserID: input.TargetUserID}, nil
}
