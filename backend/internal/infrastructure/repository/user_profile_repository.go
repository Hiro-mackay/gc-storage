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

	settingsJSON, err := json.Marshal(profile.Settings)
	if err != nil {
		return apperror.NewInternalError(err)
	}

	var displayName *string
	if profile.DisplayName != "" {
		displayName = &profile.DisplayName
	}

	var avatarURL *string
	if profile.AvatarURL != "" {
		avatarURL = &profile.AvatarURL
	}

	var bio *string
	if profile.Bio != "" {
		bio = &profile.Bio
	}

	_, err = queries.CreateUserProfile(ctx, sqlcgen.CreateUserProfileParams{
		UserID:      profile.UserID,
		DisplayName: displayName,
		AvatarUrl:   avatarURL,
		Bio:         bio,
		Locale:      profile.Locale,
		Timezone:    profile.Timezone,
		Settings:    settingsJSON,
		UpdatedAt:   time.Now(),
	})

	return r.HandleError(err)
}

// Update はユーザープロファイルを更新します
func (r *UserProfileRepository) Update(ctx context.Context, profile *entity.UserProfile) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	settingsJSON, err := json.Marshal(profile.Settings)
	if err != nil {
		return apperror.NewInternalError(err)
	}

	var displayName *string
	if profile.DisplayName != "" {
		displayName = &profile.DisplayName
	}

	var avatarURL *string
	if profile.AvatarURL != "" {
		avatarURL = &profile.AvatarURL
	}

	var bio *string
	if profile.Bio != "" {
		bio = &profile.Bio
	}

	_, err = queries.UpdateUserProfile(ctx, sqlcgen.UpdateUserProfileParams{
		UserID:      profile.UserID,
		DisplayName: displayName,
		AvatarUrl:   avatarURL,
		Bio:         bio,
		Locale:      &profile.Locale,
		Timezone:    &profile.Timezone,
		Settings:    settingsJSON,
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

	settingsJSON, err := json.Marshal(profile.Settings)
	if err != nil {
		return apperror.NewInternalError(err)
	}

	var displayName *string
	if profile.DisplayName != "" {
		displayName = &profile.DisplayName
	}

	var avatarURL *string
	if profile.AvatarURL != "" {
		avatarURL = &profile.AvatarURL
	}

	var bio *string
	if profile.Bio != "" {
		bio = &profile.Bio
	}

	_, err = queries.UpsertUserProfile(ctx, sqlcgen.UpsertUserProfileParams{
		UserID:      profile.UserID,
		DisplayName: displayName,
		AvatarUrl:   avatarURL,
		Bio:         bio,
		Locale:      profile.Locale,
		Timezone:    profile.Timezone,
		Settings:    settingsJSON,
	})

	return r.HandleError(err)
}

// toEntity はsqlcgen.UserProfileをentity.UserProfileに変換します
func (r *UserProfileRepository) toEntity(row sqlcgen.UserProfile) (*entity.UserProfile, error) {
	profile := &entity.UserProfile{
		UserID:    row.UserID,
		Locale:    row.Locale,
		Timezone:  row.Timezone,
		UpdatedAt: row.UpdatedAt,
	}

	if row.DisplayName != nil {
		profile.DisplayName = *row.DisplayName
	}

	if row.AvatarUrl != nil {
		profile.AvatarURL = *row.AvatarUrl
	}

	if row.Bio != nil {
		profile.Bio = *row.Bio
	}

	if row.Settings != nil {
		if err := json.Unmarshal(row.Settings, &profile.Settings); err != nil {
			return nil, apperror.NewInternalError(err)
		}
	}

	return profile, nil
}

// インターフェースの実装を保証
var _ repository.UserProfileRepository = (*UserProfileRepository)(nil)
