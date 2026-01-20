package command

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// UpdateProfileInput はプロファイル更新の入力を定義します
type UpdateProfileInput struct {
	UserID                  uuid.UUID
	AvatarURL               *string
	Bio                     *string
	Locale                  *string
	Timezone                *string
	Theme                   *string
	NotificationPreferences *entity.NotificationPreferences
}

// UpdateProfileOutput はプロファイル更新の出力を定義します
type UpdateProfileOutput struct {
	Profile *entity.UserProfile
}

// UpdateProfileCommand はプロファイル更新コマンドです
type UpdateProfileCommand struct {
	profileRepo repository.UserProfileRepository
	userRepo    repository.UserRepository
}

// NewUpdateProfileCommand は新しいUpdateProfileCommandを作成します
func NewUpdateProfileCommand(
	profileRepo repository.UserProfileRepository,
	userRepo repository.UserRepository,
) *UpdateProfileCommand {
	return &UpdateProfileCommand{
		profileRepo: profileRepo,
		userRepo:    userRepo,
	}
}

// Execute はプロファイル更新を実行します
func (c *UpdateProfileCommand) Execute(ctx context.Context, input UpdateProfileInput) (*UpdateProfileOutput, error) {
	// ユーザーの存在確認
	_, err := c.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, apperror.NewNotFoundError("user")
	}

	// 既存プロファイルを取得（なければデフォルト作成）
	profile, err := c.profileRepo.FindByUserID(ctx, input.UserID)
	if err != nil {
		if apperror.IsNotFound(err) {
			profile = entity.NewUserProfile(input.UserID)
		} else {
			return nil, err
		}
	}

	// フィールドを更新
	if input.AvatarURL != nil {
		profile.AvatarURL = *input.AvatarURL
	}
	if input.Bio != nil {
		profile.Bio = *input.Bio
		// bioの長さ検証
		if !profile.ValidateBio() {
			return nil, apperror.NewValidationError("bio must not exceed 500 characters", nil)
		}
	}
	if input.Locale != nil {
		profile.Locale = *input.Locale
	}
	if input.Timezone != nil {
		profile.Timezone = *input.Timezone
	}
	if input.Theme != nil {
		profile.SetTheme(*input.Theme)
	}
	if input.NotificationPreferences != nil {
		profile.SetNotificationPreferences(*input.NotificationPreferences)
	}

	profile.UpdatedAt = time.Now()

	// Upsertで保存
	if err := c.profileRepo.Upsert(ctx, profile); err != nil {
		return nil, apperror.NewInternalError(err)
	}

	return &UpdateProfileOutput{Profile: profile}, nil
}
