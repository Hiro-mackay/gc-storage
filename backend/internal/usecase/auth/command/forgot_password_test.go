package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

func TestForgotPasswordCommand_Execute_ValidEmail_ReturnsSecurityMessage(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)

	userRepo := mocks.NewMockUserRepository(t)
	tokenRepo := mocks.NewMockPasswordResetTokenRepository(t)

	userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).Return(user, nil)
	tokenRepo.On("DeleteByUserID", ctx, user.ID).Return(nil)
	tokenRepo.On("Create", ctx, mock.Anything).Return(nil)

	cmd := command.NewForgotPasswordCommand(userRepo, tokenRepo, nil, "http://localhost:3000")
	output, err := cmd.Execute(ctx, command.ForgotPasswordInput{Email: "test@example.com"})

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "If your email address is registered, a password reset email has been sent.", output.Message)
}

func TestForgotPasswordCommand_Execute_NonExistentEmail_ReturnsSecurityMessage(t *testing.T) {
	ctx := context.Background()

	userRepo := mocks.NewMockUserRepository(t)
	tokenRepo := mocks.NewMockPasswordResetTokenRepository(t)

	userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).
		Return(nil, errors.New("not found"))

	cmd := command.NewForgotPasswordCommand(userRepo, tokenRepo, nil, "http://localhost:3000")
	output, err := cmd.Execute(ctx, command.ForgotPasswordInput{Email: "nonexistent@example.com"})

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "If your email address is registered, a password reset email has been sent.", output.Message)
}

func TestForgotPasswordCommand_Execute_InvalidEmail_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()

	userRepo := mocks.NewMockUserRepository(t)
	tokenRepo := mocks.NewMockPasswordResetTokenRepository(t)

	cmd := command.NewForgotPasswordCommand(userRepo, tokenRepo, nil, "http://localhost:3000")
	output, err := cmd.Execute(ctx, command.ForgotPasswordInput{Email: "invalid"})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestForgotPasswordCommand_Execute_TokenCreateFails_ReturnsInternalError(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)

	userRepo := mocks.NewMockUserRepository(t)
	tokenRepo := mocks.NewMockPasswordResetTokenRepository(t)

	userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).Return(user, nil)
	tokenRepo.On("DeleteByUserID", ctx, user.ID).Return(nil)
	tokenRepo.On("Create", ctx, mock.Anything).Return(errors.New("db error"))

	cmd := command.NewForgotPasswordCommand(userRepo, tokenRepo, nil, "http://localhost:3000")
	output, err := cmd.Execute(ctx, command.ForgotPasswordInput{Email: "test@example.com"})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeInternalError, appErr.Code)
}
