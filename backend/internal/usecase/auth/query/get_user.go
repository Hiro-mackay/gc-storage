package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// GetUserInput はユーザー取得の入力を定義します
type GetUserInput struct {
	UserID uuid.UUID
}

// GetUserOutput はユーザー取得の出力を定義します
type GetUserOutput struct {
	User *entity.User
}

// GetUserQuery はユーザー取得クエリです
type GetUserQuery struct {
	userRepo repository.UserRepository
}

// NewGetUserQuery は新しいGetUserQueryを作成します
func NewGetUserQuery(userRepo repository.UserRepository) *GetUserQuery {
	return &GetUserQuery{userRepo: userRepo}
}

// Execute はユーザー取得を実行します
func (q *GetUserQuery) Execute(ctx context.Context, input GetUserInput) (*GetUserOutput, error) {
	user, err := q.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, apperror.NewNotFoundError("user")
	}
	return &GetUserOutput{User: user}, nil
}
