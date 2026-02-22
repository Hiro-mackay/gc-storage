package query_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/sharing/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type getShareLinkHistoryTestDeps struct {
	shareLinkRepo       *mocks.MockShareLinkRepository
	shareLinkAccessRepo *mocks.MockShareLinkAccessRepository
	permissionResolver  *mocks.MockPermissionResolver
}

func newGetShareLinkHistoryTestDeps(t *testing.T) *getShareLinkHistoryTestDeps {
	t.Helper()
	return &getShareLinkHistoryTestDeps{
		shareLinkRepo:       mocks.NewMockShareLinkRepository(t),
		shareLinkAccessRepo: mocks.NewMockShareLinkAccessRepository(t),
		permissionResolver:  mocks.NewMockPermissionResolver(t),
	}
}

func (d *getShareLinkHistoryTestDeps) newQuery() *query.GetShareLinkHistoryQuery {
	return query.NewGetShareLinkHistoryQuery(d.shareLinkRepo, d.shareLinkAccessRepo, d.permissionResolver)
}

func buildTestShareLink(createdBy uuid.UUID, resourceType authz.ResourceType) *entity.ShareLink {
	token, _ := valueobject.NewShareToken()
	perm, _ := valueobject.NewSharePermission("read")
	now := time.Now()
	return &entity.ShareLink{
		ID:           uuid.New(),
		Token:        token,
		ResourceType: resourceType,
		ResourceID:   uuid.New(),
		CreatedBy:    createdBy,
		Permission:   perm,
		Status:       valueobject.ShareLinkStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func buildTestShareLinkAccess(shareLinkID uuid.UUID) *entity.ShareLinkAccess {
	return &entity.ShareLinkAccess{
		ID:          uuid.New(),
		ShareLinkID: shareLinkID,
		AccessedAt:  time.Now(),
		IPAddress:   "127.0.0.1",
		UserAgent:   "test-agent",
		Action:      entity.AccessActionView,
	}
}

func TestGetShareLinkHistoryQuery_Execute_ValidInput_ReturnsHistory(t *testing.T) {
	ctx := context.Background()
	deps := newGetShareLinkHistoryTestDeps(t)
	userID := uuid.New()
	shareLink := buildTestShareLink(userID, authz.ResourceTypeFile)
	accesses := []*entity.ShareLinkAccess{
		buildTestShareLinkAccess(shareLink.ID),
		buildTestShareLinkAccess(shareLink.ID),
	}

	input := query.GetShareLinkHistoryInput{
		ShareLinkID: shareLink.ID,
		UserID:      userID,
		Limit:       10,
		Offset:      0,
	}

	deps.shareLinkRepo.On("FindByID", ctx, shareLink.ID).Return(shareLink, nil)
	deps.shareLinkAccessRepo.On("FindByShareLinkIDWithPagination", ctx, shareLink.ID, 10, 0).Return(accesses, nil)
	deps.shareLinkAccessRepo.On("CountByShareLinkID", ctx, shareLink.ID).Return(2, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Len(t, output.Accesses, 2)
	assert.Equal(t, 2, output.Total)
}

func TestGetShareLinkHistoryQuery_Execute_NonCreatorWithPermission_ReturnsHistory(t *testing.T) {
	ctx := context.Background()
	deps := newGetShareLinkHistoryTestDeps(t)
	creatorID := uuid.New()
	otherUserID := uuid.New()
	shareLink := buildTestShareLink(creatorID, authz.ResourceTypeFile)

	input := query.GetShareLinkHistoryInput{
		ShareLinkID: shareLink.ID,
		UserID:      otherUserID,
		Limit:       10,
		Offset:      0,
	}

	deps.shareLinkRepo.On("FindByID", ctx, shareLink.ID).Return(shareLink, nil)
	deps.permissionResolver.On("HasPermission", ctx, otherUserID, authz.ResourceTypeFile, shareLink.ResourceID, authz.PermFileShare).Return(true, nil)
	deps.shareLinkAccessRepo.On("FindByShareLinkIDWithPagination", ctx, shareLink.ID, 10, 0).Return([]*entity.ShareLinkAccess{}, nil)
	deps.shareLinkAccessRepo.On("CountByShareLinkID", ctx, shareLink.ID).Return(0, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, 0, output.Total)
}

func TestGetShareLinkHistoryQuery_Execute_NonCreatorWithoutPermission_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newGetShareLinkHistoryTestDeps(t)
	creatorID := uuid.New()
	otherUserID := uuid.New()
	shareLink := buildTestShareLink(creatorID, authz.ResourceTypeFile)

	input := query.GetShareLinkHistoryInput{
		ShareLinkID: shareLink.ID,
		UserID:      otherUserID,
		Limit:       10,
		Offset:      0,
	}

	deps.shareLinkRepo.On("FindByID", ctx, shareLink.ID).Return(shareLink, nil)
	deps.permissionResolver.On("HasPermission", ctx, otherUserID, authz.ResourceTypeFile, shareLink.ResourceID, authz.PermFileShare).Return(false, nil)

	q := deps.newQuery()
	_, err := q.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestGetShareLinkHistoryQuery_Execute_ShareLinkNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	deps := newGetShareLinkHistoryTestDeps(t)

	input := query.GetShareLinkHistoryInput{
		ShareLinkID: uuid.New(),
		UserID:      uuid.New(),
		Limit:       10,
		Offset:      0,
	}

	deps.shareLinkRepo.On("FindByID", ctx, input.ShareLinkID).Return(nil, apperror.NewNotFoundError("share link"))

	q := deps.newQuery()
	_, err := q.Execute(ctx, input)

	require.Error(t, err)
}

func TestGetShareLinkHistoryQuery_Execute_DefaultPagination_UsesDefaults(t *testing.T) {
	ctx := context.Background()
	deps := newGetShareLinkHistoryTestDeps(t)
	userID := uuid.New()
	shareLink := buildTestShareLink(userID, authz.ResourceTypeFolder)

	input := query.GetShareLinkHistoryInput{
		ShareLinkID: shareLink.ID,
		UserID:      userID,
	}

	deps.shareLinkRepo.On("FindByID", ctx, shareLink.ID).Return(shareLink, nil)
	deps.shareLinkAccessRepo.On("FindByShareLinkIDWithPagination", ctx, shareLink.ID, 20, 0).Return([]*entity.ShareLinkAccess{}, nil)
	deps.shareLinkAccessRepo.On("CountByShareLinkID", ctx, shareLink.ID).Return(0, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
}

func TestGetShareLinkHistoryQuery_Execute_LimitExceedsMax_CapsAtMax(t *testing.T) {
	ctx := context.Background()
	deps := newGetShareLinkHistoryTestDeps(t)
	userID := uuid.New()
	shareLink := buildTestShareLink(userID, authz.ResourceTypeFile)

	input := query.GetShareLinkHistoryInput{
		ShareLinkID: shareLink.ID,
		UserID:      userID,
		Limit:       999, // exceeds max of 100
	}

	deps.shareLinkRepo.On("FindByID", ctx, shareLink.ID).Return(shareLink, nil)
	// should be capped at 100, not 999
	deps.shareLinkAccessRepo.On("FindByShareLinkIDWithPagination", ctx, shareLink.ID, 100, 0).Return([]*entity.ShareLinkAccess{}, nil)
	deps.shareLinkAccessRepo.On("CountByShareLinkID", ctx, shareLink.ID).Return(0, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
}
