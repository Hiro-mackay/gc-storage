package command_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type emptyTrashTestDeps struct {
	archivedFileRepo        *mocks.MockArchivedFileRepository
	archivedFileVersionRepo *mocks.MockArchivedFileVersionRepository
	storageService          *mocks.MockStorageService
	txManager               *mocks.MockTransactionManager
}

func newEmptyTrashTestDeps(t *testing.T) *emptyTrashTestDeps {
	t.Helper()
	return &emptyTrashTestDeps{
		archivedFileRepo:        mocks.NewMockArchivedFileRepository(t),
		archivedFileVersionRepo: mocks.NewMockArchivedFileVersionRepository(t),
		storageService:          mocks.NewMockStorageService(t),
		txManager:               mocks.NewMockTransactionManager(t),
	}
}

func (d *emptyTrashTestDeps) newCommand() *command.EmptyTrashCommand {
	return command.NewEmptyTrashCommand(
		d.archivedFileRepo,
		d.archivedFileVersionRepo,
		d.storageService,
		d.txManager,
	)
}

func newArchivedFileEntry(ownerID uuid.UUID, name string) *entity.ArchivedFile {
	fileID := uuid.New()
	fileName, _ := valueobject.NewFileName(name)
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(fileID)
	return entity.ReconstructArchivedFile(
		uuid.New(),
		fileID,
		uuid.New(),
		"/docs/"+name,
		fileName,
		mimeType,
		256,
		ownerID,
		ownerID,
		storageKey,
		time.Now().Add(-time.Hour),
		ownerID,
		time.Now().Add(29*24*time.Hour),
	)
}

func TestEmptyTrashCommand_Execute_EmptyTrash_ReturnsZeroDeleted(t *testing.T) {
	ctx := context.Background()
	deps := newEmptyTrashTestDeps(t)

	ownerID := uuid.New()

	deps.archivedFileRepo.On("FindByOwner", ctx, ownerID).Return([]*entity.ArchivedFile{}, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.EmptyTrashInput{
		OwnerID: ownerID,
		UserID:  ownerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 0, output.DeletedCount)
}

func TestEmptyTrashCommand_Execute_WithFiles_DeletesAllAndReturnsCount(t *testing.T) {
	ctx := context.Background()
	deps := newEmptyTrashTestDeps(t)

	ownerID := uuid.New()
	file1 := newArchivedFileEntry(ownerID, "a.txt")
	file2 := newArchivedFileEntry(ownerID, "b.txt")
	archivedFiles := []*entity.ArchivedFile{file1, file2}

	deps.archivedFileRepo.On("FindByOwner", ctx, ownerID).Return(archivedFiles, nil)
	deps.archivedFileVersionRepo.On("DeleteByArchivedFileID", ctx, file1.ID).Return(nil)
	deps.archivedFileRepo.On("Delete", ctx, file1.ID).Return(nil)
	deps.archivedFileVersionRepo.On("DeleteByArchivedFileID", ctx, file2.ID).Return(nil)
	deps.archivedFileRepo.On("Delete", ctx, file2.ID).Return(nil)
	deps.storageService.On("DeleteObject", ctx, file1.StorageKey.String()).Return(nil)
	deps.storageService.On("DeleteObject", ctx, file2.StorageKey.String()).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.EmptyTrashInput{
		OwnerID: ownerID,
		UserID:  ownerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 2, output.DeletedCount)
}

func TestEmptyTrashCommand_Execute_NotOwner_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newEmptyTrashTestDeps(t)

	ownerID := uuid.New()
	otherUserID := uuid.New()

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.EmptyTrashInput{
		OwnerID: ownerID,
		UserID:  otherUserID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}
