package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewFileVersion_SetsFieldsCorrectly(t *testing.T) {
	fileID := uuid.New()
	uploadedBy := uuid.New()
	versionNumber := 2
	minioVersionID := "minio-version-abc123"
	size := int64(4096)
	checksum := "sha256:abcdef1234567890"

	before := time.Now()
	fv := NewFileVersion(fileID, versionNumber, minioVersionID, size, checksum, uploadedBy)
	after := time.Now()

	if fv.ID == uuid.Nil {
		t.Error("NewFileVersion should assign a non-nil ID")
	}
	if fv.FileID != fileID {
		t.Errorf("expected FileID %v, got %v", fileID, fv.FileID)
	}
	if fv.VersionNumber != versionNumber {
		t.Errorf("expected VersionNumber %d, got %d", versionNumber, fv.VersionNumber)
	}
	if fv.MinioVersionID != minioVersionID {
		t.Errorf("expected MinioVersionID %q, got %q", minioVersionID, fv.MinioVersionID)
	}
	if fv.Size != size {
		t.Errorf("expected Size %d, got %d", size, fv.Size)
	}
	if fv.Checksum != checksum {
		t.Errorf("expected Checksum %q, got %q", checksum, fv.Checksum)
	}
	if fv.UploadedBy != uploadedBy {
		t.Errorf("expected UploadedBy %v, got %v", uploadedBy, fv.UploadedBy)
	}
	if fv.CreatedAt.Before(before) || fv.CreatedAt.After(after) {
		t.Errorf("expected CreatedAt between %v and %v, got %v", before, after, fv.CreatedAt)
	}
}

func TestFileVersion_IsLatest_MatchingVersion_ReturnsTrue(t *testing.T) {
	fv := NewFileVersion(uuid.New(), 3, "minio-v3", 512, "sha256:abc", uuid.New())

	if !fv.IsLatest(3) {
		t.Error("IsLatest should return true when VersionNumber matches currentVersion")
	}
}

func TestFileVersion_IsLatest_NonMatchingVersion_ReturnsFalse(t *testing.T) {
	fv := NewFileVersion(uuid.New(), 2, "minio-v2", 512, "sha256:abc", uuid.New())

	if fv.IsLatest(5) {
		t.Error("IsLatest should return false when VersionNumber does not match currentVersion")
	}
}
