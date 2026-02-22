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

type getDownloadViaShareTestDeps struct {
	shareLinkRepo       *mocks.MockShareLinkRepository
	shareLinkAccessRepo *mocks.MockShareLinkAccessRepository
	fileRepo            *mocks.MockFileRepository
	fileVersionRepo     *mocks.MockFileVersionRepository
	folderClosureRepo   *mocks.MockFolderClosureRepository
	storageService      *mocks.MockStorageService
}

func newGetDownloadViaShareTestDeps(t *testing.T) *getDownloadViaShareTestDeps {
	t.Helper()
	return &getDownloadViaShareTestDeps{
		shareLinkRepo:       mocks.NewMockShareLinkRepository(t),
		shareLinkAccessRepo: mocks.NewMockShareLinkAccessRepository(t),
		fileRepo:            mocks.NewMockFileRepository(t),
		fileVersionRepo:     mocks.NewMockFileVersionRepository(t),
		folderClosureRepo:   mocks.NewMockFolderClosureRepository(t),
		storageService:      mocks.NewMockStorageService(t),
	}
}

func (d *getDownloadViaShareTestDeps) newQuery() *query.GetDownloadViaShareQuery {
	return query.NewGetDownloadViaShareQuery(
		d.shareLinkRepo,
		d.shareLinkAccessRepo,
		d.fileRepo,
		d.fileVersionRepo,
		d.folderClosureRepo,
		d.storageService,
	)
}

func buildFileShareLink(resourceID uuid.UUID) *entity.ShareLink {
	token, _ := valueobject.NewShareToken()
	perm, _ := valueobject.NewSharePermission("read")
	now := time.Now()
	return &entity.ShareLink{
		ID:           uuid.New(),
		Token:        token,
		ResourceType: authz.ResourceTypeFile,
		ResourceID:   resourceID,
		CreatedBy:    uuid.New(),
		Permission:   perm,
		Status:       valueobject.ShareLinkStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func buildActiveFile(fileID uuid.UUID) *entity.File {
	name, _ := valueobject.NewFileName("test.txt")
	mime, _ := valueobject.NewMimeType("text/plain")
	return entity.ReconstructFile(
		fileID,
		uuid.New(),
		uuid.New(),
		uuid.New(),
		name,
		mime,
		1024,
		valueobject.NewStorageKey(fileID),
		1,
		entity.FileStatusActive,
		time.Now(),
		time.Now(),
	)
}

func buildFileVersion(fileID uuid.UUID) *entity.FileVersion {
	return entity.ReconstructFileVersion(
		uuid.New(),
		fileID,
		1,
		"minio-version-id",
		1024,
		"checksum",
		uuid.New(),
		time.Now(),
	)
}

func TestGetDownloadViaShareQuery_Execute_FileShareLink_ReturnsPresignedURL(t *testing.T) {
	ctx := context.Background()
	deps := newGetDownloadViaShareTestDeps(t)

	fileID := uuid.New()
	shareLink := buildFileShareLink(fileID)
	token := shareLink.Token
	file := buildActiveFile(fileID)
	fileVersion := buildFileVersion(fileID)
	presignedURL := &service.PresignedURL{
		URL:       "https://example.com/download",
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	input := query.GetDownloadViaShareInput{
		Token:     token.String(),
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
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
	assert.Equal(t, presignedURL.URL, output.PresignedURL)
	assert.Equal(t, file.Name.String(), output.FileName)
}

func TestGetDownloadViaShareQuery_Execute_ExpiredShareLink_ReturnsGoneError(t *testing.T) {
	ctx := context.Background()
	deps := newGetDownloadViaShareTestDeps(t)

	fileID := uuid.New()
	shareLink := buildFileShareLink(fileID)
	past := time.Now().Add(-1 * time.Hour)
	shareLink.ExpiresAt = &past
	token := shareLink.Token

	input := query.GetDownloadViaShareInput{
		Token:     token.String(),
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	deps.shareLinkRepo.On("FindByToken", ctx, token).Return(shareLink, nil)

	q := deps.newQuery()
	_, err := q.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeGone, appErr.Code)
}

func TestGetDownloadViaShareQuery_Execute_RevokedShareLink_ReturnsGoneError(t *testing.T) {
	ctx := context.Background()
	deps := newGetDownloadViaShareTestDeps(t)

	fileID := uuid.New()
	shareLink := buildFileShareLink(fileID)
	shareLink.Revoke()
	token := shareLink.Token

	input := query.GetDownloadViaShareInput{
		Token:     token.String(),
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	deps.shareLinkRepo.On("FindByToken", ctx, token).Return(shareLink, nil)

	q := deps.newQuery()
	_, err := q.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeGone, appErr.Code)
}

func TestGetDownloadViaShareQuery_Execute_PasswordProtected_MissingPassword_ReturnsUnauthorized(t *testing.T) {
	ctx := context.Background()
	deps := newGetDownloadViaShareTestDeps(t)

	fileID := uuid.New()
	token, _ := valueobject.NewShareToken()
	perm, _ := valueobject.NewSharePermission("read")
	now := time.Now()
	shareLink := &entity.ShareLink{
		ID:           uuid.New(),
		Token:        token,
		ResourceType: authz.ResourceTypeFile,
		ResourceID:   fileID,
		CreatedBy:    uuid.New(),
		Permission:   perm,
		PasswordHash: "$2a$12$somehash",
		Status:       valueobject.ShareLinkStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	input := query.GetDownloadViaShareInput{
		Token:     token.String(),
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	deps.shareLinkRepo.On("FindByToken", ctx, token).Return(shareLink, nil)

	q := deps.newQuery()
	_, err := q.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeUnauthorized, appErr.Code)
}

func buildFolderShareLink(folderID uuid.UUID) *entity.ShareLink {
	token, _ := valueobject.NewShareToken()
	perm, _ := valueobject.NewSharePermission("read")
	now := time.Now()
	return &entity.ShareLink{
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
}

func buildActiveFileInFolder(fileID, folderID uuid.UUID) *entity.File {
	name, _ := valueobject.NewFileName("doc.pdf")
	mime, _ := valueobject.NewMimeType("application/pdf")
	return entity.ReconstructFile(
		fileID,
		folderID,
		uuid.New(),
		uuid.New(),
		name,
		mime,
		2048,
		valueobject.NewStorageKey(fileID),
		1,
		entity.FileStatusActive,
		time.Now(),
		time.Now(),
	)
}

func TestGetDownloadViaShareQuery_Execute_FolderShareWithFileID_ReturnsPresignedURL(t *testing.T) {
	ctx := context.Background()
	deps := newGetDownloadViaShareTestDeps(t)

	folderID := uuid.New()
	fileID := uuid.New()
	shareLink := buildFolderShareLink(folderID)
	token := shareLink.Token
	file := buildActiveFileInFolder(fileID, folderID)
	fileVersion := buildFileVersion(fileID)
	presignedURL := &service.PresignedURL{
		URL:       "https://example.com/download/doc.pdf",
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	input := query.GetDownloadViaShareInput{
		Token:     token.String(),
		FileID:    &fileID,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	// folderID is a descendant of itself (closure table includes self)
	deps.shareLinkRepo.On("FindByToken", ctx, token).Return(shareLink, nil)
	deps.folderClosureRepo.On("FindDescendantIDs", ctx, folderID).Return([]uuid.UUID{folderID}, nil)
	deps.fileRepo.On("FindByID", ctx, fileID).Return(file, nil)
	deps.fileVersionRepo.On("FindLatestByFileID", ctx, fileID).Return(fileVersion, nil)
	deps.storageService.On("GenerateGetURL", ctx, file.StorageKey.String(), query.DownloadPresignedURLExpiry).Return(presignedURL, nil)
	deps.shareLinkRepo.On("Update", ctx, shareLink).Return(nil)
	deps.shareLinkAccessRepo.On("Create", ctx, mock.AnythingOfType("*entity.ShareLinkAccess")).Return(nil).Maybe()

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, presignedURL.URL, output.PresignedURL)
	assert.Equal(t, file.Name.String(), output.FileName)
}

func TestGetDownloadViaShareQuery_Execute_FolderShareFileNotInSubtree_ReturnsNotFound(t *testing.T) {
	ctx := context.Background()
	deps := newGetDownloadViaShareTestDeps(t)

	folderID := uuid.New()
	fileID := uuid.New()
	otherFolderID := uuid.New() // file belongs to a different folder
	shareLink := buildFolderShareLink(folderID)
	token := shareLink.Token
	file := buildActiveFileInFolder(fileID, otherFolderID)

	input := query.GetDownloadViaShareInput{
		Token:     token.String(),
		FileID:    &fileID,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	deps.shareLinkRepo.On("FindByToken", ctx, token).Return(shareLink, nil)
	deps.folderClosureRepo.On("FindDescendantIDs", ctx, folderID).Return([]uuid.UUID{folderID}, nil)
	deps.fileRepo.On("FindByID", ctx, fileID).Return(file, nil)

	q := deps.newQuery()
	_, err := q.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestGetDownloadViaShareQuery_Execute_FolderShareWithoutFileID_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newGetDownloadViaShareTestDeps(t)

	folderID := uuid.New()
	shareLink := buildFolderShareLink(folderID)
	token := shareLink.Token

	input := query.GetDownloadViaShareInput{
		Token:     token.String(),
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
		// FileID is nil
	}

	deps.shareLinkRepo.On("FindByToken", ctx, token).Return(shareLink, nil)

	q := deps.newQuery()
	_, err := q.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestGetDownloadViaShareQuery_Execute_FileNotActive_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newGetDownloadViaShareTestDeps(t)

	fileID := uuid.New()
	shareLink := buildFileShareLink(fileID)
	token := shareLink.Token
	name, _ := valueobject.NewFileName("test.txt")
	mime, _ := valueobject.NewMimeType("text/plain")
	file := entity.ReconstructFile(
		fileID,
		uuid.New(),
		uuid.New(),
		uuid.New(),
		name,
		mime,
		1024,
		valueobject.NewStorageKey(fileID),
		1,
		entity.FileStatusUploading,
		time.Now(),
		time.Now(),
	)

	input := query.GetDownloadViaShareInput{
		Token:     token.String(),
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	deps.shareLinkRepo.On("FindByToken", ctx, token).Return(shareLink, nil)
	deps.fileRepo.On("FindByID", ctx, fileID).Return(file, nil)

	q := deps.newQuery()
	_, err := q.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}
