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

const testPassword = "Password123"

func newActiveUser(t *testing.T) *entity.User {
	t.Helper()
	email, err := valueobject.NewEmail("test@example.com")
	require.NoError(t, err)

	pw, err := valueobject.NewPassword(testPassword, "test@example.com")
	require.NoError(t, err)

	return &entity.User{
		ID:            uuid.New(),
		Email:         email,
		Name:          "Test User",
		PasswordHash:  pw.Hash(),
		Status:        entity.UserStatusActive,
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func newLoginInput() command.LoginInput {
	return command.LoginInput{
		Email:     "test@example.com",
		Password:  testPassword,
		UserAgent: "test-agent",
		IPAddress: "127.0.0.1",
	}
}

func TestLoginCommand_Execute_ValidCredentials_ReturnsSession(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)
	input := newLoginInput()

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).Return(user, nil)
	sessionRepo.On("CountByUserID", ctx, user.ID).Return(int64(0), nil)
	sessionRepo.On("Save", ctx, mock.AnythingOfType("*entity.Session")).Return(nil)

	cmd := command.NewLoginCommand(userRepo, sessionRepo)
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.NotEmpty(t, output.SessionID)
	assert.Equal(t, user.ID, output.User.ID)
}

func TestLoginCommand_Execute_InvalidEmail_ReturnsUnauthorized(t *testing.T) {
	ctx := context.Background()
	input := command.LoginInput{
		Email:    "invalid-email",
		Password: testPassword,
	}

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	cmd := command.NewLoginCommand(userRepo, sessionRepo)
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeUnauthorized, appErr.Code)
}

func TestLoginCommand_Execute_UserNotFound_ReturnsUnauthorized(t *testing.T) {
	ctx := context.Background()
	input := newLoginInput()

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).
		Return(nil, errors.New("not found"))

	cmd := command.NewLoginCommand(userRepo, sessionRepo)
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeUnauthorized, appErr.Code)
	assert.Contains(t, appErr.Message, "invalid credentials")
}

func TestLoginCommand_Execute_WrongPassword_ReturnsUnauthorized(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)
	input := command.LoginInput{
		Email:    "test@example.com",
		Password: "WrongPassword123",
	}

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).Return(user, nil)

	cmd := command.NewLoginCommand(userRepo, sessionRepo)
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeUnauthorized, appErr.Code)
	assert.Contains(t, appErr.Message, "invalid credentials")
}

func TestLoginCommand_Execute_PendingUser_ReturnsSession(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)
	user.Status = entity.UserStatusPending
	user.EmailVerified = false
	input := newLoginInput()

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).Return(user, nil)
	sessionRepo.On("CountByUserID", ctx, user.ID).Return(int64(0), nil)
	sessionRepo.On("Save", ctx, mock.AnythingOfType("*entity.Session")).Return(nil)

	cmd := command.NewLoginCommand(userRepo, sessionRepo)
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.NotEmpty(t, output.SessionID)
	assert.Equal(t, user.ID, output.User.ID)
}

func TestLoginCommand_Execute_SuspendedUser_ReturnsUnauthorized(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)
	user.Status = entity.UserStatusSuspended
	input := newLoginInput()

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).Return(user, nil)

	cmd := command.NewLoginCommand(userRepo, sessionRepo)
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeUnauthorized, appErr.Code)
	assert.Contains(t, appErr.Message, "account suspended")
}

func TestLoginCommand_Execute_DeactivatedUser_ReturnsUnauthorized(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)
	user.Status = entity.UserStatusDeactivated
	input := newLoginInput()

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).Return(user, nil)

	cmd := command.NewLoginCommand(userRepo, sessionRepo)
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeUnauthorized, appErr.Code)
	assert.Contains(t, appErr.Message, "account deactivated")
}

func TestLoginCommand_Execute_OAuthOnlyUser_ReturnsUnauthorized(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)
	user.PasswordHash = "" // OAuth-only user has no password
	input := newLoginInput()

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).Return(user, nil)

	cmd := command.NewLoginCommand(userRepo, sessionRepo)
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeUnauthorized, appErr.Code)
	assert.Contains(t, appErr.Message, "invalid credentials")
}

func TestLoginCommand_Execute_SessionLimitReached_DeletesOldestAndCreatesNew(t *testing.T) {
	ctx := context.Background()
	user := newActiveUser(t)
	input := newLoginInput()

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).Return(user, nil)
	sessionRepo.On("CountByUserID", ctx, user.ID).Return(int64(entity.MaxActiveSessionsPerUser), nil)
	sessionRepo.On("DeleteOldestByUserID", ctx, user.ID).Return(nil)
	sessionRepo.On("Save", ctx, mock.AnythingOfType("*entity.Session")).Return(nil)

	cmd := command.NewLoginCommand(userRepo, sessionRepo)
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.NotEmpty(t, output.SessionID)
	sessionRepo.AssertCalled(t, "DeleteOldestByUserID", ctx, user.ID)
}
