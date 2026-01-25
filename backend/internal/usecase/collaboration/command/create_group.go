package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// CreateGroupInput はグループ作成の入力を定義します
type CreateGroupInput struct {
	Name        string
	Description string
	OwnerID     uuid.UUID
}

// CreateGroupOutput はグループ作成の出力を定義します
type CreateGroupOutput struct {
	Group      *entity.Group
	Membership *entity.Membership
}

// CreateGroupCommand はグループ作成コマンドです
type CreateGroupCommand struct {
	groupRepo      repository.GroupRepository
	membershipRepo repository.MembershipRepository
	txManager      repository.TransactionManager
}

// NewCreateGroupCommand は新しいCreateGroupCommandを作成します
func NewCreateGroupCommand(
	groupRepo repository.GroupRepository,
	membershipRepo repository.MembershipRepository,
	txManager repository.TransactionManager,
) *CreateGroupCommand {
	return &CreateGroupCommand{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
		txManager:      txManager,
	}
}

// Execute はグループ作成を実行します
func (c *CreateGroupCommand) Execute(ctx context.Context, input CreateGroupInput) (*CreateGroupOutput, error) {
	// 1. グループ名のバリデーション
	groupName, err := valueobject.NewGroupName(input.Name)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 2. グループエンティティの作成
	group := entity.NewGroup(groupName, input.Description, input.OwnerID)

	// 3. オーナーのメンバーシップを作成
	membership := entity.NewMembership(group.ID, input.OwnerID, valueobject.GroupRoleOwner)

	// 4. トランザクションでグループとメンバーシップを作成
	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		if err := c.groupRepo.Create(ctx, group); err != nil {
			return err
		}
		if err := c.membershipRepo.Create(ctx, membership); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &CreateGroupOutput{
		Group:      group,
		Membership: membership,
	}, nil
}
