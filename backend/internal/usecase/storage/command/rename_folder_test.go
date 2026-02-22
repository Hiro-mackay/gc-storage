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
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type renameFolderTestDeps struct {
	folderRepo *mocks.MockFolderRepository
	userRepo   *mocks.MockUserRepository
}

func newRenameFolderTestDeps(t *testing.T) *renameFolderTestDeps {
	t.Helper()
	return &renameFolderTestDeps{
		folderRepo: mocks.NewMockFolderRepository(t),
		userRepo:   mocks.NewMockUserRepository(t),
	}
}

func (d *renameFolderTestDeps) newCommand() *command.RenameFolderCommand {
	return command.NewRenameFolderCommand(d.folderRepo, d.userRepo)
}

func newRootFolderEntity(ownerID uuid.UUID) *entity.Folder {
	folderName, _ := valueobject.NewFolderName("test-folder")
	return entity.ReconstructFolder(
		uuid.New(), folderName, nil, ownerID, ownerID, 0,
		entity.FolderStatusActive, time.Now(), time.Now(),
	)
}

func newChildFolderEntity(ownerID, parentID uuid.UUID) *entity.Folder {
	folderName, _ := valueobject.NewFolderName("child-folder")
	return entity.ReconstructFolder(
		uuid.New(), folderName, &parentID, ownerID, ownerID, 1,
		entity.FolderStatusActive, time.Now(), time.Now(),
	)
}

func newUserWithPersonalFolder(folderID uuid.UUID) *entity.User {
	email, _ := valueobject.NewEmail("test@example.com")
	user := entity.NewUser(email, "Test User", "hashedpw")
	user.PersonalFolderID = &folderID
	return user
}

func newUserWithoutPersonalFolder() *entity.User {
	email, _ := valueobject.NewEmail("test@example.com")
	return entity.NewUser(email, "Test User", "hashedpw")
}

func TestRenameFolderCommand_Execute_ValidRename_ReturnsFolder(t *testing.T) {
	ctx := context.Background()
	deps := newRenameFolderTestDeps(t)

	ownerID := uuid.New()
	folder := newRootFolderEntity(ownerID)
	user := newUserWithoutPersonalFolder()
	user.ID = ownerID

	input := command.RenameFolderInput{
		FolderID: folder.ID,
		NewName:  "renamed-folder",
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(user, nil)
	deps.folderRepo.On("ExistsByNameAndOwnerRoot", ctx, mock.AnythingOfType("valueobject.FolderName"), ownerID).Return(false, nil)
	deps.folderRepo.On("Update", ctx, folder).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, folder.ID, output.Folder.ID)
	assert.Equal(t, "renamed-folder", output.Folder.Name.String())
}

func TestRenameFolderCommand_Execute_SameName_SkipsDuplicateCheck(t *testing.T) {
	ctx := context.Background()
	deps := newRenameFolderTestDeps(t)

	ownerID := uuid.New()
	folder := newRootFolderEntity(ownerID)
	user := newUserWithoutPersonalFolder()
	user.ID = ownerID

	input := command.RenameFolderInput{
		FolderID: folder.ID,
		NewName:  "test-folder",
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(user, nil)
	deps.folderRepo.On("Update", ctx, folder).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
}

func TestRenameFolderCommand_Execute_WithParent_ChecksNameInParent(t *testing.T) {
	ctx := context.Background()
	deps := newRenameFolderTestDeps(t)

	ownerID := uuid.New()
	parentID := uuid.New()
	folder := newChildFolderEntity(ownerID, parentID)
	user := newUserWithoutPersonalFolder()
	user.ID = ownerID

	input := command.RenameFolderInput{
		FolderID: folder.ID,
		NewName:  "renamed-child",
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(user, nil)
	deps.folderRepo.On("ExistsByNameAndParent", ctx, mock.AnythingOfType("valueobject.FolderName"), &parentID, ownerID).Return(false, nil)
	deps.folderRepo.On("Update", ctx, folder).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
}

func TestRenameFolderCommand_Execute_InvalidName_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newRenameFolderTestDeps(t)

	input := command.RenameFolderInput{
		FolderID: uuid.New(),
		NewName:  "",
		UserID:   uuid.New(),
	}

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestRenameFolderCommand_Execute_FolderNotFound_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newRenameFolderTestDeps(t)

	folderID := uuid.New()
	notFoundErr := apperror.NewNotFoundError("folder")

	input := command.RenameFolderInput{
		FolderID: folderID,
		NewName:  "new-name",
		UserID:   uuid.New(),
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

func TestRenameFolderCommand_Execute_NotOwner_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newRenameFolderTestDeps(t)

	ownerID := uuid.New()
	differentUserID := uuid.New()
	folder := newRootFolderEntity(ownerID)

	input := command.RenameFolderInput{
		FolderID: folder.ID,
		NewName:  "new-name",
		UserID:   differentUserID,
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

func TestRenameFolderCommand_Execute_PersonalFolder_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newRenameFolderTestDeps(t)

	ownerID := uuid.New()
	folder := newRootFolderEntity(ownerID)
	user := newUserWithPersonalFolder(folder.ID)
	user.ID = ownerID

	input := command.RenameFolderInput{
		FolderID: folder.ID,
		NewName:  "new-name",
		UserID:   ownerID,
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

func TestRenameFolderCommand_Execute_DuplicateName_ReturnsConflict(t *testing.T) {
	ctx := context.Background()
	deps := newRenameFolderTestDeps(t)

	ownerID := uuid.New()
	folder := newRootFolderEntity(ownerID)
	user := newUserWithoutPersonalFolder()
	user.ID = ownerID

	input := command.RenameFolderInput{
		FolderID: folder.ID,
		NewName:  "duplicate-name",
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(user, nil)
	deps.folderRepo.On("ExistsByNameAndOwnerRoot", ctx, mock.AnythingOfType("valueobject.FolderName"), ownerID).Return(true, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestRenameFolderCommand_Execute_UserRepoError_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newRenameFolderTestDeps(t)

	ownerID := uuid.New()
	folder := newRootFolderEntity(ownerID)
	repoErr := errors.New("database error")

	input := command.RenameFolderInput{
		FolderID: folder.ID,
		NewName:  "new-name",
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(nil, repoErr)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	assert.Equal(t, repoErr, err)
}
