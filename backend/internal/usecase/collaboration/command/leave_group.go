package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// LeaveGroupInput はグループ退出の入力を定義します
type LeaveGroupInput struct {
	GroupID uuid.UUID
	UserID  uuid.UUID
}

// LeaveGroupOutput はグループ退出の出力を定義します
type LeaveGroupOutput struct {
	LeftGroupID uuid.UUID
}

// LeaveGroupCommand はグループ退出コマンドです
type LeaveGroupCommand struct {
	groupRepo      repository.GroupRepository
	membershipRepo repository.MembershipRepository
}

// NewLeaveGroupCommand は新しいLeaveGroupCommandを作成します
func NewLeaveGroupCommand(
	groupRepo repository.GroupRepository,
	membershipRepo repository.MembershipRepository,
) *LeaveGroupCommand {
	return &LeaveGroupCommand{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
	}
}

// Execute はグループ退出を実行します
func (c *LeaveGroupCommand) Execute(ctx context.Context, input LeaveGroupInput) (*LeaveGroupOutput, error) {
	// 1. グループの存在確認
	_, err := c.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	// 2. メンバーシップの取得
	membership, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.UserID)
	if err != nil {
		return nil, err
	}

	// 3. 脱退可能か確認（グループ、ユーザー、ロールの検証）
	if !membership.CanLeave(input.GroupID, input.UserID) {
		return nil, apperror.NewForbiddenError("cannot leave the group. Owner must transfer ownership first.")
	}

	// 4. メンバーシップを削除
	if err := c.membershipRepo.Delete(ctx, membership.ID); err != nil {
		return nil, err
	}

	return &LeaveGroupOutput{LeftGroupID: input.GroupID}, nil
}
