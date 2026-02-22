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
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

func newValidResetToken(userID uuid.UUID) *entity.PasswordResetToken {
	return &entity.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     "valid-reset-token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}
}

func TestResetPasswordCommand_Execute_ValidToken_ResetsPassword(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)
	token := newValidResetToken(user.ID)

	userRepo := mocks.NewMockUserRepository(t)
	tokenRepo := mocks.NewMockPasswordResetTokenRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)
	txManager := mocks.NewMockTransactionManager(t)

	tokenRepo.On("FindByToken", ctx, "valid-reset-token").Return(token, nil)
	userRepo.On("FindByID", ctx, user.ID).Return(user, nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)
	tokenRepo.On("MarkAsUsed", ctx, token.ID).Return(nil)
	sessionRepo.On("DeleteByUserID", ctx, user.ID).Return(nil)

	cmd := command.NewResetPasswordCommand(userRepo, tokenRepo, sessionRepo, txManager)
	output, err := cmd.Execute(ctx, command.ResetPasswordInput{
		Token:    "valid-reset-token",
		Password: "NewPassword123",
	})

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "password reset successfully", output.Message)
}

func TestResetPasswordCommand_Execute_EmptyToken_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()

	userRepo := mocks.NewMockUserRepository(t)
	tokenRepo := mocks.NewMockPasswordResetTokenRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)
	txManager := mocks.NewMockTransactionManager(t)

	cmd := command.NewResetPasswordCommand(userRepo, tokenRepo, sessionRepo, txManager)
	output, err := cmd.Execute(ctx, command.ResetPasswordInput{
		Token:    "",
		Password: "NewPassword123",
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
	assert.Contains(t, appErr.Message, "token is required")
}

func TestResetPasswordCommand_Execute_TokenNotFound_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()

	userRepo := mocks.NewMockUserRepository(t)
	tokenRepo := mocks.NewMockPasswordResetTokenRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)
	txManager := mocks.NewMockTransactionManager(t)

	tokenRepo.On("FindByToken", ctx, "nonexistent-token").
		Return(nil, errors.New("not found"))

	cmd := command.NewResetPasswordCommand(userRepo, tokenRepo, sessionRepo, txManager)
	output, err := cmd.Execute(ctx, command.ResetPasswordInput{
		Token:    "nonexistent-token",
		Password: "NewPassword123",
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
	assert.Contains(t, appErr.Message, "invalid or expired token")
}

func TestResetPasswordCommand_Execute_UsedToken_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	usedAt := time.Now().Add(-30 * time.Minute)
	token := &entity.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     "used-token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now().Add(-1 * time.Hour),
		UsedAt:    &usedAt,
	}

	userRepo := mocks.NewMockUserRepository(t)
	tokenRepo := mocks.NewMockPasswordResetTokenRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)
	txManager := mocks.NewMockTransactionManager(t)

	tokenRepo.On("FindByToken", ctx, "used-token").Return(token, nil)

	cmd := command.NewResetPasswordCommand(userRepo, tokenRepo, sessionRepo, txManager)
	output, err := cmd.Execute(ctx, command.ResetPasswordInput{
		Token:    "used-token",
		Password: "NewPassword123",
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
	assert.Contains(t, appErr.Message, "invalid or expired token")
}

func TestResetPasswordCommand_Execute_ExpiredToken_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	token := &entity.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}

	userRepo := mocks.NewMockUserRepository(t)
	tokenRepo := mocks.NewMockPasswordResetTokenRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)
	txManager := mocks.NewMockTransactionManager(t)

	tokenRepo.On("FindByToken", ctx, "expired-token").Return(token, nil)

	cmd := command.NewResetPasswordCommand(userRepo, tokenRepo, sessionRepo, txManager)
	output, err := cmd.Execute(ctx, command.ResetPasswordInput{
		Token:    "expired-token",
		Password: "NewPassword123",
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
	assert.Contains(t, appErr.Message, "invalid or expired token")
}

func TestResetPasswordCommand_Execute_UserNotFound_ReturnsNotFoundError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	token := newValidResetToken(userID)

	userRepo := mocks.NewMockUserRepository(t)
	tokenRepo := mocks.NewMockPasswordResetTokenRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)
	txManager := mocks.NewMockTransactionManager(t)

	tokenRepo.On("FindByToken", ctx, "valid-reset-token").Return(token, nil)
	userRepo.On("FindByID", ctx, userID).Return(nil, errors.New("not found"))

	cmd := command.NewResetPasswordCommand(userRepo, tokenRepo, sessionRepo, txManager)
	output, err := cmd.Execute(ctx, command.ResetPasswordInput{
		Token:    "valid-reset-token",
		Password: "NewPassword123",
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestResetPasswordCommand_Execute_WeakPassword_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)
	token := newValidResetToken(user.ID)

	userRepo := mocks.NewMockUserRepository(t)
	tokenRepo := mocks.NewMockPasswordResetTokenRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)
	txManager := mocks.NewMockTransactionManager(t)

	tokenRepo.On("FindByToken", ctx, "valid-reset-token").Return(token, nil)
	userRepo.On("FindByID", ctx, user.ID).Return(user, nil)

	cmd := command.NewResetPasswordCommand(userRepo, tokenRepo, sessionRepo, txManager)
	output, err := cmd.Execute(ctx, command.ResetPasswordInput{
		Token:    "valid-reset-token",
		Password: "weak",
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}
