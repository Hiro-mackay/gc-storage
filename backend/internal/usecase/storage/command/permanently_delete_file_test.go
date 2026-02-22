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

type permDeleteTestDeps struct {
	archivedFileRepo        *mocks.MockArchivedFileRepository
	archivedFileVersionRepo *mocks.MockArchivedFileVersionRepository
	storageService          *mocks.MockStorageService
	txManager               *mocks.MockTransactionManager
}

func newPermDeleteTestDeps(t *testing.T) *permDeleteTestDeps {
	t.Helper()
	return &permDeleteTestDeps{
		archivedFileRepo:        mocks.NewMockArchivedFileRepository(t),
		archivedFileVersionRepo: mocks.NewMockArchivedFileVersionRepository(t),
		storageService:          mocks.NewMockStorageService(t),
		txManager:               mocks.NewMockTransactionManager(t),
	}
}

func (d *permDeleteTestDeps) newCommand() *command.PermanentlyDeleteFileCommand {
	return command.NewPermanentlyDeleteFileCommand(
		d.archivedFileRepo,
		d.archivedFileVersionRepo,
		d.storageService,
		d.txManager,
	)
}

func newArchivedFileForDelete(ownerID uuid.UUID) *entity.ArchivedFile {
	fileID := uuid.New()
	name, _ := valueobject.NewFileName("delete-me.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(fileID)
	return entity.ReconstructArchivedFile(
		uuid.New(),
		fileID,
		uuid.New(),
		"/docs/delete-me.txt",
		name,
		mimeType,
		512,
		ownerID,
		ownerID,
		storageKey,
		time.Now().Add(-time.Hour),
		ownerID,
		time.Now().Add(29*24*time.Hour),
	)
}

func TestPermanentlyDeleteFileCommand_Execute_ValidDelete_DeletesDBAndStorage(t *testing.T) {
	ctx := context.Background()
	deps := newPermDeleteTestDeps(t)

	ownerID := uuid.New()
	archivedFile := newArchivedFileForDelete(ownerID)
	storageKey := archivedFile.StorageKey.String()

	deps.archivedFileRepo.On("FindByID", ctx, archivedFile.ID).Return(archivedFile, nil)
	deps.archivedFileVersionRepo.On("DeleteByArchivedFileID", ctx, archivedFile.ID).Return(nil)
	deps.archivedFileRepo.On("Delete", ctx, archivedFile.ID).Return(nil)
	deps.storageService.On("DeleteObject", ctx, storageKey).Return(nil)

	cmd := deps.newCommand()
	err := cmd.Execute(ctx, command.PermanentlyDeleteFileInput{
		ArchivedFileID: archivedFile.ID,
		UserID:         ownerID,
	})

	require.NoError(t, err)
	deps.storageService.AssertExpectations(t)
}

func TestPermanentlyDeleteFileCommand_Execute_NotOwner_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newPermDeleteTestDeps(t)

	ownerID := uuid.New()
	otherUserID := uuid.New()
	archivedFile := newArchivedFileForDelete(ownerID)

	deps.archivedFileRepo.On("FindByID", ctx, archivedFile.ID).Return(archivedFile, nil)

	cmd := deps.newCommand()
	err := cmd.Execute(ctx, command.PermanentlyDeleteFileInput{
		ArchivedFileID: archivedFile.ID,
		UserID:         otherUserID,
	})

	require.Error(t, err)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestPermanentlyDeleteFileCommand_Execute_FileNotFound_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newPermDeleteTestDeps(t)

	archivedFileID := uuid.New()
	notFoundErr := apperror.NewNotFoundError("archived file")

	deps.archivedFileRepo.On("FindByID", ctx, archivedFileID).Return(nil, notFoundErr)

	cmd := deps.newCommand()
	err := cmd.Execute(ctx, command.PermanentlyDeleteFileInput{
		ArchivedFileID: archivedFileID,
		UserID:         uuid.New(),
	})

	require.Error(t, err)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}
