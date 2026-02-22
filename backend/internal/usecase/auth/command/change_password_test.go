package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

func TestChangePasswordCommand_Execute_ValidInput_ChangesPassword(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)
	sessionID := "current-session-id"
	currentSession := &entity.Session{ID: sessionID, UserID: user.ID}

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	userRepo.On("FindByID", ctx, user.ID).Return(user, nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)
	sessionRepo.On("FindByID", ctx, sessionID).Return(currentSession, nil)
	sessionRepo.On("DeleteByUserID", ctx, user.ID).Return(nil)
	sessionRepo.On("Save", ctx, currentSession).Return(nil)

	cmd := command.NewChangePasswordCommand(userRepo, sessionRepo)
	output, err := cmd.Execute(ctx, command.ChangePasswordInput{
		UserID:           user.ID,
		CurrentPassword:  testPassword,
		NewPassword:      "NewPassword456",
		CurrentSessionID: sessionID,
	})

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "password changed successfully", output.Message)
}

func TestChangePasswordCommand_Execute_UserNotFound_ReturnsNotFoundError(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	userRepo.On("FindByID", ctx, user.ID).Return(nil, errors.New("not found"))

	cmd := command.NewChangePasswordCommand(userRepo, sessionRepo)
	output, err := cmd.Execute(ctx, command.ChangePasswordInput{
		UserID:          user.ID,
		CurrentPassword: testPassword,
		NewPassword:     "NewPassword456",
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestChangePasswordCommand_Execute_WrongCurrentPassword_ReturnsUnauthorized(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	userRepo.On("FindByID", ctx, user.ID).Return(user, nil)

	cmd := command.NewChangePasswordCommand(userRepo, sessionRepo)
	output, err := cmd.Execute(ctx, command.ChangePasswordInput{
		UserID:          user.ID,
		CurrentPassword: "WrongPassword123",
		NewPassword:     "NewPassword456",
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeUnauthorized, appErr.Code)
	assert.Contains(t, appErr.Message, "current password is incorrect")
}

func TestChangePasswordCommand_Execute_WeakNewPassword_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	userRepo.On("FindByID", ctx, user.ID).Return(user, nil)

	cmd := command.NewChangePasswordCommand(userRepo, sessionRepo)
	output, err := cmd.Execute(ctx, command.ChangePasswordInput{
		UserID:          user.ID,
		CurrentPassword: testPassword,
		NewPassword:     "weak",
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}
