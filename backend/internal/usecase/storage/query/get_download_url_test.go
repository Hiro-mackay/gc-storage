package query_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

// intPtr returns a pointer to the given int value.
func intPtr(n int) *int { return &n }

type getDownloadURLTestDeps struct {
	fileRepo        *mocks.MockFileRepository
	fileVersionRepo *mocks.MockFileVersionRepository
	storageService  *mocks.MockStorageService
}

func newGetDownloadURLTestDeps(t *testing.T) *getDownloadURLTestDeps {
	t.Helper()
	return &getDownloadURLTestDeps{
		fileRepo:        mocks.NewMockFileRepository(t),
		fileVersionRepo: mocks.NewMockFileVersionRepository(t),
		storageService:  mocks.NewMockStorageService(t),
	}
}

func (d *getDownloadURLTestDeps) newQuery() *query.GetDownloadURLQuery {
	return query.NewGetDownloadURLQuery(d.fileRepo, d.fileVersionRepo, d.storageService)
}

func newActiveFileForQuery(ownerID, folderID uuid.UUID) *entity.File {
	name, _ := valueobject.NewFileName("report.pdf")
	mimeType, _ := valueobject.NewMimeType("application/pdf")
	fileID := uuid.New()
	storageKey := valueobject.NewStorageKey(fileID)
	return entity.ReconstructFile(
		fileID, folderID, ownerID, ownerID,
		name, mimeType, 10485760, storageKey, 2,
		entity.FileStatusActive, time.Now(), time.Now(),
	)
}

func newUploadingFileForQuery(ownerID, folderID uuid.UUID) *entity.File {
	name, _ := valueobject.NewFileName("report.pdf")
	mimeType, _ := valueobject.NewMimeType("application/pdf")
	fileID := uuid.New()
	storageKey := valueobject.NewStorageKey(fileID)
	return entity.ReconstructFile(
		fileID, folderID, ownerID, ownerID,
		name, mimeType, 10485760, storageKey, 1,
		entity.FileStatusUploading, time.Now(), time.Now(),
	)
}

func newFileVersionForQuery(fileID uuid.UUID, versionNumber int) *entity.FileVersion {
	return entity.ReconstructFileVersion(
		uuid.New(), fileID, versionNumber, "minio-version-id",
		10485760, "sha256:abc123", uuid.New(), time.Now(),
	)
}

func TestGetDownloadURLQuery_Execute_ActiveFile_ReturnsPresignedURL(t *testing.T) {
	ctx := context.Background()
	ownerID := uuid.New()
	folderID := uuid.New()

	deps := newGetDownloadURLTestDeps(t)
	file := newActiveFileForQuery(ownerID, folderID)
	version := newFileVersionForQuery(file.ID, 2)
	presigned := &service.PresignedURL{
		URL:       "https://minio.example.com/presigned-url",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)
	deps.fileVersionRepo.On("FindLatestByFileID", ctx, file.ID).Return(version, nil)
	deps.storageService.On("GenerateGetURL", ctx, file.StorageKey.String(), query.DownloadURLExpiry).Return(presigned, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.GetDownloadURLInput{
		FileID: file.ID,
		UserID: ownerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, file.ID, output.FileID)
	assert.Equal(t, "report.pdf", output.FileName)
	assert.Equal(t, "application/pdf", output.MimeType)
	assert.Equal(t, version.Size, output.Size)
	assert.Equal(t, version.VersionNumber, output.VersionNumber)
	assert.Equal(t, presigned.URL, output.DownloadURL)
	assert.Equal(t, presigned.ExpiresAt, output.ExpiresAt)
}

func TestGetDownloadURLQuery_Execute_WithVersionNumber_ReturnsVersionedURL(t *testing.T) {
	ctx := context.Background()
	ownerID := uuid.New()
	folderID := uuid.New()

	deps := newGetDownloadURLTestDeps(t)
	file := newActiveFileForQuery(ownerID, folderID)
	version := newFileVersionForQuery(file.ID, 1)
	presigned := &service.PresignedURL{
		URL:       "https://minio.example.com/presigned-url-v1",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)
	deps.fileVersionRepo.On("FindByFileAndVersion", ctx, file.ID, 1).Return(version, nil)
	deps.storageService.On("GenerateGetURL", ctx, file.StorageKey.String(), query.DownloadURLExpiry).Return(presigned, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.GetDownloadURLInput{
		FileID:        file.ID,
		VersionNumber: intPtr(1),
		UserID:        ownerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 1, output.VersionNumber)
	assert.Equal(t, presigned.URL, output.DownloadURL)
}

func TestGetDownloadURLQuery_Execute_FileNotFound_PropagatesError(t *testing.T) {
	ctx := context.Background()
	fileID := uuid.New()
	userID := uuid.New()

	deps := newGetDownloadURLTestDeps(t)
	notFoundErr := apperror.NewNotFoundError("file")

	deps.fileRepo.On("FindByID", ctx, fileID).Return(nil, notFoundErr)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.GetDownloadURLInput{
		FileID: fileID,
		UserID: userID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestGetDownloadURLQuery_Execute_NotOwner_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	ownerID := uuid.New()
	otherUserID := uuid.New()
	folderID := uuid.New()

	deps := newGetDownloadURLTestDeps(t)
	file := newActiveFileForQuery(ownerID, folderID)

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.GetDownloadURLInput{
		FileID: file.ID,
		UserID: otherUserID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestGetDownloadURLQuery_Execute_UploadingFile_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	ownerID := uuid.New()
	folderID := uuid.New()

	deps := newGetDownloadURLTestDeps(t)
	file := newUploadingFileForQuery(ownerID, folderID)

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.GetDownloadURLInput{
		FileID: file.ID,
		UserID: ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestGetDownloadURLQuery_Execute_VersionNotFound_PropagatesError(t *testing.T) {
	ctx := context.Background()
	ownerID := uuid.New()
	folderID := uuid.New()

	deps := newGetDownloadURLTestDeps(t)
	file := newActiveFileForQuery(ownerID, folderID)
	notFoundErr := apperror.NewNotFoundError("file version")

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)
	deps.fileVersionRepo.On("FindByFileAndVersion", ctx, file.ID, 99).Return(nil, notFoundErr)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.GetDownloadURLInput{
		FileID:        file.ID,
		VersionNumber: intPtr(99),
		UserID:        ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestGetDownloadURLQuery_Execute_StorageServiceError_ReturnsInternalError(t *testing.T) {
	ctx := context.Background()
	ownerID := uuid.New()
	folderID := uuid.New()

	deps := newGetDownloadURLTestDeps(t)
	file := newActiveFileForQuery(ownerID, folderID)
	version := newFileVersionForQuery(file.ID, 2)
	storageErr := errors.New("storage connection failed")

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)
	deps.fileVersionRepo.On("FindLatestByFileID", ctx, file.ID).Return(version, nil)
	deps.storageService.On("GenerateGetURL", ctx, file.StorageKey.String(), query.DownloadURLExpiry).Return(nil, storageErr)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.GetDownloadURLInput{
		FileID: file.ID,
		UserID: ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeInternalError, appErr.Code)
}
