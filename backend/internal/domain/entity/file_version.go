package entity

import (
	"time"

	"github.com/google/uuid"
)

// FileVersion はファイルバージョンエンティティ
type FileVersion struct {
	ID             uuid.UUID
	FileID         uuid.UUID
	VersionNumber  int
	MinioVersionID string // MinIOが自動生成するバージョンID
	Size           int64
	Checksum       string // SHA-256チェックサム（必須）
	UploadedBy     uuid.UUID
	CreatedAt      time.Time
}

// NewFileVersion は新しいファイルバージョンを作成します
func NewFileVersion(
	fileID uuid.UUID,
	versionNumber int,
	minioVersionID string,
	size int64,
	checksum string,
	uploadedBy uuid.UUID,
) *FileVersion {
	return &FileVersion{
		ID:             uuid.New(),
		FileID:         fileID,
		VersionNumber:  versionNumber,
		MinioVersionID: minioVersionID,
		Size:           size,
		Checksum:       checksum,
		UploadedBy:     uploadedBy,
		CreatedAt:      time.Now(),
	}
}

// ReconstructFileVersion はDBからファイルバージョンを復元します
func ReconstructFileVersion(
	id uuid.UUID,
	fileID uuid.UUID,
	versionNumber int,
	minioVersionID string,
	size int64,
	checksum string,
	uploadedBy uuid.UUID,
	createdAt time.Time,
) *FileVersion {
	return &FileVersion{
		ID:             id,
		FileID:         fileID,
		VersionNumber:  versionNumber,
		MinioVersionID: minioVersionID,
		Size:           size,
		Checksum:       checksum,
		UploadedBy:     uploadedBy,
		CreatedAt:      createdAt,
	}
}

// IsLatest は最新バージョンかどうかを判定します（Fileのcurrent_versionと比較）
func (fv *FileVersion) IsLatest(currentVersion int) bool {
	return fv.VersionNumber == currentVersion
}

// ToArchived はアーカイブ用のデータを生成します
func (fv *FileVersion) ToArchived(archivedFileID uuid.UUID) *ArchivedFileVersion {
	return &ArchivedFileVersion{
		ID:                uuid.New(),
		ArchivedFileID:    archivedFileID,
		OriginalVersionID: fv.ID,
		VersionNumber:     fv.VersionNumber,
		MinioVersionID:    fv.MinioVersionID,
		Size:              fv.Size,
		Checksum:          fv.Checksum,
		UploadedBy:        fv.UploadedBy,
		CreatedAt:         fv.CreatedAt,
	}
}
