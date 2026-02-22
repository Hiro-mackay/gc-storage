package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

func newSessionFileName() valueobject.FileName {
	name, _ := valueobject.NewFileName("test.txt")
	return name
}

func newSessionMimeType() valueobject.MimeType {
	mt, _ := valueobject.NewMimeType("text/plain")
	return mt
}

func newPendingSession() *UploadSession {
	fileID := uuid.New()
	createdBy := uuid.New()
	storageKey := valueobject.NewStorageKey(fileID)
	return ReconstructUploadSession(
		uuid.New(), fileID, createdBy, createdBy, uuid.New(),
		newSessionFileName(), newSessionMimeType(), 1024, storageKey,
		nil, false, 1, 0,
		UploadSessionStatusPending,
		time.Now(), time.Now(), time.Now().Add(UploadSessionTTL),
	)
}

func newInProgressSession() *UploadSession {
	fileID := uuid.New()
	createdBy := uuid.New()
	storageKey := valueobject.NewStorageKey(fileID)
	return ReconstructUploadSession(
		uuid.New(), fileID, createdBy, createdBy, uuid.New(),
		newSessionFileName(), newSessionMimeType(), 1024, storageKey,
		nil, false, 1, 0,
		UploadSessionStatusInProgress,
		time.Now(), time.Now(), time.Now().Add(UploadSessionTTL),
	)
}

func newCompletedSession() *UploadSession {
	fileID := uuid.New()
	createdBy := uuid.New()
	storageKey := valueobject.NewStorageKey(fileID)
	return ReconstructUploadSession(
		uuid.New(), fileID, createdBy, createdBy, uuid.New(),
		newSessionFileName(), newSessionMimeType(), 1024, storageKey,
		nil, false, 1, 1,
		UploadSessionStatusCompleted,
		time.Now(), time.Now(), time.Now().Add(UploadSessionTTL),
	)
}

func newAbortedSession() *UploadSession {
	fileID := uuid.New()
	createdBy := uuid.New()
	storageKey := valueobject.NewStorageKey(fileID)
	return ReconstructUploadSession(
		uuid.New(), fileID, createdBy, createdBy, uuid.New(),
		newSessionFileName(), newSessionMimeType(), 1024, storageKey,
		nil, false, 1, 0,
		UploadSessionStatusAborted,
		time.Now(), time.Now(), time.Now().Add(UploadSessionTTL),
	)
}

func newExpiredSession() *UploadSession {
	fileID := uuid.New()
	createdBy := uuid.New()
	storageKey := valueobject.NewStorageKey(fileID)
	return ReconstructUploadSession(
		uuid.New(), fileID, createdBy, createdBy, uuid.New(),
		newSessionFileName(), newSessionMimeType(), 1024, storageKey,
		nil, false, 1, 0,
		UploadSessionStatusPending,
		time.Now().Add(-2*UploadSessionTTL), time.Now().Add(-2*UploadSessionTTL), time.Now().Add(-UploadSessionTTL),
	)
}

// NewUploadSession tests

func TestNewUploadSession_SizeLessThan5MB_NotMultipart(t *testing.T) {
	fileID := uuid.New()
	createdBy := uuid.New()
	session := NewUploadSession(fileID, createdBy, uuid.New(), newSessionFileName(), newSessionMimeType(), 1024, nil)

	if session.IsMultipart {
		t.Error("session with size < 5MB should not be multipart")
	}
}

func TestNewUploadSession_SizeLessThan5MB_TotalPartsIsOne(t *testing.T) {
	fileID := uuid.New()
	session := NewUploadSession(fileID, uuid.New(), uuid.New(), newSessionFileName(), newSessionMimeType(), 1024, nil)

	if session.TotalParts != 1 {
		t.Errorf("expected TotalParts 1, got %d", session.TotalParts)
	}
}

func TestNewUploadSession_SizeEquals5MB_IsMultipart(t *testing.T) {
	fileID := uuid.New()
	session := NewUploadSession(fileID, uuid.New(), uuid.New(), newSessionFileName(), newSessionMimeType(), MultipartThreshold, nil)

	if !session.IsMultipart {
		t.Error("session with size == 5MB should be multipart")
	}
}

func TestNewUploadSession_SizeEquals5MB_TotalPartsIsOne(t *testing.T) {
	fileID := uuid.New()
	session := NewUploadSession(fileID, uuid.New(), uuid.New(), newSessionFileName(), newSessionMimeType(), MultipartThreshold, nil)

	if session.TotalParts != 1 {
		t.Errorf("expected TotalParts 1 for exactly 5MB, got %d", session.TotalParts)
	}
}

func TestNewUploadSession_SizeEquals10MB_TotalPartsIsTwo(t *testing.T) {
	fileID := uuid.New()
	session := NewUploadSession(fileID, uuid.New(), uuid.New(), newSessionFileName(), newSessionMimeType(), 10*1024*1024, nil)

	if session.TotalParts != 2 {
		t.Errorf("expected TotalParts 2 for 10MB, got %d", session.TotalParts)
	}
}

func TestNewUploadSession_OwnerIDEqualsCreatedBy(t *testing.T) {
	fileID := uuid.New()
	createdBy := uuid.New()
	session := NewUploadSession(fileID, createdBy, uuid.New(), newSessionFileName(), newSessionMimeType(), 1024, nil)

	if session.OwnerID != session.CreatedBy {
		t.Error("OwnerID should equal CreatedBy on new session")
	}
}

// Complete tests

func TestUploadSession_Complete_FromPending_ReturnsNil(t *testing.T) {
	session := newPendingSession()

	err := session.Complete()

	if err != nil {
		t.Errorf("Complete from pending should return nil, got: %v", err)
	}
}

func TestUploadSession_Complete_FromInProgress_ReturnsNil(t *testing.T) {
	session := newInProgressSession()

	err := session.Complete()

	if err != nil {
		t.Errorf("Complete from in_progress should return nil, got: %v", err)
	}
}

func TestUploadSession_Complete_AlreadyCompleted_ReturnsErrCompleted(t *testing.T) {
	session := newCompletedSession()

	err := session.Complete()

	if err != ErrUploadSessionCompleted {
		t.Errorf("expected ErrUploadSessionCompleted, got: %v", err)
	}
}

func TestUploadSession_Complete_AlreadyAborted_ReturnsErrAborted(t *testing.T) {
	session := newAbortedSession()

	err := session.Complete()

	if err != ErrUploadSessionAborted {
		t.Errorf("expected ErrUploadSessionAborted, got: %v", err)
	}
}

func TestUploadSession_Complete_Expired_ReturnsErrExpired(t *testing.T) {
	session := newExpiredSession()

	err := session.Complete()

	if err != ErrUploadSessionExpired {
		t.Errorf("expected ErrUploadSessionExpired, got: %v", err)
	}
}

// Abort tests

func TestUploadSession_Abort_FromPending_ReturnsNil(t *testing.T) {
	session := newPendingSession()

	err := session.Abort()

	if err != nil {
		t.Errorf("Abort from pending should return nil, got: %v", err)
	}
}

func TestUploadSession_Abort_FromInProgress_ReturnsNil(t *testing.T) {
	session := newInProgressSession()

	err := session.Abort()

	if err != nil {
		t.Errorf("Abort from in_progress should return nil, got: %v", err)
	}
}

func TestUploadSession_Abort_AlreadyCompleted_ReturnsErrCompleted(t *testing.T) {
	session := newCompletedSession()

	err := session.Abort()

	if err != ErrUploadSessionCompleted {
		t.Errorf("expected ErrUploadSessionCompleted, got: %v", err)
	}
}

func TestUploadSession_Abort_AlreadyAborted_ReturnsErrAborted(t *testing.T) {
	session := newAbortedSession()

	err := session.Abort()

	if err != ErrUploadSessionAborted {
		t.Errorf("expected ErrUploadSessionAborted, got: %v", err)
	}
}

func TestUploadSession_Abort_Expired_ReturnsNil(t *testing.T) {
	session := newExpiredSession()

	err := session.Abort()

	if err != nil {
		t.Errorf("Abort on expired session should succeed, got: %v", err)
	}
	if session.Status != UploadSessionStatusAborted {
		t.Errorf("expected status aborted, got: %s", session.Status)
	}
}

// IncrementUploadedParts tests

func TestUploadSession_IncrementUploadedParts_FromPending_TransitionsToInProgress(t *testing.T) {
	session := newPendingSession()

	session.IncrementUploadedParts()

	if session.Status != UploadSessionStatusInProgress {
		t.Errorf("expected status in_progress after first increment, got: %s", session.Status)
	}
}

func TestUploadSession_IncrementUploadedParts_FromInProgress_StaysInProgress(t *testing.T) {
	session := newInProgressSession()

	session.IncrementUploadedParts()

	if session.Status != UploadSessionStatusInProgress {
		t.Errorf("expected status in_progress, got: %s", session.Status)
	}
}

// IsExpired tests

func TestUploadSession_IsExpired_ExpiresAtInPast_ReturnsTrue(t *testing.T) {
	session := newExpiredSession()

	if !session.IsExpired() {
		t.Error("session with past ExpiresAt should be expired")
	}
}

func TestUploadSession_IsExpired_ExpiresAtInFuture_ReturnsFalse(t *testing.T) {
	session := newPendingSession()

	if session.IsExpired() {
		t.Error("session with future ExpiresAt should not be expired")
	}
}

// Progress tests

func TestUploadSession_Progress_ZeroOfTwo_ReturnsZero(t *testing.T) {
	fileID := uuid.New()
	createdBy := uuid.New()
	storageKey := valueobject.NewStorageKey(fileID)
	session := ReconstructUploadSession(
		uuid.New(), fileID, createdBy, createdBy, uuid.New(),
		newSessionFileName(), newSessionMimeType(), 1024, storageKey,
		nil, true, 2, 0,
		UploadSessionStatusInProgress,
		time.Now(), time.Now(), time.Now().Add(UploadSessionTTL),
	)

	if session.Progress() != 0 {
		t.Errorf("expected progress 0, got %d", session.Progress())
	}
}

func TestUploadSession_Progress_OneOfTwo_ReturnsFifty(t *testing.T) {
	fileID := uuid.New()
	createdBy := uuid.New()
	storageKey := valueobject.NewStorageKey(fileID)
	session := ReconstructUploadSession(
		uuid.New(), fileID, createdBy, createdBy, uuid.New(),
		newSessionFileName(), newSessionMimeType(), 1024, storageKey,
		nil, true, 2, 1,
		UploadSessionStatusInProgress,
		time.Now(), time.Now(), time.Now().Add(UploadSessionTTL),
	)

	if session.Progress() != 50 {
		t.Errorf("expected progress 50, got %d", session.Progress())
	}
}

func TestUploadSession_Progress_TwoOfTwo_ReturnsOneHundred(t *testing.T) {
	fileID := uuid.New()
	createdBy := uuid.New()
	storageKey := valueobject.NewStorageKey(fileID)
	session := ReconstructUploadSession(
		uuid.New(), fileID, createdBy, createdBy, uuid.New(),
		newSessionFileName(), newSessionMimeType(), 1024, storageKey,
		nil, true, 2, 2,
		UploadSessionStatusCompleted,
		time.Now(), time.Now(), time.Now().Add(UploadSessionTTL),
	)

	if session.Progress() != 100 {
		t.Errorf("expected progress 100, got %d", session.Progress())
	}
}

func TestUploadSession_Progress_TotalPartsZeroAndCompleted_ReturnsOneHundred(t *testing.T) {
	fileID := uuid.New()
	createdBy := uuid.New()
	storageKey := valueobject.NewStorageKey(fileID)
	session := ReconstructUploadSession(
		uuid.New(), fileID, createdBy, createdBy, uuid.New(),
		newSessionFileName(), newSessionMimeType(), 0, storageKey,
		nil, false, 0, 0,
		UploadSessionStatusCompleted,
		time.Now(), time.Now(), time.Now().Add(UploadSessionTTL),
	)

	if session.Progress() != 100 {
		t.Errorf("expected progress 100 for completed session with 0 parts, got %d", session.Progress())
	}
}

// AllPartsUploaded tests

func TestUploadSession_AllPartsUploaded_EqualCounts_ReturnsTrue(t *testing.T) {
	fileID := uuid.New()
	createdBy := uuid.New()
	storageKey := valueobject.NewStorageKey(fileID)
	session := ReconstructUploadSession(
		uuid.New(), fileID, createdBy, createdBy, uuid.New(),
		newSessionFileName(), newSessionMimeType(), 1024, storageKey,
		nil, false, 2, 2,
		UploadSessionStatusInProgress,
		time.Now(), time.Now(), time.Now().Add(UploadSessionTTL),
	)

	if !session.AllPartsUploaded() {
		t.Error("AllPartsUploaded should return true when uploaded == total")
	}
}

func TestUploadSession_AllPartsUploaded_LessThanTotal_ReturnsFalse(t *testing.T) {
	fileID := uuid.New()
	createdBy := uuid.New()
	storageKey := valueobject.NewStorageKey(fileID)
	session := ReconstructUploadSession(
		uuid.New(), fileID, createdBy, createdBy, uuid.New(),
		newSessionFileName(), newSessionMimeType(), 1024, storageKey,
		nil, false, 2, 1,
		UploadSessionStatusInProgress,
		time.Now(), time.Now(), time.Now().Add(UploadSessionTTL),
	)

	if session.AllPartsUploaded() {
		t.Error("AllPartsUploaded should return false when uploaded < total")
	}
}

// CanAcceptUpload tests

func TestUploadSession_CanAcceptUpload_PendingNotExpired_ReturnsTrue(t *testing.T) {
	session := newPendingSession()

	if !session.CanAcceptUpload() {
		t.Error("pending and not expired session should accept upload")
	}
}

func TestUploadSession_CanAcceptUpload_Aborted_ReturnsFalse(t *testing.T) {
	session := newAbortedSession()

	if session.CanAcceptUpload() {
		t.Error("aborted session should not accept upload")
	}
}

func TestUploadSession_CanAcceptUpload_Expired_ReturnsFalse(t *testing.T) {
	session := newExpiredSession()

	if session.CanAcceptUpload() {
		t.Error("expired session should not accept upload")
	}
}

// IsOwnedBy tests

func TestUploadSession_IsOwnedBy_MatchingOwner_ReturnsTrue(t *testing.T) {
	session := newPendingSession()

	if !session.IsOwnedBy(session.OwnerID) {
		t.Error("IsOwnedBy should return true for the session owner")
	}
}

func TestUploadSession_IsOwnedBy_DifferentOwner_ReturnsFalse(t *testing.T) {
	session := newPendingSession()

	if session.IsOwnedBy(uuid.New()) {
		t.Error("IsOwnedBy should return false for a different owner")
	}
}

// CalculatePartCount tests

func TestCalculatePartCount_ExactlyMinPartSize_ReturnsOne(t *testing.T) {
	count := CalculatePartCount(MinPartSize)

	if count != 1 {
		t.Errorf("expected 1 part for exactly MinPartSize, got %d", count)
	}
}

func TestCalculatePartCount_TenMB_ReturnsTwo(t *testing.T) {
	count := CalculatePartCount(10 * 1024 * 1024)

	if count != 2 {
		t.Errorf("expected 2 parts for 10MB, got %d", count)
	}
}

func TestCalculatePartCount_TenMBPlusOne_ReturnsThree(t *testing.T) {
	count := CalculatePartCount(10*1024*1024 + 1)

	if count != 3 {
		t.Errorf("expected 3 parts for 10MB+1, got %d", count)
	}
}

func TestCalculatePartCount_ExceedsMaxParts_CappedAtMax(t *testing.T) {
	hugeSizeBytes := int64(MaxMultipartParts+1) * MinPartSize
	count := CalculatePartCount(hugeSizeBytes)

	if count != MaxMultipartParts {
		t.Errorf("expected count capped at %d, got %d", MaxMultipartParts, count)
	}
}

// CalculatePartSize tests

func TestCalculatePartSize_NonLastPart_ReturnsMinPartSize(t *testing.T) {
	fileSize := int64(10 * 1024 * 1024) // 10MB, 2 parts
	size := CalculatePartSize(fileSize, 1, 2)

	if size != MinPartSize {
		t.Errorf("expected MinPartSize for non-last part, got %d", size)
	}
}

func TestCalculatePartSize_LastPart_ReturnsRemainder(t *testing.T) {
	fileSize := int64(7 * 1024 * 1024) // 7MB: part1=5MB, part2=2MB
	size := CalculatePartSize(fileSize, 2, 2)

	expected := fileSize - MinPartSize
	if size != expected {
		t.Errorf("expected last part size %d, got %d", expected, size)
	}
}
