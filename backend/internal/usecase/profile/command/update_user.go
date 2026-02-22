package command

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// UpdateUserInput はユーザー更新の入力を定義します
type UpdateUserInput struct {
	UserID uuid.UUID
	Name   *string
}

// UpdateUserOutput はユーザー更新の出力を定義します
type UpdateUserOutput struct {
	User *entity.User
}

// UpdateUserCommand はユーザー情報更新コマンドです
type UpdateUserCommand struct {
	userRepo repository.UserRepository
}

// NewUpdateUserCommand は新しいUpdateUserCommandを作成します
func NewUpdateUserCommand(userRepo repository.UserRepository) *UpdateUserCommand {
	return &UpdateUserCommand{userRepo: userRepo}
}

// Execute はユーザー情報更新を実行します
func (c *UpdateUserCommand) Execute(ctx context.Context, input UpdateUserInput) (*UpdateUserOutput, error) {
	if input.Name != nil {
		if len(*input.Name) == 0 {
			return nil, apperror.NewValidationError("name cannot be empty", nil)
		}
		if len([]rune(*input.Name)) > 100 {
			return nil, apperror.NewValidationError("name must not exceed 100 characters", nil)
		}
	}

	user, err := c.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, apperror.NewNotFoundError("user")
	}

	if input.Name != nil {
		user.Name = *input.Name
		user.UpdatedAt = time.Now()
		if err := c.userRepo.Update(ctx, user); err != nil {
			return nil, apperror.NewInternalError(err)
		}
	}

	return &UpdateUserOutput{User: user}, nil
}
