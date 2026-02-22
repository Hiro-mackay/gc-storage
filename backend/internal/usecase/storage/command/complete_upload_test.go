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

type completeUploadTestDeps struct {
	fileRepo          *mocks.MockFileRepository
	fileVersionRepo   *mocks.MockFileVersionRepository
	uploadSessionRepo *mocks.MockUploadSessionRepository
	uploadPartRepo    *mocks.MockUploadPartRepository
	txManager         *mocks.MockTransactionManager
}

func newCompleteUploadTestDeps(t *testing.T) *completeUploadTestDeps {
	t.Helper()
	return &completeUploadTestDeps{
		fileRepo:          mocks.NewMockFileRepository(t),
		fileVersionRepo:   mocks.NewMockFileVersionRepository(t),
		uploadSessionRepo: mocks.NewMockUploadSessionRepository(t),
		uploadPartRepo:    mocks.NewMockUploadPartRepository(t),
		txManager:         mocks.NewMockTransactionManager(t),
	}
}

func (d *completeUploadTestDeps) newCommand() *command.CompleteUploadCommand {
	return command.NewCompleteUploadCommand(
		d.fileRepo,
		d.fileVersionRepo,
		d.uploadSessionRepo,
		d.uploadPartRepo,
		d.txManager,
	)
}

func newUploadingFileEntity(ownerID, folderID uuid.UUID) *entity.File {
	name, _ := valueobject.NewFileName("upload.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	fileID := uuid.New()
	storageKey := valueobject.NewStorageKey(fileID)
	return entity.ReconstructFile(
		fileID, folderID, ownerID, ownerID,
		name, mimeType, 1024, storageKey, 1,
		entity.FileStatusUploading, time.Now(), time.Now(),
	)
}

func newPendingSession(fileID, ownerID, folderID uuid.UUID) *entity.UploadSession {
	fileName, _ := valueobject.NewFileName("upload.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(fileID)
	return entity.ReconstructUploadSession(
		uuid.New(), fileID, ownerID, ownerID, folderID,
		fileName, mimeType, 1024, storageKey,
		nil, false, 1, 0,
		entity.UploadSessionStatusPending,
		time.Now(), time.Now(), time.Now().Add(24*time.Hour),
	)
}

func newMultipartSession(fileID, ownerID, folderID uuid.UUID, totalParts, uploadedParts int) *entity.UploadSession {
	fileName, _ := valueobject.NewFileName("upload.mp4")
	mimeType, _ := valueobject.NewMimeType("video/mp4")
	storageKey := valueobject.NewStorageKey(fileID)
	uploadID := "minio-upload-id"
	status := entity.UploadSessionStatusPending
	if uploadedParts > 0 {
		status = entity.UploadSessionStatusInProgress
	}
	return entity.ReconstructUploadSession(
		uuid.New(), fileID, ownerID, ownerID, folderID,
		fileName, mimeType, int64(totalParts)*5*1024*1024, storageKey,
		&uploadID, true, totalParts, uploadedParts,
		status,
		time.Now(), time.Now(), time.Now().Add(24*time.Hour),
	)
}

func TestCompleteUploadCommand_Execute_SinglePart_ReturnsCompleted(t *testing.T) {
	ctx := context.Background()
	deps := newCompleteUploadTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()
	file := newUploadingFileEntity(ownerID, folderID)
	session := newPendingSession(file.ID, ownerID, folderID)

	storageKey := file.StorageKey.String()
	input := command.CompleteUploadInput{
		StorageKey:     storageKey,
		MinioVersionID: "v1",
		Size:           1024,
		ETag:           "etag-abc",
	}

	deps.uploadSessionRepo.On("FindByStorageKey", ctx, mock.AnythingOfType("valueobject.StorageKey")).Return(session, nil)
	deps.fileRepo.On("FindByID", ctx, session.FileID).Return(file, nil)
	deps.fileVersionRepo.On("Create", ctx, mock.AnythingOfType("*entity.FileVersion")).Return(nil)
	deps.fileRepo.On("Update", ctx, file).Return(nil)
	deps.fileRepo.On("UpdateStatus", ctx, file.ID, entity.FileStatusActive).Return(nil)
	deps.uploadSessionRepo.On("Update", ctx, session).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.True(t, output.Completed)
	assert.Equal(t, session.FileID, output.FileID)
	assert.Equal(t, session.ID, output.SessionID)
}

func TestCompleteUploadCommand_Execute_AlreadyCompleted_ReturnsIdempotentSuccess(t *testing.T) {
	ctx := context.Background()
	deps := newCompleteUploadTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()
	fileID := uuid.New()
	fileName, _ := valueobject.NewFileName("upload.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(fileID)

	completedSession := entity.ReconstructUploadSession(
		uuid.New(), fileID, ownerID, ownerID, folderID,
		fileName, mimeType, 1024, storageKey,
		nil, false, 1, 1,
		entity.UploadSessionStatusCompleted,
		time.Now(), time.Now(), time.Now().Add(24*time.Hour),
	)

	input := command.CompleteUploadInput{
		StorageKey:     storageKey.String(),
		MinioVersionID: "v1",
		Size:           1024,
		ETag:           "etag-abc",
	}

	deps.uploadSessionRepo.On("FindByStorageKey", ctx, mock.AnythingOfType("valueobject.StorageKey")).Return(completedSession, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.True(t, output.Completed)
	assert.Equal(t, fileID, output.FileID)
}

func TestCompleteUploadCommand_Execute_AbortedSession_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newCompleteUploadTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()
	fileID := uuid.New()
	fileName, _ := valueobject.NewFileName("upload.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(fileID)

	abortedSession := entity.ReconstructUploadSession(
		uuid.New(), fileID, ownerID, ownerID, folderID,
		fileName, mimeType, 1024, storageKey,
		nil, false, 1, 0,
		entity.UploadSessionStatusAborted,
		time.Now(), time.Now(), time.Now().Add(24*time.Hour),
	)

	input := command.CompleteUploadInput{
		StorageKey:     storageKey.String(),
		MinioVersionID: "v1",
		Size:           1024,
		ETag:           "etag-abc",
	}

	deps.uploadSessionRepo.On("FindByStorageKey", ctx, mock.AnythingOfType("valueobject.StorageKey")).Return(abortedSession, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestCompleteUploadCommand_Execute_InvalidStorageKey_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newCompleteUploadTestDeps(t)

	input := command.CompleteUploadInput{
		StorageKey:     "invalid-key",
		MinioVersionID: "v1",
		Size:           1024,
		ETag:           "etag-abc",
	}

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestCompleteUploadCommand_Execute_MultipartPartial_ReturnsNotCompleted(t *testing.T) {
	ctx := context.Background()
	deps := newCompleteUploadTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()
	fileID := uuid.New()
	// 2-part upload, 0 uploaded so far -> after increment: 1 of 2 -> not complete
	session := newMultipartSession(fileID, ownerID, folderID, 2, 0)

	input := command.CompleteUploadInput{
		StorageKey:     session.StorageKey.String(),
		MinioVersionID: "v1",
		Size:           5 * 1024 * 1024,
		ETag:           "etag-part1",
	}

	deps.uploadSessionRepo.On("FindByStorageKey", ctx, mock.AnythingOfType("valueobject.StorageKey")).Return(session, nil)
	deps.fileRepo.On("FindByID", ctx, session.FileID).Return(newUploadingFileEntity(ownerID, folderID), nil)
	deps.uploadPartRepo.On("Create", ctx, mock.AnythingOfType("*entity.UploadPart")).Return(nil)
	deps.uploadSessionRepo.On("Update", ctx, session).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.False(t, output.Completed)
}

func TestCompleteUploadCommand_Execute_MultipartFinalPart_ReturnsCompleted(t *testing.T) {
	ctx := context.Background()
	deps := newCompleteUploadTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()
	fileID := uuid.New()
	// 2-part upload, 1 already uploaded -> after increment: 2 of 2 -> complete
	session := newMultipartSession(fileID, ownerID, folderID, 2, 1)
	file := newUploadingFileEntity(ownerID, folderID)

	input := command.CompleteUploadInput{
		StorageKey:     session.StorageKey.String(),
		MinioVersionID: "v1",
		Size:           5 * 1024 * 1024,
		ETag:           "etag-part2",
	}

	deps.uploadSessionRepo.On("FindByStorageKey", ctx, mock.AnythingOfType("valueobject.StorageKey")).Return(session, nil)
	deps.fileRepo.On("FindByID", ctx, session.FileID).Return(file, nil)
	deps.uploadPartRepo.On("Create", ctx, mock.AnythingOfType("*entity.UploadPart")).Return(nil)
	deps.fileVersionRepo.On("Create", ctx, mock.AnythingOfType("*entity.FileVersion")).Return(nil)
	deps.fileRepo.On("Update", ctx, file).Return(nil)
	deps.fileRepo.On("UpdateStatus", ctx, file.ID, entity.FileStatusActive).Return(nil)
	deps.uploadSessionRepo.On("Update", ctx, session).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.True(t, output.Completed)
}
