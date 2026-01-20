package entity

import (
	"time"

	"github.com/google/uuid"
)

// UploadPart はマルチパートアップロードの各パーツ情報エンティティ
type UploadPart struct {
	ID         uuid.UUID
	SessionID  uuid.UUID
	PartNumber int
	Size       int64
	ETag       string // MinIOから返却されたETag
	UploadedAt time.Time
}

// NewUploadPart は新しいUploadPartを作成します
func NewUploadPart(
	sessionID uuid.UUID,
	partNumber int,
	size int64,
	etag string,
) *UploadPart {
	return &UploadPart{
		ID:         uuid.New(),
		SessionID:  sessionID,
		PartNumber: partNumber,
		Size:       size,
		ETag:       etag,
		UploadedAt: time.Now(),
	}
}

// ReconstructUploadPart はDBからUploadPartを復元します
func ReconstructUploadPart(
	id uuid.UUID,
	sessionID uuid.UUID,
	partNumber int,
	size int64,
	etag string,
	uploadedAt time.Time,
) *UploadPart {
	return &UploadPart{
		ID:         id,
		SessionID:  sessionID,
		PartNumber: partNumber,
		Size:       size,
		ETag:       etag,
		UploadedAt: uploadedAt,
	}
}
