package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database/sqlcgen"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// UserProfileRepository はユーザープロファイルリポジトリの実装です
type UserProfileRepository struct {
	*database.BaseRepository
}

// NewUserProfileRepository は新しいUserProfileRepositoryを作成します
func NewUserProfileRepository(txManager *database.TxManager) *UserProfileRepository {
	return &UserProfileRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create はユーザープロファイルを作成します
func (r *UserProfileRepository) Create(ctx context.Context, profile *entity.UserProfile) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	notificationPrefsJSON, err := json.Marshal(profile.NotificationPreferences)
	if err != nil {
		return apperror.NewInternalError(err)
	}

	var avatarURL *string
	if profile.AvatarURL != "" {
		avatarURL = &profile.AvatarURL
	}

	var bio *string
	if profile.Bio != "" {
		bio = &profile.Bio
	}

	var timezone *string
	if profile.Timezone != "" {
		timezone = &profile.Timezone
	}

	var locale *string
	if profile.Locale != "" {
		locale = &profile.Locale
	}

	var theme *string
	if profile.Theme != "" {
		theme = &profile.Theme
	}

	now := time.Now()
	_, err = queries.CreateUserProfile(ctx, sqlcgen.CreateUserProfileParams{
		ID:                      profile.ID,
		UserID:                  profile.UserID,
		AvatarUrl:               avatarURL,
		Bio:                     bio,
		Timezone:                timezone,
		Locale:                  locale,
		Theme:                   theme,
		NotificationPreferences: notificationPrefsJSON,
		CreatedAt:               now,
		UpdatedAt:               now,
	})

	return r.HandleError(err)
}

// Update はユーザープロファイルを更新します
func (r *UserProfileRepository) Update(ctx context.Context, profile *entity.UserProfile) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	notificationPrefsJSON, err := json.Marshal(profile.NotificationPreferences)
	if err != nil {
		return apperror.NewInternalError(err)
	}

	var avatarURL *string
	if profile.AvatarURL != "" {
		avatarURL = &profile.AvatarURL
	}

	var bio *string
	if profile.Bio != "" {
		bio = &profile.Bio
	}

	var timezone *string
	if profile.Timezone != "" {
		timezone = &profile.Timezone
	}

	var locale *string
	if profile.Locale != "" {
		locale = &profile.Locale
	}

	var theme *string
	if profile.Theme != "" {
		theme = &profile.Theme
	}

	_, err = queries.UpdateUserProfile(ctx, sqlcgen.UpdateUserProfileParams{
		UserID:                  profile.UserID,
		AvatarUrl:               avatarURL,
		Bio:                     bio,
		Timezone:                timezone,
		Locale:                  locale,
		Theme:                   theme,
		NotificationPreferences: notificationPrefsJSON,
	})

	return r.HandleError(err)
}

// FindByUserID はユーザーIDでプロファイルを検索します
func (r *UserProfileRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*entity.UserProfile, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetUserProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("user profile")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row)
}

// Upsert はプロファイルを作成または更新します
func (r *UserProfileRepository) Upsert(ctx context.Context, profile *entity.UserProfile) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	notificationPrefsJSON, err := json.Marshal(profile.NotificationPreferences)
	if err != nil {
		return apperror.NewInternalError(err)
	}

	var avatarURL *string
	if profile.AvatarURL != "" {
		avatarURL = &profile.AvatarURL
	}

	var bio *string
	if profile.Bio != "" {
		bio = &profile.Bio
	}

	var timezone *string
	if profile.Timezone != "" {
		timezone = &profile.Timezone
	}

	var locale *string
	if profile.Locale != "" {
		locale = &profile.Locale
	}

	var theme *string
	if profile.Theme != "" {
		theme = &profile.Theme
	}

	_, err = queries.UpsertUserProfile(ctx, sqlcgen.UpsertUserProfileParams{
		ID:                      profile.ID,
		UserID:                  profile.UserID,
		AvatarUrl:               avatarURL,
		Bio:                     bio,
		Locale:                  locale,
		Timezone:                timezone,
		Theme:                   theme,
		NotificationPreferences: notificationPrefsJSON,
	})

	return r.HandleError(err)
}

// toEntity はsqlcgen.UserProfileをentity.UserProfileに変換します
func (r *UserProfileRepository) toEntity(row sqlcgen.UserProfile) (*entity.UserProfile, error) {
	profile := &entity.UserProfile{
		ID:        row.ID,
		UserID:    row.UserID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}

	if row.AvatarUrl != nil {
		profile.AvatarURL = *row.AvatarUrl
	}

	if row.Bio != nil {
		profile.Bio = *row.Bio
	}

	if row.Timezone != nil {
		profile.Timezone = *row.Timezone
	}

	if row.Locale != nil {
		profile.Locale = *row.Locale
	}

	if row.Theme != nil {
		profile.Theme = *row.Theme
	}

	if row.NotificationPreferences != nil {
		if err := json.Unmarshal(row.NotificationPreferences, &profile.NotificationPreferences); err != nil {
			return nil, apperror.NewInternalError(err)
		}
	}

	return profile, nil
}

// インターフェースの実装を保証
var _ repository.UserProfileRepository = (*UserProfileRepository)(nil)
