package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// TransferOwnershipInput は所有権譲渡の入力を定義します
type TransferOwnershipInput struct {
	GroupID      uuid.UUID
	NewOwnerID   uuid.UUID
	CurrentOwnerID uuid.UUID
}

// TransferOwnershipOutput は所有権譲渡の出力を定義します
type TransferOwnershipOutput struct {
	Group            *entity.Group
	NewOwnerMembership *entity.Membership
	OldOwnerMembership *entity.Membership
}

// TransferOwnershipCommand は所有権譲渡コマンドです
type TransferOwnershipCommand struct {
	groupRepo      repository.GroupRepository
	membershipRepo repository.MembershipRepository
	txManager      repository.TransactionManager
}

// NewTransferOwnershipCommand は新しいTransferOwnershipCommandを作成します
func NewTransferOwnershipCommand(
	groupRepo repository.GroupRepository,
	membershipRepo repository.MembershipRepository,
	txManager repository.TransactionManager,
) *TransferOwnershipCommand {
	return &TransferOwnershipCommand{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
		txManager:      txManager,
	}
}

// Execute は所有権譲渡を実行します
func (c *TransferOwnershipCommand) Execute(ctx context.Context, input TransferOwnershipInput) (*TransferOwnershipOutput, error) {
	// 1. グループの取得と権限確認
	group, err := c.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}
	if !group.IsOwnedBy(input.CurrentOwnerID) {
		return nil, apperror.NewForbiddenError("only the owner can transfer ownership")
	}

	// 2. 新しいオーナーのメンバーシップ確認
	newOwnerMembership, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.NewOwnerID)
	if err != nil {
		return nil, apperror.NewValidationError("new owner must be a member of the group", nil)
	}

	// 3. 現在のオーナーのメンバーシップ取得
	currentOwnerMembership, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.CurrentOwnerID)
	if err != nil {
		return nil, err
	}

	// 4. 自分自身への譲渡は不可
	if input.CurrentOwnerID == input.NewOwnerID {
		return nil, apperror.NewValidationError("cannot transfer ownership to yourself", nil)
	}

	// 5. トランザクションで所有権を譲渡
	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// グループのオーナーを変更
		group.TransferOwnership(input.NewOwnerID)
		if err := c.groupRepo.Update(ctx, group); err != nil {
			return err
		}

		// 新しいオーナーのロールをOwnerに変更
		newOwnerMembership.ChangeRole(valueobject.GroupRoleOwner)
		if err := c.membershipRepo.Update(ctx, newOwnerMembership); err != nil {
			return err
		}

		// 元オーナーのロールをContributorに変更
		currentOwnerMembership.ChangeRole(valueobject.GroupRoleContributor)
		if err := c.membershipRepo.Update(ctx, currentOwnerMembership); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &TransferOwnershipOutput{
		Group:              group,
		NewOwnerMembership: newOwnerMembership,
		OldOwnerMembership: currentOwnerMembership,
	}, nil
}
