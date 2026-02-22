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

type abortUploadTestDeps struct {
	uploadSessionRepo *mocks.MockUploadSessionRepository
	fileRepo          *mocks.MockFileRepository
	storageService    *mocks.MockStorageService
	txManager         *mocks.MockTransactionManager
}

func newAbortUploadTestDeps(t *testing.T) *abortUploadTestDeps {
	t.Helper()
	return &abortUploadTestDeps{
		uploadSessionRepo: mocks.NewMockUploadSessionRepository(t),
		fileRepo:          mocks.NewMockFileRepository(t),
		storageService:    mocks.NewMockStorageService(t),
		txManager:         mocks.NewMockTransactionManager(t),
	}
}

func (d *abortUploadTestDeps) newCommand() *command.AbortUploadCommand {
	return command.NewAbortUploadCommand(
		d.uploadSessionRepo,
		d.fileRepo,
		d.storageService,
		d.txManager,
	)
}

func newSinglePartPendingSession(ownerID, folderID uuid.UUID) *entity.UploadSession {
	fileID := uuid.New()
	fileName, _ := valueobject.NewFileName("file.txt")
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

func newMultipartPendingSession(ownerID, folderID uuid.UUID) *entity.UploadSession {
	fileID := uuid.New()
	fileName, _ := valueobject.NewFileName("video.mp4")
	mimeType, _ := valueobject.NewMimeType("video/mp4")
	storageKey := valueobject.NewStorageKey(fileID)
	uploadID := "minio-upload-id"
	return entity.ReconstructUploadSession(
		uuid.New(), fileID, ownerID, ownerID, folderID,
		fileName, mimeType, 10*1024*1024, storageKey,
		&uploadID, true, 2, 0,
		entity.UploadSessionStatusPending,
		time.Now(), time.Now(), time.Now().Add(24*time.Hour),
	)
}

func TestAbortUploadCommand_Execute_SinglePart_ReturnsAborted(t *testing.T) {
	ctx := context.Background()
	deps := newAbortUploadTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()
	session := newSinglePartPendingSession(ownerID, folderID)
	file := newUploadingFileEntity(ownerID, folderID)
	// Override fileID to match session
	file = entity.ReconstructFile(
		session.FileID, folderID, ownerID, ownerID,
		file.Name, file.MimeType, file.Size, file.StorageKey, 1,
		entity.FileStatusUploading, time.Now(), time.Now(),
	)

	input := command.AbortUploadInput{
		SessionID: session.ID,
		UserID:    ownerID,
	}

	deps.uploadSessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)
	deps.uploadSessionRepo.On("Update", ctx, session).Return(nil)
	deps.fileRepo.On("FindByID", ctx, session.FileID).Return(file, nil)
	deps.fileRepo.On("Update", ctx, file).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.True(t, output.Aborted)
	assert.Equal(t, session.ID, output.SessionID)
}

func TestAbortUploadCommand_Execute_Multipart_CallsAbortMultipartUpload(t *testing.T) {
	ctx := context.Background()
	deps := newAbortUploadTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()
	session := newMultipartPendingSession(ownerID, folderID)
	file := entity.ReconstructFile(
		session.FileID, folderID, ownerID, ownerID,
		session.FileName, session.MimeType, session.TotalSize, session.StorageKey, 1,
		entity.FileStatusUploading, time.Now(), time.Now(),
	)

	input := command.AbortUploadInput{
		SessionID: session.ID,
		UserID:    ownerID,
	}

	deps.uploadSessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)
	deps.uploadSessionRepo.On("Update", ctx, session).Return(nil)
	deps.fileRepo.On("FindByID", ctx, session.FileID).Return(file, nil)
	deps.fileRepo.On("Update", ctx, file).Return(nil)
	deps.storageService.On("AbortMultipartUpload", ctx, session.StorageKey.String(), *session.MinioUploadID).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.True(t, output.Aborted)
}

func TestAbortUploadCommand_Execute_NotOwner_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newAbortUploadTestDeps(t)

	ownerID := uuid.New()
	differentUserID := uuid.New()
	folderID := uuid.New()
	session := newSinglePartPendingSession(ownerID, folderID)

	input := command.AbortUploadInput{
		SessionID: session.ID,
		UserID:    differentUserID,
	}

	deps.uploadSessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestAbortUploadCommand_Execute_AlreadyCompleted_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newAbortUploadTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()
	fileID := uuid.New()
	fileName, _ := valueobject.NewFileName("file.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(fileID)

	completedSession := entity.ReconstructUploadSession(
		uuid.New(), fileID, ownerID, ownerID, folderID,
		fileName, mimeType, 1024, storageKey,
		nil, false, 1, 1,
		entity.UploadSessionStatusCompleted,
		time.Now(), time.Now(), time.Now().Add(24*time.Hour),
	)

	input := command.AbortUploadInput{
		SessionID: completedSession.ID,
		UserID:    ownerID,
	}

	deps.uploadSessionRepo.On("FindByID", ctx, completedSession.ID).Return(completedSession, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestAbortUploadCommand_Execute_AlreadyAborted_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newAbortUploadTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()
	fileID := uuid.New()
	fileName, _ := valueobject.NewFileName("file.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(fileID)

	abortedSession := entity.ReconstructUploadSession(
		uuid.New(), fileID, ownerID, ownerID, folderID,
		fileName, mimeType, 1024, storageKey,
		nil, false, 1, 0,
		entity.UploadSessionStatusAborted,
		time.Now(), time.Now(), time.Now().Add(24*time.Hour),
	)

	input := command.AbortUploadInput{
		SessionID: abortedSession.ID,
		UserID:    ownerID,
	}

	deps.uploadSessionRepo.On("FindByID", ctx, abortedSession.ID).Return(abortedSession, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestAbortUploadCommand_Execute_SessionNotFound_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newAbortUploadTestDeps(t)

	sessionID := uuid.New()
	notFoundErr := apperror.NewNotFoundError("upload session")

	input := command.AbortUploadInput{
		SessionID: sessionID,
		UserID:    uuid.New(),
	}

	deps.uploadSessionRepo.On("FindByID", ctx, sessionID).Return(nil, notFoundErr)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}
