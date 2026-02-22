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

type trashFileTestDeps struct {
	fileRepo                *mocks.MockFileRepository
	fileVersionRepo         *mocks.MockFileVersionRepository
	folderRepo              *mocks.MockFolderRepository
	folderClosureRepo       *mocks.MockFolderClosureRepository
	archivedFileRepo        *mocks.MockArchivedFileRepository
	archivedFileVersionRepo *mocks.MockArchivedFileVersionRepository
	txManager               *mocks.MockTransactionManager
}

func newTrashFileTestDeps(t *testing.T) *trashFileTestDeps {
	t.Helper()
	return &trashFileTestDeps{
		fileRepo:                mocks.NewMockFileRepository(t),
		fileVersionRepo:         mocks.NewMockFileVersionRepository(t),
		folderRepo:              mocks.NewMockFolderRepository(t),
		folderClosureRepo:       mocks.NewMockFolderClosureRepository(t),
		archivedFileRepo:        mocks.NewMockArchivedFileRepository(t),
		archivedFileVersionRepo: mocks.NewMockArchivedFileVersionRepository(t),
		txManager:               mocks.NewMockTransactionManager(t),
	}
}

func (d *trashFileTestDeps) newCommand() *command.TrashFileCommand {
	return command.NewTrashFileCommand(
		d.fileRepo,
		d.fileVersionRepo,
		d.folderRepo,
		d.folderClosureRepo,
		d.archivedFileRepo,
		d.archivedFileVersionRepo,
		d.txManager,
	)
}

func newActiveFileForTrash(ownerID, folderID uuid.UUID) *entity.File {
	fileID := uuid.New()
	name, _ := valueobject.NewFileName("test.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(fileID)
	return entity.ReconstructFile(
		fileID, folderID, ownerID, ownerID,
		name, mimeType, 1024, storageKey, 1,
		entity.FileStatusActive, time.Now(), time.Now(),
	)
}

func newTrashFolderEntity(id uuid.UUID, ownerID uuid.UUID) *entity.Folder {
	name, _ := valueobject.NewFolderName("my-folder")
	return entity.ReconstructFolder(
		id, name, nil, ownerID, ownerID, 0,
		entity.FolderStatusActive, time.Now(), time.Now(),
	)
}

func TestTrashFileCommand_Execute_Success_ArchivesFile(t *testing.T) {
	ctx := context.Background()
	deps := newTrashFileTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()
	file := newActiveFileForTrash(ownerID, folderID)

	folder := newTrashFolderEntity(folderID, ownerID)

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)
	deps.folderClosureRepo.On("FindAncestorIDs", ctx, folderID).Return([]uuid.UUID{}, nil)
	deps.folderRepo.On("FindByID", ctx, folderID).Return(folder, nil)
	deps.archivedFileRepo.On("Create", ctx, mock.AnythingOfType("*entity.ArchivedFile")).Return(nil)
	deps.fileVersionRepo.On("FindByFileID", ctx, file.ID).Return([]*entity.FileVersion{}, nil)
	deps.fileVersionRepo.On("DeleteByFileID", ctx, file.ID).Return(nil)
	deps.fileRepo.On("Delete", ctx, file.ID).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.TrashFileInput{
		FileID: file.ID,
		UserID: ownerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.NotEqual(t, uuid.Nil, output.ArchivedFileID)
}

func TestTrashFileCommand_Execute_OutputIncludesExpiresAt(t *testing.T) {
	ctx := context.Background()
	deps := newTrashFileTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()
	file := newActiveFileForTrash(ownerID, folderID)

	folder := newTrashFolderEntity(folderID, ownerID)

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)
	deps.folderClosureRepo.On("FindAncestorIDs", ctx, folderID).Return([]uuid.UUID{}, nil)
	deps.folderRepo.On("FindByID", ctx, folderID).Return(folder, nil)
	deps.archivedFileRepo.On("Create", ctx, mock.AnythingOfType("*entity.ArchivedFile")).Return(nil)
	deps.fileVersionRepo.On("FindByFileID", ctx, file.ID).Return([]*entity.FileVersion{}, nil)
	deps.fileVersionRepo.On("DeleteByFileID", ctx, file.ID).Return(nil)
	deps.fileRepo.On("Delete", ctx, file.ID).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.TrashFileInput{
		FileID: file.ID,
		UserID: ownerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.False(t, output.ExpiresAt.IsZero(), "ExpiresAt should not be zero")
	assert.True(t, output.ExpiresAt.After(time.Now()), "ExpiresAt should be in the future")
}

func TestTrashFileCommand_Execute_NotOwner_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newTrashFileTestDeps(t)

	ownerID := uuid.New()
	otherUserID := uuid.New()
	folderID := uuid.New()
	file := newActiveFileForTrash(ownerID, folderID)

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.TrashFileInput{
		FileID: file.ID,
		UserID: otherUserID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestTrashFileCommand_Execute_NotActive_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newTrashFileTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()
	fileID := uuid.New()
	name, _ := valueobject.NewFileName("uploading.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(fileID)
	uploadingFile := entity.ReconstructFile(
		fileID, folderID, ownerID, ownerID,
		name, mimeType, 1024, storageKey, 1,
		entity.FileStatusUploading, time.Now(), time.Now(),
	)

	deps.fileRepo.On("FindByID", ctx, uploadingFile.ID).Return(uploadingFile, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.TrashFileInput{
		FileID: uploadingFile.ID,
		UserID: ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}
