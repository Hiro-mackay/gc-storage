package entity

import (
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// ゴミ箱関連の定数
const (
	TrashRetentionDays = 30
)

// ArchivedFile はゴミ箱に移動されたファイルエンティティ
// 復元に必要な情報を完全に保持する
// Note: owner_typeは削除。ファイルは常にユーザーが所有者。グループはPermissionGrantでアクセス。
type ArchivedFile struct {
	ID               uuid.UUID
	OriginalFileID   uuid.UUID
	OriginalFolderID uuid.UUID // 必須 - 復元先フォルダID
	OriginalPath     string    // 復元時の参考パス（例: "/documents/report.pdf"）
	Name             valueobject.FileName
	MimeType         valueobject.MimeType
	Size             int64
	OwnerID          uuid.UUID // 現在の所有者ID
	CreatedBy        uuid.UUID // 最初の作成者ID（不変）
	StorageKey       valueobject.StorageKey
	ArchivedAt       time.Time
	ArchivedBy       uuid.UUID
	ExpiresAt        time.Time
}

// NewArchivedFile は新しいArchivedFileを作成します
func NewArchivedFile(
	originalFileID uuid.UUID,
	originalFolderID uuid.UUID,
	originalPath string,
	name valueobject.FileName,
	mimeType valueobject.MimeType,
	size int64,
	ownerID uuid.UUID,
	createdBy uuid.UUID,
	storageKey valueobject.StorageKey,
	archivedBy uuid.UUID,
) *ArchivedFile {
	now := time.Now()
	return &ArchivedFile{
		ID:               uuid.New(),
		OriginalFileID:   originalFileID,
		OriginalFolderID: originalFolderID,
		OriginalPath:     originalPath,
		Name:             name,
		MimeType:         mimeType,
		Size:             size,
		OwnerID:          ownerID,
		CreatedBy:        createdBy,
		StorageKey:       storageKey,
		ArchivedAt:       now,
		ArchivedBy:       archivedBy,
		ExpiresAt:        now.AddDate(0, 0, TrashRetentionDays),
	}
}

// ReconstructArchivedFile はDBからArchivedFileを復元します
func ReconstructArchivedFile(
	id uuid.UUID,
	originalFileID uuid.UUID,
	originalFolderID uuid.UUID,
	originalPath string,
	name valueobject.FileName,
	mimeType valueobject.MimeType,
	size int64,
	ownerID uuid.UUID,
	createdBy uuid.UUID,
	storageKey valueobject.StorageKey,
	archivedAt time.Time,
	archivedBy uuid.UUID,
	expiresAt time.Time,
) *ArchivedFile {
	return &ArchivedFile{
		ID:               id,
		OriginalFileID:   originalFileID,
		OriginalFolderID: originalFolderID,
		OriginalPath:     originalPath,
		Name:             name,
		MimeType:         mimeType,
		Size:             size,
		OwnerID:          ownerID,
		CreatedBy:        createdBy,
		StorageKey:       storageKey,
		ArchivedAt:       archivedAt,
		ArchivedBy:       archivedBy,
		ExpiresAt:        expiresAt,
	}
}

// IsExpired は期限切れかどうかを判定します
func (af *ArchivedFile) IsExpired() bool {
	return time.Now().After(af.ExpiresAt)
}

// IsOwnedBy は指定ユーザーが所有者かどうかを判定します
// Note: ファイルは常にユーザーが所有者（グループはPermissionGrantでアクセス）
func (af *ArchivedFile) IsOwnedBy(ownerID uuid.UUID) bool {
	return af.OwnerID == ownerID
}

// IsCreatedBy は指定ユーザーが作成者かどうかを判定します
func (af *ArchivedFile) IsCreatedBy(userID uuid.UUID) bool {
	return af.CreatedBy == userID
}

// ToFile は復元用のFileデータを生成します
func (af *ArchivedFile) ToFile(restoreFolderID uuid.UUID) *File {
	now := time.Now()
	return &File{
		ID:             af.OriginalFileID,
		FolderID:       restoreFolderID,
		OwnerID:        af.OwnerID,
		CreatedBy:      af.CreatedBy,
		Name:           af.Name,
		MimeType:       af.MimeType,
		Size:           af.Size,
		StorageKey:     af.StorageKey,
		CurrentVersion: 1, // 復元時にFileVersionから再計算が必要
		Status:         FileStatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// DaysUntilExpiration は期限切れまでの日数を返します
func (af *ArchivedFile) DaysUntilExpiration() int {
	duration := time.Until(af.ExpiresAt)
	if duration < 0 {
		return 0
	}
	return int(duration.Hours() / 24)
}
