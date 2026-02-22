package command_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/profile/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type updateProfileTestDeps struct {
	profileRepo *mocks.MockUserProfileRepository
	userRepo    *mocks.MockUserRepository
}

func newUpdateProfileTestDeps(t *testing.T) *updateProfileTestDeps {
	t.Helper()
	return &updateProfileTestDeps{
		profileRepo: mocks.NewMockUserProfileRepository(t),
		userRepo:    mocks.NewMockUserRepository(t),
	}
}

func (d *updateProfileTestDeps) newCommand() *command.UpdateProfileCommand {
	return command.NewUpdateProfileCommand(d.profileRepo, d.userRepo)
}

func TestUpdateProfileCommand_Execute_Success(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateProfileTestDeps(t)

	userID := uuid.New()
	existingUser := &entity.User{ID: userID}
	existingProfile := entity.NewUserProfile(userID)

	avatarURL := "https://example.com/avatar.png"
	bio := "This is my bio"
	locale := "en"
	timezone := "UTC"
	theme := "dark"
	notifPrefs := entity.NotificationPreferences{EmailEnabled: true, PushEnabled: false}

	deps.userRepo.On("FindByID", ctx, userID).Return(existingUser, nil)
	deps.profileRepo.On("FindByUserID", ctx, userID).Return(existingProfile, nil)
	deps.profileRepo.On("Upsert", ctx, mock.AnythingOfType("*entity.UserProfile")).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.UpdateProfileInput{
		UserID:                  userID,
		AvatarURL:               &avatarURL,
		Bio:                     &bio,
		Locale:                  &locale,
		Timezone:                &timezone,
		Theme:                   &theme,
		NotificationPreferences: &notifPrefs,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, avatarURL, output.Profile.AvatarURL)
	assert.Equal(t, bio, output.Profile.Bio)
	assert.Equal(t, locale, output.Profile.Locale)
	assert.Equal(t, timezone, output.Profile.Timezone)
	assert.Equal(t, theme, output.Profile.Theme)
	assert.Equal(t, notifPrefs, output.Profile.NotificationPreferences)
}

func TestUpdateProfileCommand_Execute_PartialUpdate_OnlyBio(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateProfileTestDeps(t)

	userID := uuid.New()
	existingUser := &entity.User{ID: userID}
	existingProfile := entity.NewUserProfile(userID)
	existingProfile.AvatarURL = "https://example.com/old-avatar.png"

	bio := "Updated bio only"

	deps.userRepo.On("FindByID", ctx, userID).Return(existingUser, nil)
	deps.profileRepo.On("FindByUserID", ctx, userID).Return(existingProfile, nil)
	deps.profileRepo.On("Upsert", ctx, mock.AnythingOfType("*entity.UserProfile")).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.UpdateProfileInput{
		UserID: userID,
		Bio:    &bio,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, bio, output.Profile.Bio)
	assert.Equal(t, "https://example.com/old-avatar.png", output.Profile.AvatarURL)
}

func TestUpdateProfileCommand_Execute_BioTooLong_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateProfileTestDeps(t)

	userID := uuid.New()
	existingUser := &entity.User{ID: userID}
	existingProfile := entity.NewUserProfile(userID)

	longBio := strings.Repeat("a", 501)

	deps.userRepo.On("FindByID", ctx, userID).Return(existingUser, nil)
	deps.profileRepo.On("FindByUserID", ctx, userID).Return(existingProfile, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.UpdateProfileInput{
		UserID: userID,
		Bio:    &longBio,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestUpdateProfileCommand_Execute_UserNotFound_ReturnsNotFoundError(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateProfileTestDeps(t)

	userID := uuid.New()
	bio := "Some bio"

	deps.userRepo.On("FindByID", ctx, userID).Return(nil, apperror.NewNotFoundError("user"))

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.UpdateProfileInput{
		UserID: userID,
		Bio:    &bio,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestUpdateProfileCommand_Execute_ProfileNotFoundCreatesDefault(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateProfileTestDeps(t)

	userID := uuid.New()
	existingUser := &entity.User{ID: userID}
	bio := "Bio for new profile"

	deps.userRepo.On("FindByID", ctx, userID).Return(existingUser, nil)
	deps.profileRepo.On("FindByUserID", ctx, userID).Return(nil, apperror.NewNotFoundError("profile"))
	deps.profileRepo.On("Upsert", ctx, mock.AnythingOfType("*entity.UserProfile")).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.UpdateProfileInput{
		UserID: userID,
		Bio:    &bio,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, bio, output.Profile.Bio)
	assert.Equal(t, userID, output.Profile.UserID)
}

func TestUpdateProfileCommand_Execute_InvalidLocale_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateProfileTestDeps(t)

	userID := uuid.New()
	existingUser := &entity.User{ID: userID}
	existingProfile := entity.NewUserProfile(userID)
	invalidLocale := "xx"

	deps.userRepo.On("FindByID", ctx, userID).Return(existingUser, nil)
	deps.profileRepo.On("FindByUserID", ctx, userID).Return(existingProfile, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.UpdateProfileInput{
		UserID: userID,
		Locale: &invalidLocale,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestUpdateProfileCommand_Execute_InvalidTimezone_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateProfileTestDeps(t)

	userID := uuid.New()
	existingUser := &entity.User{ID: userID}
	existingProfile := entity.NewUserProfile(userID)
	invalidTimezone := "Not/ATimezone"

	deps.userRepo.On("FindByID", ctx, userID).Return(existingUser, nil)
	deps.profileRepo.On("FindByUserID", ctx, userID).Return(existingProfile, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.UpdateProfileInput{
		UserID:   userID,
		Timezone: &invalidTimezone,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}
