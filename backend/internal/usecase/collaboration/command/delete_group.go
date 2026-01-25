package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// DeleteGroupInput はグループ削除の入力を定義します
type DeleteGroupInput struct {
	GroupID   uuid.UUID
	DeletedBy uuid.UUID
}

// DeleteGroupOutput はグループ削除の出力を定義します
type DeleteGroupOutput struct {
	DeletedGroupID uuid.UUID
}

// DeleteGroupCommand はグループ削除コマンドです
type DeleteGroupCommand struct {
	groupRepo      repository.GroupRepository
	membershipRepo repository.MembershipRepository
	invitationRepo repository.InvitationRepository
	txManager      repository.TransactionManager
}

// NewDeleteGroupCommand は新しいDeleteGroupCommandを作成します
func NewDeleteGroupCommand(
	groupRepo repository.GroupRepository,
	membershipRepo repository.MembershipRepository,
	invitationRepo repository.InvitationRepository,
	txManager repository.TransactionManager,
) *DeleteGroupCommand {
	return &DeleteGroupCommand{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
		invitationRepo: invitationRepo,
		txManager:      txManager,
	}
}

// Execute はグループ削除を実行します
func (c *DeleteGroupCommand) Execute(ctx context.Context, input DeleteGroupInput) (*DeleteGroupOutput, error) {
	// 1. グループの取得と権限確認
	group, err := c.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}
	if !group.IsOwnedBy(input.DeletedBy) {
		return nil, apperror.NewForbiddenError("only the owner can delete the group")
	}

	// 2. トランザクションで削除処理
	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// 招待を削除
		if err := c.invitationRepo.DeleteByGroupID(ctx, input.GroupID); err != nil {
			return err
		}

		// メンバーシップを削除
		if err := c.membershipRepo.DeleteByGroupID(ctx, input.GroupID); err != nil {
			return err
		}

		// グループを論理削除
		if err := c.groupRepo.Delete(ctx, input.GroupID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &DeleteGroupOutput{DeletedGroupID: input.GroupID}, nil
}
