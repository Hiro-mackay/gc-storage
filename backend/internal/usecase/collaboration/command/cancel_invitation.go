package command

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// CancelInvitationInput は招待キャンセルの入力を定義します
type CancelInvitationInput struct {
	InvitationID uuid.UUID
	GroupID      uuid.UUID
	CancelledBy  uuid.UUID
}

// CancelInvitationCommand は招待キャンセルコマンドです
type CancelInvitationCommand struct {
	invitationRepo repository.InvitationRepository
	membershipRepo repository.MembershipRepository
	groupRepo      repository.GroupRepository
}

// NewCancelInvitationCommand は新しいCancelInvitationCommandを作成します
func NewCancelInvitationCommand(
	invitationRepo repository.InvitationRepository,
	membershipRepo repository.MembershipRepository,
	groupRepo repository.GroupRepository,
) *CancelInvitationCommand {
	return &CancelInvitationCommand{
		invitationRepo: invitationRepo,
		membershipRepo: membershipRepo,
		groupRepo:      groupRepo,
	}
}

// Execute は招待キャンセルを実行します
func (c *CancelInvitationCommand) Execute(ctx context.Context, input CancelInvitationInput) error {
	// 1. グループの存在確認
	_, err := c.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return apperror.NewNotFoundError("group")
	}

	// 2. 操作者のメンバーシップ確認（owner only）
	membership, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.CancelledBy)
	if err != nil {
		return apperror.NewForbiddenError("not a member of this group")
	}
	if !membership.CanManageMembers() {
		return apperror.NewForbiddenError("only owners can cancel invitations")
	}

	// 3. 招待の取得
	invitation, err := c.invitationRepo.FindByID(ctx, input.InvitationID)
	if err != nil {
		return apperror.NewNotFoundError("invitation")
	}

	// 4. 招待がこのグループのものか確認
	if !invitation.IsForGroup(input.GroupID) {
		return apperror.NewForbiddenError("invitation does not belong to this group")
	}

	// 5. 招待のキャンセル
	if err := invitation.Cancel(); err != nil {
		if errors.Is(err, entity.ErrInvitationNotPending) {
			return apperror.NewValidationError("invitation is not pending", nil)
		}
		return err
	}

	// 6. 更新を保存
	return c.invitationRepo.Update(ctx, invitation)
}
