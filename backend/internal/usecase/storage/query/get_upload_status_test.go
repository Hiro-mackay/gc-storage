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
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type getUploadStatusTestDeps struct {
	uploadSessionRepo *mocks.MockUploadSessionRepository
}

func newGetUploadStatusTestDeps(t *testing.T) *getUploadStatusTestDeps {
	t.Helper()
	return &getUploadStatusTestDeps{
		uploadSessionRepo: mocks.NewMockUploadSessionRepository(t),
	}
}

func (d *getUploadStatusTestDeps) newQuery() *query.GetUploadStatusQuery {
	return query.NewGetUploadStatusQuery(d.uploadSessionRepo)
}

func newPendingUploadSession(ownerID, folderID uuid.UUID) *entity.UploadSession {
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

func TestGetUploadStatusQuery_Execute_ValidOwner_ReturnsOutput(t *testing.T) {
	ctx := context.Background()
	deps := newGetUploadStatusTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()
	session := newPendingUploadSession(ownerID, folderID)

	input := query.GetUploadStatusInput{
		SessionID: session.ID,
		UserID:    ownerID,
	}

	deps.uploadSessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, session.ID, output.SessionID)
	assert.Equal(t, session.FileID, output.FileID)
	assert.Equal(t, session.Status, output.Status)
	assert.Equal(t, session.IsMultipart, output.IsMultipart)
	assert.Equal(t, session.TotalParts, output.TotalParts)
	assert.Equal(t, session.UploadedParts, output.UploadedParts)
	assert.False(t, output.IsExpired)
}

func TestGetUploadStatusQuery_Execute_ExpiredSession_ReturnsIsExpiredTrue(t *testing.T) {
	ctx := context.Background()
	deps := newGetUploadStatusTestDeps(t)

	ownerID := uuid.New()
	folderID := uuid.New()
	fileID := uuid.New()
	fileName, _ := valueobject.NewFileName("file.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(fileID)

	expiredSession := entity.ReconstructUploadSession(
		uuid.New(), fileID, ownerID, ownerID, folderID,
		fileName, mimeType, 1024, storageKey,
		nil, false, 1, 0,
		entity.UploadSessionStatusPending,
		time.Now().Add(-48*time.Hour), time.Now().Add(-48*time.Hour), time.Now().Add(-24*time.Hour), // expiresAt in past
	)

	input := query.GetUploadStatusInput{
		SessionID: expiredSession.ID,
		UserID:    ownerID,
	}

	deps.uploadSessionRepo.On("FindByID", ctx, expiredSession.ID).Return(expiredSession, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.True(t, output.IsExpired)
}

func TestGetUploadStatusQuery_Execute_NotOwner_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newGetUploadStatusTestDeps(t)

	ownerID := uuid.New()
	differentUserID := uuid.New()
	folderID := uuid.New()
	session := newPendingUploadSession(ownerID, folderID)

	input := query.GetUploadStatusInput{
		SessionID: session.ID,
		UserID:    differentUserID,
	}

	deps.uploadSessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestGetUploadStatusQuery_Execute_SessionNotFound_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newGetUploadStatusTestDeps(t)

	sessionID := uuid.New()
	notFoundErr := apperror.NewNotFoundError("upload session")

	input := query.GetUploadStatusInput{
		SessionID: sessionID,
		UserID:    uuid.New(),
	}

	deps.uploadSessionRepo.On("FindByID", ctx, sessionID).Return(nil, notFoundErr)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}
