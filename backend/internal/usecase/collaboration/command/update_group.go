package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// UpdateGroupInput はグループ更新の入力を定義します
type UpdateGroupInput struct {
	GroupID     uuid.UUID
	Name        *string
	Description *string
	UpdatedBy   uuid.UUID
}

// UpdateGroupOutput はグループ更新の出力を定義します
type UpdateGroupOutput struct {
	Group *entity.Group
}

// UpdateGroupCommand はグループ更新コマンドです
type UpdateGroupCommand struct {
	groupRepo      repository.GroupRepository
	membershipRepo repository.MembershipRepository
}

// NewUpdateGroupCommand は新しいUpdateGroupCommandを作成します
func NewUpdateGroupCommand(
	groupRepo repository.GroupRepository,
	membershipRepo repository.MembershipRepository,
) *UpdateGroupCommand {
	return &UpdateGroupCommand{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
	}
}

// Execute はグループ更新を実行します
func (c *UpdateGroupCommand) Execute(ctx context.Context, input UpdateGroupInput) (*UpdateGroupOutput, error) {
	// 1. グループの取得
	group, err := c.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, apperror.NewNotFoundError("group")
	}

	// 2. 操作者のメンバーシップ確認（contributor以上）
	membership, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.UpdatedBy)
	if err != nil {
		return nil, apperror.NewForbiddenError("not a member of this group")
	}
	if !membership.CanInvite() { // Contributor以上
		return nil, apperror.NewForbiddenError("insufficient permission to update group")
	}

	// 3. グループ名の更新
	if input.Name != nil {
		newName, err := valueobject.NewGroupName(*input.Name)
		if err != nil {
			return nil, apperror.NewValidationError(err.Error(), nil)
		}
		group.Rename(newName)
	}

	// 4. 説明の更新
	if input.Description != nil {
		group.UpdateDescription(*input.Description)
	}

	// 5. 更新を保存
	if err := c.groupRepo.Update(ctx, group); err != nil {
		return nil, err
	}

	return &UpdateGroupOutput{Group: group}, nil
}
