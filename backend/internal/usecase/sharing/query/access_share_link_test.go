package query_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/sharing/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type accessShareLinkTestDeps struct {
	shareLinkRepo       *mocks.MockShareLinkRepository
	shareLinkAccessRepo *mocks.MockShareLinkAccessRepository
	fileRepo            *mocks.MockFileRepository
	fileVersionRepo     *mocks.MockFileVersionRepository
	folderRepo          *mocks.MockFolderRepository
	storageService      *mocks.MockStorageService
}

func newAccessShareLinkTestDeps(t *testing.T) *accessShareLinkTestDeps {
	t.Helper()
	return &accessShareLinkTestDeps{
		shareLinkRepo:       mocks.NewMockShareLinkRepository(t),
		shareLinkAccessRepo: mocks.NewMockShareLinkAccessRepository(t),
		fileRepo:            mocks.NewMockFileRepository(t),
		fileVersionRepo:     mocks.NewMockFileVersionRepository(t),
		folderRepo:          mocks.NewMockFolderRepository(t),
		storageService:      mocks.NewMockStorageService(t),
	}
}

func (d *accessShareLinkTestDeps) newQuery() *query.AccessShareLinkQuery {
	return query.NewAccessShareLinkQuery(
		d.shareLinkRepo,
		d.shareLinkAccessRepo,
		d.fileRepo,
		d.fileVersionRepo,
		d.folderRepo,
		d.storageService,
	)
}

func buildActiveShareLinkWithToken() (*entity.ShareLink, valueobject.ShareToken) {
	token, _ := valueobject.NewShareToken()
	perm, _ := valueobject.NewSharePermission("read")
	now := time.Now()
	sl := &entity.ShareLink{
		ID:           uuid.New(),
		Token:        token,
		ResourceType: authz.ResourceTypeFile,
		ResourceID:   uuid.New(),
		CreatedBy:    uuid.New(),
		Permission:   perm,
		Status:       valueobject.ShareLinkStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	return sl, token
}

func TestAccessShareLinkQuery_Execute_ViewAction_FileShare_ReturnsFileInfoWithoutPresignedURL(t *testing.T) {
	ctx := context.Background()
	deps := newAccessShareLinkTestDeps(t)
	shareLink, token := buildActiveShareLinkWithToken()

	fileID := shareLink.ResourceID
	file := buildActiveFile(fileID)

	input := query.AccessShareLinkInput{
		Token:     token.String(),
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
		Action:    "view",
	}

	deps.shareLinkRepo.On("FindByToken", ctx, token).Return(shareLink, nil)
	deps.fileRepo.On("FindByID", ctx, fileID).Return(file, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, shareLink, output.ShareLink)
	assert.Equal(t, file.Name.String(), output.ResourceName)
	assert.Nil(t, output.PresignedURL, "view action should NOT generate a presigned URL")
}

func TestAccessShareLinkQuery_Execute_DownloadAction_FileShare_ReturnsPresignedURL(t *testing.T) {
	ctx := context.Background()
	deps := newAccessShareLinkTestDeps(t)
	shareLink, token := buildActiveShareLinkWithToken()

	fileID := shareLink.ResourceID
	file := buildActiveFile(fileID)
	fileVersion := buildFileVersion(fileID)
	presignedURL := &service.PresignedURL{
		URL:       "https://example.com/download",
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	input := query.AccessShareLinkInput{
		Token:     token.String(),
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
		Action:    "download",
	}

	deps.shareLinkRepo.On("FindByToken", ctx, token).Return(shareLink, nil)
	deps.fileRepo.On("FindByID", ctx, fileID).Return(file, nil)
	deps.fileVersionRepo.On("FindLatestByFileID", ctx, fileID).Return(fileVersion, nil)
	deps.storageService.On("GenerateGetURL", ctx, file.StorageKey.String(), query.DownloadPresignedURLExpiry).Return(presignedURL, nil)
	deps.shareLinkRepo.On("Update", ctx, shareLink).Return(nil)
	deps.shareLinkAccessRepo.On("Create", ctx, mock.AnythingOfType("*entity.ShareLinkAccess")).Return(nil).Maybe()

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	require.NotNil(t, output.PresignedURL, "download action should generate a presigned URL")
	assert.Equal(t, presignedURL.URL, *output.PresignedURL)
}

func TestAccessShareLinkQuery_Execute_ViewAction_FolderShare_ReturnsFolderInfo(t *testing.T) {
	ctx := context.Background()
	deps := newAccessShareLinkTestDeps(t)

	token, _ := valueobject.NewShareToken()
	perm, _ := valueobject.NewSharePermission("read")
	folderID := uuid.New()
	now := time.Now()
	shareLink := &entity.ShareLink{
		ID:           uuid.New(),
		Token:        token,
		ResourceType: authz.ResourceTypeFolder,
		ResourceID:   folderID,
		CreatedBy:    uuid.New(),
		Permission:   perm,
		Status:       valueobject.ShareLinkStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	folderName, _ := valueobject.NewFolderName("My Folder")
	folder := entity.ReconstructFolder(
		folderID,
		folderName,
		nil,
		uuid.New(),
		uuid.New(),
		0,
		entity.FolderStatusActive,
		time.Now(),
		time.Now(),
	)

	input := query.AccessShareLinkInput{
		Token:     token.String(),
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
		Action:    "view",
	}

	deps.shareLinkRepo.On("FindByToken", ctx, token).Return(shareLink, nil)
	deps.folderRepo.On("FindByID", ctx, folderID).Return(folder, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, folder.Name.String(), output.ResourceName)
	assert.Nil(t, output.PresignedURL, "view action should NOT generate a presigned URL")
	assert.Nil(t, output.Contents, "view action should NOT list folder contents")
}

func TestAccessShareLinkQuery_Execute_DownloadAction_IncreasesAccessCount(t *testing.T) {
	ctx := context.Background()
	deps := newAccessShareLinkTestDeps(t)
	shareLink, token := buildActiveShareLinkWithToken()

	fileID := shareLink.ResourceID
	file := buildActiveFile(fileID)
	fileVersion := buildFileVersion(fileID)
	presignedURL := &service.PresignedURL{
		URL:       "https://example.com/download",
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	input := query.AccessShareLinkInput{
		Token:     token.String(),
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
		Action:    "download",
	}

	deps.shareLinkRepo.On("FindByToken", ctx, token).Return(shareLink, nil)
	deps.fileRepo.On("FindByID", ctx, fileID).Return(file, nil)
	deps.fileVersionRepo.On("FindLatestByFileID", ctx, fileID).Return(fileVersion, nil)
	deps.storageService.On("GenerateGetURL", ctx, file.StorageKey.String(), query.DownloadPresignedURLExpiry).Return(presignedURL, nil)
	deps.shareLinkRepo.On("Update", ctx, shareLink).Return(nil)
	deps.shareLinkAccessRepo.On("Create", ctx, mock.AnythingOfType("*entity.ShareLinkAccess")).Return(nil).Maybe()

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, 1, output.ShareLink.AccessCount)
}

func TestAccessShareLinkQuery_Execute_ExpiredShareLink_ReturnsGoneError(t *testing.T) {
	ctx := context.Background()
	deps := newAccessShareLinkTestDeps(t)

	token, _ := valueobject.NewShareToken()
	perm, _ := valueobject.NewSharePermission("read")
	past := time.Now().Add(-1 * time.Hour)
	now := time.Now()
	shareLink := &entity.ShareLink{
		ID:           uuid.New(),
		Token:        token,
		ResourceType: authz.ResourceTypeFile,
		ResourceID:   uuid.New(),
		CreatedBy:    uuid.New(),
		Permission:   perm,
		ExpiresAt:    &past,
		Status:       valueobject.ShareLinkStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	input := query.AccessShareLinkInput{
		Token:     token.String(),
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
		Action:    "view",
	}

	deps.shareLinkRepo.On("FindByToken", ctx, token).Return(shareLink, nil)

	q := deps.newQuery()
	_, err := q.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeGone, appErr.Code)
}

func TestAccessShareLinkQuery_Execute_InvalidToken_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newAccessShareLinkTestDeps(t)

	input := query.AccessShareLinkInput{
		Token:     "invalid-token",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
		Action:    "view",
	}

	q := deps.newQuery()
	_, err := q.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestAccessShareLinkQuery_Execute_ViewAction_FileShare_ResourceDeleted_ReturnsNotFound(t *testing.T) {
	ctx := context.Background()
	deps := newAccessShareLinkTestDeps(t)
	shareLink, token := buildActiveShareLinkWithToken()

	fileID := shareLink.ResourceID

	input := query.AccessShareLinkInput{
		Token:     token.String(),
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
		Action:    "view",
	}

	deps.shareLinkRepo.On("FindByToken", ctx, token).Return(shareLink, nil)
	deps.fileRepo.On("FindByID", ctx, fileID).Return(nil, apperror.NewNotFoundError("file"))

	q := deps.newQuery()
	_, err := q.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestAccessShareLinkQuery_Execute_ViewAction_FolderShare_FolderDeleted_ReturnsNotFound(t *testing.T) {
	ctx := context.Background()
	deps := newAccessShareLinkTestDeps(t)

	token, _ := valueobject.NewShareToken()
	perm, _ := valueobject.NewSharePermission("read")
	folderID := uuid.New()
	now := time.Now()
	shareLink := &entity.ShareLink{
		ID:           uuid.New(),
		Token:        token,
		ResourceType: authz.ResourceTypeFolder,
		ResourceID:   folderID,
		CreatedBy:    uuid.New(),
		Permission:   perm,
		Status:       valueobject.ShareLinkStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	input := query.AccessShareLinkInput{
		Token:     token.String(),
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
		Action:    "view",
	}

	deps.shareLinkRepo.On("FindByToken", ctx, token).Return(shareLink, nil)
	deps.folderRepo.On("FindByID", ctx, folderID).Return(nil, apperror.NewNotFoundError("folder"))

	q := deps.newQuery()
	_, err := q.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestAccessShareLinkQuery_Execute_RevokedShareLink_ReturnsGoneError(t *testing.T) {
	ctx := context.Background()
	deps := newAccessShareLinkTestDeps(t)

	token, _ := valueobject.NewShareToken()
	perm, _ := valueobject.NewSharePermission("read")
	now := time.Now()
	shareLink := &entity.ShareLink{
		ID:           uuid.New(),
		Token:        token,
		ResourceType: authz.ResourceTypeFile,
		ResourceID:   uuid.New(),
		CreatedBy:    uuid.New(),
		Permission:   perm,
		Status:       valueobject.ShareLinkStatusRevoked,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	input := query.AccessShareLinkInput{
		Token:     token.String(),
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
		Action:    "view",
	}

	deps.shareLinkRepo.On("FindByToken", ctx, token).Return(shareLink, nil)

	q := deps.newQuery()
	_, err := q.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeGone, appErr.Code)
}
