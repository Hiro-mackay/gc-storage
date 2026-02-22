package command_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

func newOAuthUser(t *testing.T) *entity.User {
	t.Helper()
	email, err := valueobject.NewEmail("oauth@example.com")
	require.NoError(t, err)

	return &entity.User{
		ID:            uuid.New(),
		Email:         email,
		Name:          "OAuth User",
		PasswordHash:  "",
		Status:        entity.UserStatusActive,
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func TestSetPasswordCommand_Execute_OAuthUser_SetsPassword(t *testing.T) {
	ctx := context.Background()
	user := newOAuthUser(t)
	sessionID := "current-session-id"
	currentSession := &entity.Session{ID: sessionID, UserID: user.ID}

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	userRepo.On("FindByID", ctx, user.ID).Return(user, nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)
	sessionRepo.On("FindByID", ctx, sessionID).Return(currentSession, nil)
	sessionRepo.On("DeleteByUserID", ctx, user.ID).Return(nil)
	sessionRepo.On("Save", ctx, currentSession).Return(nil)

	cmd := command.NewSetPasswordCommand(userRepo, sessionRepo)
	output, err := cmd.Execute(ctx, command.SetPasswordInput{
		UserID:           user.ID,
		Password:         "NewPassword123",
		CurrentSessionID: sessionID,
	})

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "password set successfully", output.Message)
}

func TestSetPasswordCommand_Execute_UserAlreadyHasPassword_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	userRepo.On("FindByID", ctx, user.ID).Return(user, nil)

	cmd := command.NewSetPasswordCommand(userRepo, sessionRepo)
	output, err := cmd.Execute(ctx, command.SetPasswordInput{
		UserID:   user.ID,
		Password: "NewPassword123",
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
	assert.Contains(t, appErr.Message, "password already set")
}

func TestSetPasswordCommand_Execute_UserNotFound_ReturnsNotFoundError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	userRepo.On("FindByID", ctx, userID).Return(nil, errors.New("not found"))

	cmd := command.NewSetPasswordCommand(userRepo, sessionRepo)
	output, err := cmd.Execute(ctx, command.SetPasswordInput{
		UserID:   userID,
		Password: "NewPassword123",
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestSetPasswordCommand_Execute_WeakPassword_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	user := newOAuthUser(t)

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	userRepo.On("FindByID", ctx, user.ID).Return(user, nil)

	cmd := command.NewSetPasswordCommand(userRepo, sessionRepo)
	output, err := cmd.Execute(ctx, command.SetPasswordInput{
		UserID:   user.ID,
		Password: "weak",
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}
