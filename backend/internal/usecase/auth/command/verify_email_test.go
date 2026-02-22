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

func newValidEmailVerificationToken(userID uuid.UUID) *entity.EmailVerificationToken {
	return &entity.EmailVerificationToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     "valid-token-string",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
}

func newExpiredEmailVerificationToken(userID uuid.UUID) *entity.EmailVerificationToken {
	return &entity.EmailVerificationToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     "expired-token-string",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now().Add(-25 * time.Hour),
	}
}

func TestVerifyEmailCommand_Execute_ValidToken_ActivatesUser(t *testing.T) {
	ctx := context.Background()
	user := newPendingUser(t)
	token := newValidEmailVerificationToken(user.ID)

	userRepo := mocks.NewMockUserRepository(t)
	emailVerificationTokenRepo := mocks.NewMockEmailVerificationTokenRepository(t)
	txManager := mocks.NewMockTransactionManager(t)

	emailVerificationTokenRepo.On("FindByToken", ctx, "valid-token-string").Return(token, nil)
	userRepo.On("FindByID", ctx, user.ID).Return(user, nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)
	emailVerificationTokenRepo.On("Delete", ctx, token.ID).Return(nil)

	cmd := command.NewVerifyEmailCommand(userRepo, emailVerificationTokenRepo, txManager)
	output, err := cmd.Execute(ctx, command.VerifyEmailInput{Token: "valid-token-string"})

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "email verified successfully", output.Message)
}

func TestVerifyEmailCommand_Execute_ValidToken_SetsUserStatusActive(t *testing.T) {
	ctx := context.Background()
	user := newPendingUser(t)
	token := newValidEmailVerificationToken(user.ID)

	userRepo := mocks.NewMockUserRepository(t)
	emailVerificationTokenRepo := mocks.NewMockEmailVerificationTokenRepository(t)
	txManager := mocks.NewMockTransactionManager(t)

	emailVerificationTokenRepo.On("FindByToken", ctx, "valid-token-string").Return(token, nil)
	userRepo.On("FindByID", ctx, user.ID).Return(user, nil)
	userRepo.On("Update", ctx, mock.MatchedBy(func(u *entity.User) bool {
		return u.Status == entity.UserStatusActive && u.EmailVerified
	})).Return(nil)
	emailVerificationTokenRepo.On("Delete", ctx, token.ID).Return(nil)

	cmd := command.NewVerifyEmailCommand(userRepo, emailVerificationTokenRepo, txManager)
	_, err := cmd.Execute(ctx, command.VerifyEmailInput{Token: "valid-token-string"})

	require.NoError(t, err)
	userRepo.AssertCalled(t, "Update", ctx, mock.AnythingOfType("*entity.User"))
}

func TestVerifyEmailCommand_Execute_EmptyToken_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()

	userRepo := mocks.NewMockUserRepository(t)
	emailVerificationTokenRepo := mocks.NewMockEmailVerificationTokenRepository(t)
	txManager := mocks.NewMockTransactionManager(t)

	cmd := command.NewVerifyEmailCommand(userRepo, emailVerificationTokenRepo, txManager)
	output, err := cmd.Execute(ctx, command.VerifyEmailInput{Token: ""})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
	assert.Contains(t, appErr.Message, "token is required")
}

func TestVerifyEmailCommand_Execute_TokenNotFound_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()

	userRepo := mocks.NewMockUserRepository(t)
	emailVerificationTokenRepo := mocks.NewMockEmailVerificationTokenRepository(t)
	txManager := mocks.NewMockTransactionManager(t)

	emailVerificationTokenRepo.On("FindByToken", ctx, "nonexistent-token").
		Return(nil, errors.New("not found"))

	cmd := command.NewVerifyEmailCommand(userRepo, emailVerificationTokenRepo, txManager)
	output, err := cmd.Execute(ctx, command.VerifyEmailInput{Token: "nonexistent-token"})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
	assert.Contains(t, appErr.Message, "invalid or expired token")
}

func TestVerifyEmailCommand_Execute_ExpiredToken_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	expiredToken := newExpiredEmailVerificationToken(userID)

	userRepo := mocks.NewMockUserRepository(t)
	emailVerificationTokenRepo := mocks.NewMockEmailVerificationTokenRepository(t)
	txManager := mocks.NewMockTransactionManager(t)

	emailVerificationTokenRepo.On("FindByToken", ctx, "expired-token-string").Return(expiredToken, nil)
	emailVerificationTokenRepo.On("Delete", ctx, expiredToken.ID).Return(nil)

	cmd := command.NewVerifyEmailCommand(userRepo, emailVerificationTokenRepo, txManager)
	output, err := cmd.Execute(ctx, command.VerifyEmailInput{Token: "expired-token-string"})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
	assert.Contains(t, appErr.Message, "token has expired")
}

func TestVerifyEmailCommand_Execute_AlreadyVerified_ReturnsSuccessMessage(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)
	token := newValidEmailVerificationToken(user.ID)

	userRepo := mocks.NewMockUserRepository(t)
	emailVerificationTokenRepo := mocks.NewMockEmailVerificationTokenRepository(t)
	txManager := mocks.NewMockTransactionManager(t)

	emailVerificationTokenRepo.On("FindByToken", ctx, "valid-token-string").Return(token, nil)
	userRepo.On("FindByID", ctx, user.ID).Return(user, nil)
	emailVerificationTokenRepo.On("Delete", ctx, token.ID).Return(nil)

	cmd := command.NewVerifyEmailCommand(userRepo, emailVerificationTokenRepo, txManager)
	output, err := cmd.Execute(ctx, command.VerifyEmailInput{Token: "valid-token-string"})

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "email already verified", output.Message)
}

func TestVerifyEmailCommand_Execute_UserNotFound_ReturnsNotFoundError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	token := newValidEmailVerificationToken(userID)

	userRepo := mocks.NewMockUserRepository(t)
	emailVerificationTokenRepo := mocks.NewMockEmailVerificationTokenRepository(t)
	txManager := mocks.NewMockTransactionManager(t)

	emailVerificationTokenRepo.On("FindByToken", ctx, "valid-token-string").Return(token, nil)
	userRepo.On("FindByID", ctx, userID).Return(nil, errors.New("not found"))

	cmd := command.NewVerifyEmailCommand(userRepo, emailVerificationTokenRepo, txManager)
	output, err := cmd.Execute(ctx, command.VerifyEmailInput{Token: "valid-token-string"})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
	assert.Contains(t, appErr.Message, "user not found")
}
