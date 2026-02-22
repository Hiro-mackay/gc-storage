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

type updateUserTestDeps struct {
	userRepo *mocks.MockUserRepository
}

func newUpdateUserTestDeps(t *testing.T) *updateUserTestDeps {
	t.Helper()
	return &updateUserTestDeps{
		userRepo: mocks.NewMockUserRepository(t),
	}
}

func (d *updateUserTestDeps) newCommand() *command.UpdateUserCommand {
	return command.NewUpdateUserCommand(d.userRepo)
}

func TestUpdateUserCommand_Execute_Success(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateUserTestDeps(t)

	userID := uuid.New()
	name := "New Name"
	existingUser := &entity.User{
		ID:   userID,
		Name: "Old Name",
	}

	deps.userRepo.On("FindByID", ctx, userID).Return(existingUser, nil)
	deps.userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.UpdateUserInput{
		UserID: userID,
		Name:   &name,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, name, output.User.Name)
}

func TestUpdateUserCommand_Execute_EmptyName_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateUserTestDeps(t)

	userID := uuid.New()
	emptyName := ""

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.UpdateUserInput{
		UserID: userID,
		Name:   &emptyName,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestUpdateUserCommand_Execute_NameTooLong_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateUserTestDeps(t)

	userID := uuid.New()
	longName := strings.Repeat("a", 101)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.UpdateUserInput{
		UserID: userID,
		Name:   &longName,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestUpdateUserCommand_Execute_UserNotFound_ReturnsNotFoundError(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateUserTestDeps(t)

	userID := uuid.New()
	name := "Some Name"

	deps.userRepo.On("FindByID", ctx, userID).Return(nil, apperror.NewNotFoundError("user"))

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.UpdateUserInput{
		UserID: userID,
		Name:   &name,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestUpdateUserCommand_Execute_NilName_NoChange_ReturnsUser(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateUserTestDeps(t)

	userID := uuid.New()
	existingUser := &entity.User{
		ID:   userID,
		Name: "Unchanged Name",
	}

	deps.userRepo.On("FindByID", ctx, userID).Return(existingUser, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.UpdateUserInput{
		UserID: userID,
		Name:   nil,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, "Unchanged Name", output.User.Name)
}
