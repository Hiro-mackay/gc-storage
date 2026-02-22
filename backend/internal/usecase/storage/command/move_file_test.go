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

type moveFileTestDeps struct {
	fileRepo   *mocks.MockFileRepository
	folderRepo *mocks.MockFolderRepository
}

func newMoveFileTestDeps(t *testing.T) *moveFileTestDeps {
	t.Helper()
	return &moveFileTestDeps{
		fileRepo:   mocks.NewMockFileRepository(t),
		folderRepo: mocks.NewMockFolderRepository(t),
	}
}

func (d *moveFileTestDeps) newCommand() *command.MoveFileCommand {
	return command.NewMoveFileCommand(d.fileRepo, d.folderRepo)
}

func newFolderEntity(ownerID uuid.UUID) *entity.Folder {
	folderName, _ := valueobject.NewFolderName("destination")
	return entity.ReconstructFolder(
		uuid.New(), folderName, nil, ownerID, ownerID, 0,
		entity.FolderStatusActive, time.Now(), time.Now(),
	)
}

func TestMoveFileCommand_Execute_ValidInput_ReturnsOutput(t *testing.T) {
	ctx := context.Background()
	deps := newMoveFileTestDeps(t)

	ownerID := uuid.New()
	sourceFolderID := uuid.New()
	file := newActiveFileEntity(ownerID, sourceFolderID)
	destFolder := newFolderEntity(ownerID)

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)
	deps.folderRepo.On("FindByID", ctx, destFolder.ID).Return(destFolder, nil)
	deps.fileRepo.On("ExistsByNameAndFolder", ctx, file.Name, destFolder.ID).Return(false, nil)
	deps.fileRepo.On("Update", ctx, mock.AnythingOfType("*entity.File")).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.MoveFileInput{
		FileID:      file.ID,
		NewFolderID: destFolder.ID,
		UserID:      ownerID,
	})

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, file.ID, output.FileID)
	assert.Equal(t, destFolder.ID, output.FolderID)
}

func TestMoveFileCommand_Execute_SameFolder_ReturnsCurrentState(t *testing.T) {
	ctx := context.Background()
	deps := newMoveFileTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()
	file := newActiveFileEntity(ownerID, folderID)

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.MoveFileInput{
		FileID:      file.ID,
		NewFolderID: folderID,
		UserID:      ownerID,
	})

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, file.ID, output.FileID)
	assert.Equal(t, folderID, output.FolderID)
}

func TestMoveFileCommand_Execute_FileNotFound_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newMoveFileTestDeps(t)

	fileID := uuid.New()
	repoErr := errors.New("file not found")

	deps.fileRepo.On("FindByID", ctx, fileID).Return(nil, repoErr)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.MoveFileInput{
		FileID:      fileID,
		NewFolderID: uuid.New(),
		UserID:      uuid.New(),
	})

	require.Error(t, err)
	assert.Nil(t, output)
	assert.Equal(t, repoErr, err)
}

func TestMoveFileCommand_Execute_NotOwner_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newMoveFileTestDeps(t)

	ownerID := uuid.New()
	differentUserID := uuid.New()
	folderID := uuid.New()
	file := newActiveFileEntity(ownerID, folderID)

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.MoveFileInput{
		FileID:      file.ID,
		NewFolderID: uuid.New(),
		UserID:      differentUserID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestMoveFileCommand_Execute_DestinationFolderNotFound_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newMoveFileTestDeps(t)

	ownerID := uuid.New()
	sourceFolderID := uuid.New()
	destFolderID := uuid.New()
	file := newActiveFileEntity(ownerID, sourceFolderID)
	repoErr := errors.New("folder not found")

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)
	deps.folderRepo.On("FindByID", ctx, destFolderID).Return(nil, repoErr)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.MoveFileInput{
		FileID:      file.ID,
		NewFolderID: destFolderID,
		UserID:      ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	assert.Equal(t, repoErr, err)
}

func TestMoveFileCommand_Execute_DestinationNotOwned_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newMoveFileTestDeps(t)

	ownerID := uuid.New()
	differentOwnerID := uuid.New()
	sourceFolderID := uuid.New()
	file := newActiveFileEntity(ownerID, sourceFolderID)
	destFolder := newFolderEntity(differentOwnerID)

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)
	deps.folderRepo.On("FindByID", ctx, destFolder.ID).Return(destFolder, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.MoveFileInput{
		FileID:      file.ID,
		NewFolderID: destFolder.ID,
		UserID:      ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestMoveFileCommand_Execute_DuplicateNameInDestination_ReturnsConflict(t *testing.T) {
	ctx := context.Background()
	deps := newMoveFileTestDeps(t)

	ownerID := uuid.New()
	sourceFolderID := uuid.New()
	file := newActiveFileEntity(ownerID, sourceFolderID)
	destFolder := newFolderEntity(ownerID)

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)
	deps.folderRepo.On("FindByID", ctx, destFolder.ID).Return(destFolder, nil)
	deps.fileRepo.On("ExistsByNameAndFolder", ctx, file.Name, destFolder.ID).Return(true, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.MoveFileInput{
		FileID:      file.ID,
		NewFolderID: destFolder.ID,
		UserID:      ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestMoveFileCommand_Execute_UploadingFile_ReturnsError(t *testing.T) {
	ctx := context.Background()
	deps := newMoveFileTestDeps(t)

	ownerID := uuid.New()
	sourceFolderID := uuid.New()

	fileID := uuid.New()
	name, _ := valueobject.NewFileName("uploading.pdf")
	mimeType, _ := valueobject.NewMimeType("application/pdf")
	storageKey := valueobject.NewStorageKey(fileID)
	uploadingFile := entity.ReconstructFile(
		fileID,
		sourceFolderID,
		ownerID,
		ownerID,
		name,
		mimeType,
		2048,
		storageKey,
		1,
		entity.FileStatusUploading,
		time.Now(),
		time.Now(),
	)

	destFolder := newFolderEntity(ownerID)

	deps.fileRepo.On("FindByID", ctx, uploadingFile.ID).Return(uploadingFile, nil)
	deps.folderRepo.On("FindByID", ctx, destFolder.ID).Return(destFolder, nil)
	deps.fileRepo.On("ExistsByNameAndFolder", ctx, uploadingFile.Name, destFolder.ID).Return(false, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.MoveFileInput{
		FileID:      uploadingFile.ID,
		NewFolderID: destFolder.ID,
		UserID:      ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	assert.ErrorIs(t, err, entity.ErrFileNotActive)
}
