package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewUploadPart_AssignsSessionID(t *testing.T) {
	sessionID := uuid.New()

	part := NewUploadPart(sessionID, 1, 1024, "etag-abc")

	if part.SessionID != sessionID {
		t.Errorf("expected SessionID %v, got %v", sessionID, part.SessionID)
	}
}

func TestNewUploadPart_AssignsPartNumber(t *testing.T) {
	part := NewUploadPart(uuid.New(), 3, 1024, "etag-abc")

	if part.PartNumber != 3 {
		t.Errorf("expected PartNumber 3, got %d", part.PartNumber)
	}
}

func TestNewUploadPart_AssignsSize(t *testing.T) {
	part := NewUploadPart(uuid.New(), 1, 2048, "etag-abc")

	if part.Size != 2048 {
		t.Errorf("expected Size 2048, got %d", part.Size)
	}
}

func TestNewUploadPart_AssignsETag(t *testing.T) {
	part := NewUploadPart(uuid.New(), 1, 1024, "etag-xyz")

	if part.ETag != "etag-xyz" {
		t.Errorf("expected ETag %q, got %q", "etag-xyz", part.ETag)
	}
}

func TestNewUploadPart_GeneratesNonZeroID(t *testing.T) {
	part := NewUploadPart(uuid.New(), 1, 1024, "etag-abc")

	if part.ID == uuid.Nil {
		t.Error("NewUploadPart should generate a non-zero UUID")
	}
}

func TestNewUploadPart_SetsUploadedAt(t *testing.T) {
	before := time.Now()
	part := NewUploadPart(uuid.New(), 1, 1024, "etag-abc")
	after := time.Now()

	if part.UploadedAt.Before(before) || part.UploadedAt.After(after) {
		t.Error("NewUploadPart should set UploadedAt to approximately now")
	}
}

func TestReconstructUploadPart_AssignsAllFields(t *testing.T) {
	id := uuid.New()
	sessionID := uuid.New()
	uploadedAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	part := ReconstructUploadPart(id, sessionID, 2, 5120, "etag-reconstructed", uploadedAt)

	if part.ID != id {
		t.Errorf("expected ID %v, got %v", id, part.ID)
	}
	if part.SessionID != sessionID {
		t.Errorf("expected SessionID %v, got %v", sessionID, part.SessionID)
	}
	if part.PartNumber != 2 {
		t.Errorf("expected PartNumber 2, got %d", part.PartNumber)
	}
	if part.Size != 5120 {
		t.Errorf("expected Size 5120, got %d", part.Size)
	}
	if part.ETag != "etag-reconstructed" {
		t.Errorf("expected ETag %q, got %q", "etag-reconstructed", part.ETag)
	}
	if !part.UploadedAt.Equal(uploadedAt) {
		t.Errorf("expected UploadedAt %v, got %v", uploadedAt, part.UploadedAt)
	}
}
