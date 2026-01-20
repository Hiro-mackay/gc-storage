package entity

import (
	"time"

	"github.com/google/uuid"
)

// ArchivedFileVersion はゴミ箱内ファイルのバージョン情報エンティティ
type ArchivedFileVersion struct {
	ID                uuid.UUID
	ArchivedFileID    uuid.UUID
	OriginalVersionID uuid.UUID
	VersionNumber     int
	MinioVersionID    string
	Size              int64
	Checksum          string
	UploadedBy        uuid.UUID
	CreatedAt         time.Time // 元の作成日時
}

// NewArchivedFileVersion は新しいArchivedFileVersionを作成します
func NewArchivedFileVersion(
	archivedFileID uuid.UUID,
	originalVersionID uuid.UUID,
	versionNumber int,
	minioVersionID string,
	size int64,
	checksum string,
	uploadedBy uuid.UUID,
	createdAt time.Time,
) *ArchivedFileVersion {
	return &ArchivedFileVersion{
		ID:                uuid.New(),
		ArchivedFileID:    archivedFileID,
		OriginalVersionID: originalVersionID,
		VersionNumber:     versionNumber,
		MinioVersionID:    minioVersionID,
		Size:              size,
		Checksum:          checksum,
		UploadedBy:        uploadedBy,
		CreatedAt:         createdAt,
	}
}

// ReconstructArchivedFileVersion はDBからArchivedFileVersionを復元します
func ReconstructArchivedFileVersion(
	id uuid.UUID,
	archivedFileID uuid.UUID,
	originalVersionID uuid.UUID,
	versionNumber int,
	minioVersionID string,
	size int64,
	checksum string,
	uploadedBy uuid.UUID,
	createdAt time.Time,
) *ArchivedFileVersion {
	return &ArchivedFileVersion{
		ID:                id,
		ArchivedFileID:    archivedFileID,
		OriginalVersionID: originalVersionID,
		VersionNumber:     versionNumber,
		MinioVersionID:    minioVersionID,
		Size:              size,
		Checksum:          checksum,
		UploadedBy:        uploadedBy,
		CreatedAt:         createdAt,
	}
}

// ToFileVersion は復元用のFileVersionデータを生成します
func (afv *ArchivedFileVersion) ToFileVersion(fileID uuid.UUID) *FileVersion {
	return &FileVersion{
		ID:             afv.OriginalVersionID,
		FileID:         fileID,
		VersionNumber:  afv.VersionNumber,
		MinioVersionID: afv.MinioVersionID,
		Size:           afv.Size,
		Checksum:       afv.Checksum,
		UploadedBy:     afv.UploadedBy,
		CreatedAt:      afv.CreatedAt,
	}
}
