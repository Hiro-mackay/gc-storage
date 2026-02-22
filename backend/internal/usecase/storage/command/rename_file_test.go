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

type renameFileTestDeps struct {
	fileRepo *mocks.MockFileRepository
}

func newRenameFileTestDeps(t *testing.T) *renameFileTestDeps {
	t.Helper()
	return &renameFileTestDeps{
		fileRepo: mocks.NewMockFileRepository(t),
	}
}

func (d *renameFileTestDeps) newCommand() *command.RenameFileCommand {
	return command.NewRenameFileCommand(d.fileRepo)
}

func newActiveFileEntity(ownerID, folderID uuid.UUID) *entity.File {
	name, _ := valueobject.NewFileName("original.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(uuid.New())
	return entity.ReconstructFile(
		uuid.New(), folderID, ownerID, ownerID,
		name, mimeType, 1024, storageKey, 1,
		entity.FileStatusActive, time.Now(), time.Now(),
	)
}

func TestRenameFileCommand_Execute_ValidInput_ReturnsOutput(t *testing.T) {
	ctx := context.Background()
	deps := newRenameFileTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()
	file := newActiveFileEntity(ownerID, folderID)

	input := command.RenameFileInput{
		FileID:  file.ID,
		NewName: "renamed.txt",
		UserID:  ownerID,
	}

	deps.fileRepo.On("FindByID", ctx, input.FileID).Return(file, nil)
	deps.fileRepo.On("ExistsByNameAndFolder", ctx, mock.AnythingOfType("valueobject.FileName"), folderID).Return(false, nil)
	deps.fileRepo.On("Update", ctx, file).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, file.ID, output.FileID)
	assert.Equal(t, "renamed.txt", output.Name)
}

func TestRenameFileCommand_Execute_InvalidName_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newRenameFileTestDeps(t)

	input := command.RenameFileInput{
		FileID:  uuid.New(),
		NewName: "",
		UserID:  uuid.New(),
	}

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestRenameFileCommand_Execute_ForbiddenChar_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newRenameFileTestDeps(t)

	input := command.RenameFileInput{
		FileID:  uuid.New(),
		NewName: "invalid/name.txt",
		UserID:  uuid.New(),
	}

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestRenameFileCommand_Execute_FileNotFound_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newRenameFileTestDeps(t)

	fileID := uuid.New()
	notFoundErr := apperror.NewNotFoundError("file")

	input := command.RenameFileInput{
		FileID:  fileID,
		NewName: "renamed.txt",
		UserID:  uuid.New(),
	}

	deps.fileRepo.On("FindByID", ctx, fileID).Return(nil, notFoundErr)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestRenameFileCommand_Execute_NotOwner_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newRenameFileTestDeps(t)

	ownerID := uuid.New()
	differentUserID := uuid.New()
	folderID := uuid.New()
	file := newActiveFileEntity(ownerID, folderID)

	input := command.RenameFileInput{
		FileID:  file.ID,
		NewName: "renamed.txt",
		UserID:  differentUserID,
	}

	deps.fileRepo.On("FindByID", ctx, input.FileID).Return(file, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestRenameFileCommand_Execute_DuplicateName_ReturnsConflict(t *testing.T) {
	ctx := context.Background()
	deps := newRenameFileTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()
	file := newActiveFileEntity(ownerID, folderID)

	input := command.RenameFileInput{
		FileID:  file.ID,
		NewName: "duplicate.txt",
		UserID:  ownerID,
	}

	deps.fileRepo.On("FindByID", ctx, input.FileID).Return(file, nil)
	deps.fileRepo.On("ExistsByNameAndFolder", ctx, mock.AnythingOfType("valueobject.FileName"), folderID).Return(true, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestRenameFileCommand_Execute_UploadingFile_ReturnsError(t *testing.T) {
	ctx := context.Background()
	deps := newRenameFileTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()

	name, _ := valueobject.NewFileName("uploading.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(uuid.New())
	uploadingFile := entity.ReconstructFile(
		uuid.New(), folderID, ownerID, ownerID,
		name, mimeType, 1024, storageKey, 1,
		entity.FileStatusUploading, time.Now(), time.Now(),
	)

	input := command.RenameFileInput{
		FileID:  uploadingFile.ID,
		NewName: "renamed.txt",
		UserID:  ownerID,
	}

	deps.fileRepo.On("FindByID", ctx, input.FileID).Return(uploadingFile, nil)
	deps.fileRepo.On("ExistsByNameAndFolder", ctx, mock.AnythingOfType("valueobject.FileName"), folderID).Return(false, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}
