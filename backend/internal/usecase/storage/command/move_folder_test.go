package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type moveFolderTestDeps struct {
	folderRepo        *mocks.MockFolderRepository
	folderClosureRepo *mocks.MockFolderClosureRepository
	txManager         *mocks.MockTransactionManager
	userRepo          *mocks.MockUserRepository
}

func newMoveFolderTestDeps(t *testing.T) *moveFolderTestDeps {
	t.Helper()
	return &moveFolderTestDeps{
		folderRepo:        mocks.NewMockFolderRepository(t),
		folderClosureRepo: mocks.NewMockFolderClosureRepository(t),
		txManager:         mocks.NewMockTransactionManager(t),
		userRepo:          mocks.NewMockUserRepository(t),
	}
}

func (d *moveFolderTestDeps) newCommand() *command.MoveFolderCommand {
	return command.NewMoveFolderCommand(d.folderRepo, d.folderClosureRepo, d.txManager, d.userRepo)
}

func newMoveFolderUserWithoutPersonalFolder(ownerID uuid.UUID) *entity.User {
	user := newUserWithoutPersonalFolder()
	user.ID = ownerID
	return user
}

func TestMoveFolderCommand_Execute_MoveToNewParent_ReturnsFolder(t *testing.T) {
	ctx := context.Background()
	deps := newMoveFolderTestDeps(t)

	ownerID := uuid.New()
	newParentID := uuid.New()
	folder := newRootFolderEntity(ownerID)
	newParent := newRootFolderEntity(ownerID)
	newParent.ID = newParentID
	user := newUserWithoutPersonalFolder()
	user.ID = ownerID

	input := command.MoveFolderInput{
		FolderID:    folder.ID,
		NewParentID: &newParentID,
		UserID:      ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(user, nil)
	deps.folderRepo.On("FindByID", ctx, newParentID).Return(newParent, nil)
	deps.folderClosureRepo.On("FindDescendantIDs", ctx, folder.ID).Return([]uuid.UUID{}, nil)
	deps.folderRepo.On("ExistsByNameAndParent", ctx, mock.AnythingOfType("valueobject.FolderName"), &newParentID, ownerID).Return(false, nil)
	deps.folderClosureRepo.On("FindAncestorPaths", ctx, newParentID).Return([]*entity.FolderPath{}, nil)
	deps.folderClosureRepo.On("MoveSubtree", ctx, folder.ID, mock.Anything).Return(nil)
	deps.folderRepo.On("Update", ctx, folder).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, folder.ID, output.Folder.ID)
	assert.Equal(t, &newParentID, output.Folder.ParentID)
}
func TestMoveFolderCommand_Execute_MoveToRoot_ReturnsFolder(t *testing.T) {
	ctx := context.Background()
	deps := newMoveFolderTestDeps(t)

	ownerID := uuid.New()
	parentID := uuid.New()
	folder := newChildFolderEntity(ownerID, parentID)
	user := newMoveFolderUserWithoutPersonalFolder(ownerID)

	input := command.MoveFolderInput{
		FolderID:    folder.ID,
		NewParentID: nil,
		UserID:      ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(user, nil)
	deps.folderClosureRepo.On("FindDescendantIDs", ctx, folder.ID).Return([]uuid.UUID{}, nil)
	deps.folderRepo.On("ExistsByNameAndOwnerRoot", ctx, mock.AnythingOfType("valueobject.FolderName"), ownerID).Return(false, nil)
	deps.folderClosureRepo.On("FindAncestorPaths", ctx, mock.Anything).Return([]*entity.FolderPath{}, nil).Maybe()
	deps.folderClosureRepo.On("MoveSubtree", ctx, folder.ID, mock.Anything).Return(nil)
	deps.folderRepo.On("Update", ctx, folder).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Nil(t, output.Folder.ParentID)
}
func TestMoveFolderCommand_Execute_SameLocation_ReturnsNoOp(t *testing.T) {
	ctx := context.Background()
	deps := newMoveFolderTestDeps(t)

	ownerID := uuid.New()
	folder := newRootFolderEntity(ownerID)
	user := newMoveFolderUserWithoutPersonalFolder(ownerID)

	input := command.MoveFolderInput{
		FolderID:    folder.ID,
		NewParentID: nil,
		UserID:      ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(user, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, folder.ID, output.Folder.ID)
}
func TestMoveFolderCommand_Execute_FolderNotFound_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newMoveFolderTestDeps(t)

	folderID := uuid.New()
	notFoundErr := apperror.NewNotFoundError("folder")

	input := command.MoveFolderInput{
		FolderID:    folderID,
		NewParentID: nil,
		UserID:      uuid.New(),
	}

	deps.folderRepo.On("FindByID", ctx, folderID).Return(nil, notFoundErr)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}
func TestMoveFolderCommand_Execute_NotOwner_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newMoveFolderTestDeps(t)

	ownerID := uuid.New()
	differentUserID := uuid.New()
	folder := newRootFolderEntity(ownerID)

	input := command.MoveFolderInput{
		FolderID:    folder.ID,
		NewParentID: nil,
		UserID:      differentUserID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}
func TestMoveFolderCommand_Execute_PersonalFolder_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newMoveFolderTestDeps(t)

	ownerID := uuid.New()
	folder := newRootFolderEntity(ownerID)
	user := newUserWithPersonalFolder(folder.ID)
	user.ID = ownerID

	newParentID := uuid.New()
	input := command.MoveFolderInput{
		FolderID:    folder.ID,
		NewParentID: &newParentID,
		UserID:      ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(user, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}
func TestMoveFolderCommand_Execute_NewParentNotOwned_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newMoveFolderTestDeps(t)

	ownerID := uuid.New()
	anotherOwnerID := uuid.New()
	newParentID := uuid.New()
	folder := newRootFolderEntity(ownerID)
	newParent := newRootFolderEntity(anotherOwnerID)
	newParent.ID = newParentID
	user := newMoveFolderUserWithoutPersonalFolder(ownerID)

	input := command.MoveFolderInput{
		FolderID:    folder.ID,
		NewParentID: &newParentID,
		UserID:      ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(user, nil)
	deps.folderRepo.On("FindByID", ctx, newParentID).Return(newParent, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}
func TestMoveFolderCommand_Execute_CircularMove_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newMoveFolderTestDeps(t)

	ownerID := uuid.New()
	folder := newRootFolderEntity(ownerID)
	user := newMoveFolderUserWithoutPersonalFolder(ownerID)

	selfID := folder.ID
	input := command.MoveFolderInput{
		FolderID:    folder.ID,
		NewParentID: &selfID,
		UserID:      ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(user, nil)
	deps.folderRepo.On("FindByID", ctx, selfID).Return(folder, nil)
	deps.folderClosureRepo.On("FindDescendantIDs", ctx, folder.ID).Return([]uuid.UUID{}, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}
func TestMoveFolderCommand_Execute_DuplicateNameInDestination_ReturnsConflict(t *testing.T) {
	ctx := context.Background()
	deps := newMoveFolderTestDeps(t)

	ownerID := uuid.New()
	newParentID := uuid.New()
	folder := newRootFolderEntity(ownerID)
	newParent := newRootFolderEntity(ownerID)
	newParent.ID = newParentID
	user := newMoveFolderUserWithoutPersonalFolder(ownerID)

	input := command.MoveFolderInput{
		FolderID:    folder.ID,
		NewParentID: &newParentID,
		UserID:      ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(user, nil)
	deps.folderRepo.On("FindByID", ctx, newParentID).Return(newParent, nil)
	deps.folderClosureRepo.On("FindDescendantIDs", ctx, folder.ID).Return([]uuid.UUID{}, nil)
	deps.folderRepo.On("ExistsByNameAndParent", ctx, mock.AnythingOfType("valueobject.FolderName"), &newParentID, ownerID).Return(true, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeConflict, appErr.Code)
}
