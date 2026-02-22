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

const securityMessage = "If your email address is registered, a verification email has been sent."

func newPendingUser(t *testing.T) *entity.User {
	t.Helper()
	email, err := valueobject.NewEmail("pending@example.com")
	require.NoError(t, err)
	pw, err := valueobject.NewPassword(testPassword, "pending@example.com")
	require.NoError(t, err)
	return &entity.User{
		ID:            uuid.New(),
		Email:         email,
		Name:          "Pending User",
		PasswordHash:  pw.Hash(),
		Status:        entity.UserStatusPending,
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func newResendEmailVerificationInput(email string) command.ResendEmailVerificationInput {
	return command.ResendEmailVerificationInput{
		Email: email,
	}
}

func TestResendEmailVerificationCommand_Execute_ValidEmail_ReturnsSecurityMessage(t *testing.T) {
	ctx := context.Background()
	user := newPendingUser(t)

	userRepo := mocks.NewMockUserRepository(t)
	emailVerificationTokenRepo := mocks.NewMockEmailVerificationTokenRepository(t)

	userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).Return(user, nil)
	emailVerificationTokenRepo.On("DeleteByUserID", ctx, user.ID).Return(nil)
	emailVerificationTokenRepo.On("Create", ctx, mock.AnythingOfType("*entity.EmailVerificationToken")).Return(nil)

	cmd := command.NewResendEmailVerificationCommand(userRepo, emailVerificationTokenRepo, nil, "http://localhost:3000")
	output, err := cmd.Execute(ctx, newResendEmailVerificationInput("pending@example.com"))

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, securityMessage, output.Message)
}

func TestResendEmailVerificationCommand_Execute_InvalidEmail_ReturnsSecurityMessage(t *testing.T) {
	ctx := context.Background()

	userRepo := mocks.NewMockUserRepository(t)
	emailVerificationTokenRepo := mocks.NewMockEmailVerificationTokenRepository(t)

	cmd := command.NewResendEmailVerificationCommand(userRepo, emailVerificationTokenRepo, nil, "http://localhost:3000")
	output, err := cmd.Execute(ctx, newResendEmailVerificationInput("invalid-email"))

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, securityMessage, output.Message)
}

func TestResendEmailVerificationCommand_Execute_NonExistentEmail_ReturnsSecurityMessage(t *testing.T) {
	ctx := context.Background()

	userRepo := mocks.NewMockUserRepository(t)
	emailVerificationTokenRepo := mocks.NewMockEmailVerificationTokenRepository(t)

	userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).
		Return(nil, errors.New("not found"))

	cmd := command.NewResendEmailVerificationCommand(userRepo, emailVerificationTokenRepo, nil, "http://localhost:3000")
	output, err := cmd.Execute(ctx, newResendEmailVerificationInput("notfound@example.com"))

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, securityMessage, output.Message)
}

func TestResendEmailVerificationCommand_Execute_AlreadyVerified_ReturnsSecurityMessage(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)

	userRepo := mocks.NewMockUserRepository(t)
	emailVerificationTokenRepo := mocks.NewMockEmailVerificationTokenRepository(t)

	userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).Return(user, nil)

	cmd := command.NewResendEmailVerificationCommand(userRepo, emailVerificationTokenRepo, nil, "http://localhost:3000")
	output, err := cmd.Execute(ctx, newResendEmailVerificationInput("test@example.com"))

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, securityMessage, output.Message)
}

func TestResendEmailVerificationCommand_Execute_TokenCreateFails_ReturnsInternalError(t *testing.T) {
	ctx := context.Background()
	user := newPendingUser(t)

	userRepo := mocks.NewMockUserRepository(t)
	emailVerificationTokenRepo := mocks.NewMockEmailVerificationTokenRepository(t)

	userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).Return(user, nil)
	emailVerificationTokenRepo.On("DeleteByUserID", ctx, user.ID).Return(nil)
	emailVerificationTokenRepo.On("Create", ctx, mock.AnythingOfType("*entity.EmailVerificationToken")).
		Return(errors.New("db error"))

	cmd := command.NewResendEmailVerificationCommand(userRepo, emailVerificationTokenRepo, nil, "http://localhost:3000")
	output, err := cmd.Execute(ctx, newResendEmailVerificationInput("pending@example.com"))

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeInternalError, appErr.Code)
}
