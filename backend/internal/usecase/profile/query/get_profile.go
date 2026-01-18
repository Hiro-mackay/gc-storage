package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// GetProfileInput はプロファイル取得の入力を定義します
type GetProfileInput struct {
	UserID uuid.UUID
}

// GetProfileOutput はプロファイル取得の出力を定義します
type GetProfileOutput struct {
	Profile *entity.UserProfile
	User    *entity.User
}

// GetProfileQuery はプロファイル取得クエリです
type GetProfileQuery struct {
	profileRepo repository.UserProfileRepository
	userRepo    repository.UserRepository
}

// NewGetProfileQuery は新しいGetProfileQueryを作成します
func NewGetProfileQuery(
	profileRepo repository.UserProfileRepository,
	userRepo repository.UserRepository,
) *GetProfileQuery {
	return &GetProfileQuery{
		profileRepo: profileRepo,
		userRepo:    userRepo,
	}
}

// Execute はプロファイル取得を実行します
func (q *GetProfileQuery) Execute(ctx context.Context, input GetProfileInput) (*GetProfileOutput, error) {
	// ユーザーを取得
	user, err := q.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, apperror.NewNotFoundError("user")
	}

	// プロファイルを取得
	profile, err := q.profileRepo.FindByUserID(ctx, input.UserID)
	if err != nil {
		// プロファイルが存在しない場合はデフォルトを返す
		if apperror.IsNotFound(err) {
			profile = entity.NewUserProfile(input.UserID)
		} else {
			return nil, err
		}
	}

	return &GetProfileOutput{
		Profile: profile,
		User:    user,
	}, nil
}
