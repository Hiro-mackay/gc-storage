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
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type initiateUploadTestDeps struct {
	fileRepo          *mocks.MockFileRepository
	folderRepo        *mocks.MockFolderRepository
	uploadSessionRepo *mocks.MockUploadSessionRepository
	storageService    *mocks.MockStorageService
	txManager         *mocks.MockTransactionManager
}

func newInitiateUploadTestDeps(t *testing.T) *initiateUploadTestDeps {
	t.Helper()
	return &initiateUploadTestDeps{
		fileRepo:          mocks.NewMockFileRepository(t),
		folderRepo:        mocks.NewMockFolderRepository(t),
		uploadSessionRepo: mocks.NewMockUploadSessionRepository(t),
		storageService:    mocks.NewMockStorageService(t),
		txManager:         mocks.NewMockTransactionManager(t),
	}
}

func (d *initiateUploadTestDeps) newCommand() *command.InitiateUploadCommand {
	return command.NewInitiateUploadCommand(
		d.fileRepo,
		d.folderRepo,
		d.uploadSessionRepo,
		d.storageService,
		d.txManager,
	)
}

func newActiveFolderEntity(ownerID uuid.UUID) *entity.Folder {
	folderName, _ := valueobject.NewFolderName("test-folder")
	return entity.ReconstructFolder(
		uuid.New(), folderName, nil, ownerID, ownerID, 0,
		entity.FolderStatusActive, time.Now(), time.Now(),
	)
}

func TestInitiateUploadCommand_Execute_SinglePart_ReturnsOutput(t *testing.T) {
	ctx := context.Background()
	deps := newInitiateUploadTestDeps(t)

	ownerID := uuid.New()
	folder := newActiveFolderEntity(ownerID)

	// single-part: size < 5MB
	input := command.InitiateUploadInput{
		FolderID: folder.ID,
		FileName: "document.pdf",
		MimeType: "application/pdf",
		Size:     1024 * 1024, // 1MB
		OwnerID:  ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.fileRepo.On("ExistsByNameAndFolder", ctx, mock.AnythingOfType("valueobject.FileName"), folder.ID).Return(false, nil)
	deps.fileRepo.On("Create", ctx, mock.AnythingOfType("*entity.File")).Return(nil)
	deps.uploadSessionRepo.On("Create", ctx, mock.AnythingOfType("*entity.UploadSession")).Return(nil)
	deps.storageService.On("GeneratePutURL", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).
		Return(&service.PresignedURL{URL: "https://example.com/upload", ExpiresAt: time.Now().Add(time.Hour)}, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.False(t, output.IsMultipart)
	assert.Len(t, output.UploadURLs, 1)
	assert.NotEqual(t, uuid.Nil, output.SessionID)
	assert.NotEqual(t, uuid.Nil, output.FileID)
}

func TestInitiateUploadCommand_Execute_Multipart_ReturnsOutput(t *testing.T) {
	ctx := context.Background()
	deps := newInitiateUploadTestDeps(t)

	ownerID := uuid.New()
	folder := newActiveFolderEntity(ownerID)

	// multipart: size >= 5MB (2 parts: 10MB)
	const fileSize = 10 * 1024 * 1024 // 10MB
	input := command.InitiateUploadInput{
		FolderID: folder.ID,
		FileName: "video.mp4",
		MimeType: "video/mp4",
		Size:     fileSize,
		OwnerID:  ownerID,
	}

	uploadID := "minio-upload-id"
	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.fileRepo.On("ExistsByNameAndFolder", ctx, mock.AnythingOfType("valueobject.FileName"), folder.ID).Return(false, nil)
	deps.storageService.On("CreateMultipartUpload", ctx, mock.AnythingOfType("string")).Return(uploadID, nil)
	deps.fileRepo.On("Create", ctx, mock.AnythingOfType("*entity.File")).Return(nil)
	deps.uploadSessionRepo.On("Create", ctx, mock.AnythingOfType("*entity.UploadSession")).Return(nil)
	// 2 parts for 10MB file
	deps.storageService.On("GeneratePartUploadURL", ctx, mock.AnythingOfType("string"), uploadID, 1).
		Return(&service.MultipartUploadURL{PartNumber: 1, URL: "https://example.com/part1", ExpiresAt: time.Now().Add(time.Hour)}, nil)
	deps.storageService.On("GeneratePartUploadURL", ctx, mock.AnythingOfType("string"), uploadID, 2).
		Return(&service.MultipartUploadURL{PartNumber: 2, URL: "https://example.com/part2", ExpiresAt: time.Now().Add(time.Hour)}, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.True(t, output.IsMultipart)
	assert.Len(t, output.UploadURLs, 2)
	assert.NotEqual(t, uuid.Nil, output.SessionID)
}

func TestInitiateUploadCommand_Execute_EmptyFileName_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newInitiateUploadTestDeps(t)

	input := command.InitiateUploadInput{
		FolderID: uuid.New(),
		FileName: "",
		MimeType: "text/plain",
		Size:     1024,
		OwnerID:  uuid.New(),
	}

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestInitiateUploadCommand_Execute_EmptyMimeType_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newInitiateUploadTestDeps(t)

	input := command.InitiateUploadInput{
		FolderID: uuid.New(),
		FileName: "file.txt",
		MimeType: "",
		Size:     1024,
		OwnerID:  uuid.New(),
	}

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestInitiateUploadCommand_Execute_FolderNotFound_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newInitiateUploadTestDeps(t)

	folderID := uuid.New()
	notFoundErr := apperror.NewNotFoundError("folder")

	input := command.InitiateUploadInput{
		FolderID: folderID,
		FileName: "file.txt",
		MimeType: "text/plain",
		Size:     1024,
		OwnerID:  uuid.New(),
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

func TestInitiateUploadCommand_Execute_NotFolderOwner_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newInitiateUploadTestDeps(t)

	ownerID := uuid.New()
	differentUserID := uuid.New()
	folder := newActiveFolderEntity(ownerID)

	input := command.InitiateUploadInput{
		FolderID: folder.ID,
		FileName: "file.txt",
		MimeType: "text/plain",
		Size:     1024,
		OwnerID:  differentUserID,
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

func TestInitiateUploadCommand_Execute_DuplicateFileName_ReturnsConflict(t *testing.T) {
	ctx := context.Background()
	deps := newInitiateUploadTestDeps(t)

	ownerID := uuid.New()
	folder := newActiveFolderEntity(ownerID)

	input := command.InitiateUploadInput{
		FolderID: folder.ID,
		FileName: "existing.txt",
		MimeType: "text/plain",
		Size:     1024,
		OwnerID:  ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.fileRepo.On("ExistsByNameAndFolder", ctx, mock.AnythingOfType("valueobject.FileName"), folder.ID).Return(true, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeConflict, appErr.Code)
}
