package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// ファイルステータス
type FileStatus string

const (
	FileStatusUploading    FileStatus = "uploading"
	FileStatusActive       FileStatus = "active"
	FileStatusUploadFailed FileStatus = "upload_failed"
)

// ファイル関連エラー
var (
	ErrFileNotActive         = errors.New("file is not active")
	ErrFileUploading         = errors.New("file upload is not completed")
	ErrFileUploadFailed      = errors.New("file upload has failed")
	ErrFileNameConflict      = errors.New("file name already exists in folder")
	ErrFileCannotDownload    = errors.New("file cannot be downloaded in current state")
	ErrFileInvalidTransition = errors.New("invalid file status transition")
)

// File はファイルエンティティ（集約ルート）
// Note: ファイルのゴミ箱移動は ArchivedFile テーブルへの移動として実現される。
// 論理削除（trashed_at）ではなく、別テーブルへ移動する設計。
type File struct {
	ID             uuid.UUID
	OwnerID        uuid.UUID
	OwnerType      valueobject.OwnerType
	FolderID       *uuid.UUID
	Name           valueobject.FileName
	MimeType       valueobject.MimeType
	Size           int64
	StorageKey     valueobject.StorageKey
	CurrentVersion int
	Status         FileStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewFile は新しいファイルを作成します（uploading状態で作成）
func NewFile(
	ownerID uuid.UUID,
	ownerType valueobject.OwnerType,
	folderID *uuid.UUID,
	name valueobject.FileName,
	mimeType valueobject.MimeType,
	size int64,
) *File {
	fileID := uuid.New()
	now := time.Now()
	return &File{
		ID:             fileID,
		OwnerID:        ownerID,
		OwnerType:      ownerType,
		FolderID:       folderID,
		Name:           name,
		MimeType:       mimeType,
		Size:           size,
		StorageKey:     valueobject.NewStorageKey(fileID),
		CurrentVersion: 1,
		Status:         FileStatusUploading,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// NewFileWithID は指定IDで新しいファイルを作成します（UploadSession連携用）
func NewFileWithID(
	fileID uuid.UUID,
	ownerID uuid.UUID,
	ownerType valueobject.OwnerType,
	folderID *uuid.UUID,
	name valueobject.FileName,
	mimeType valueobject.MimeType,
	size int64,
) *File {
	now := time.Now()
	return &File{
		ID:             fileID,
		OwnerID:        ownerID,
		OwnerType:      ownerType,
		FolderID:       folderID,
		Name:           name,
		MimeType:       mimeType,
		Size:           size,
		StorageKey:     valueobject.NewStorageKey(fileID),
		CurrentVersion: 1,
		Status:         FileStatusUploading,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// ReconstructFile はDBからファイルを復元します
func ReconstructFile(
	id uuid.UUID,
	ownerID uuid.UUID,
	ownerType valueobject.OwnerType,
	folderID *uuid.UUID,
	name valueobject.FileName,
	mimeType valueobject.MimeType,
	size int64,
	storageKey valueobject.StorageKey,
	currentVersion int,
	status FileStatus,
	createdAt time.Time,
	updatedAt time.Time,
) *File {
	return &File{
		ID:             id,
		OwnerID:        ownerID,
		OwnerType:      ownerType,
		FolderID:       folderID,
		Name:           name,
		MimeType:       mimeType,
		Size:           size,
		StorageKey:     storageKey,
		CurrentVersion: currentVersion,
		Status:         status,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
}

// Activate はファイルを有効化します（アップロード完了時に呼ばれる）
func (f *File) Activate() error {
	if f.Status != FileStatusUploading {
		return ErrFileInvalidTransition
	}
	f.Status = FileStatusActive
	f.UpdatedAt = time.Now()
	return nil
}

// MarkUploadFailed はアップロード失敗をマークします
func (f *File) MarkUploadFailed() error {
	if f.Status != FileStatusUploading {
		return ErrFileInvalidTransition
	}
	f.Status = FileStatusUploadFailed
	f.UpdatedAt = time.Now()
	return nil
}

// Rename はファイル名を変更します
func (f *File) Rename(newName valueobject.FileName) error {
	if f.Status != FileStatusActive {
		return ErrFileNotActive
	}
	f.Name = newName
	f.UpdatedAt = time.Now()
	return nil
}

// MoveTo はファイルを別のフォルダに移動します
func (f *File) MoveTo(newFolderID *uuid.UUID) error {
	if f.Status != FileStatusActive {
		return ErrFileNotActive
	}
	f.FolderID = newFolderID
	f.UpdatedAt = time.Now()
	return nil
}

// IncrementVersion はバージョンをインクリメントします
func (f *File) IncrementVersion() {
	f.CurrentVersion++
	f.UpdatedAt = time.Now()
}

// UpdateSize はファイルサイズを更新します
func (f *File) UpdateSize(size int64) {
	f.Size = size
	f.UpdatedAt = time.Now()
}

// CanDownload はダウンロード可能かどうかを判定します
func (f *File) CanDownload() bool {
	return f.Status == FileStatusActive
}

// IsActive はアクティブかどうかを判定します
func (f *File) IsActive() bool {
	return f.Status == FileStatusActive
}

// IsUploading はアップロード中かどうかを判定します
func (f *File) IsUploading() bool {
	return f.Status == FileStatusUploading
}

// IsUploadFailed はアップロード失敗かどうかを判定します
func (f *File) IsUploadFailed() bool {
	return f.Status == FileStatusUploadFailed
}

// IsOwnedBy は指定ユーザー/グループが所有者かどうかを判定します
func (f *File) IsOwnedBy(ownerID uuid.UUID, ownerType valueobject.OwnerType) bool {
	return f.OwnerID == ownerID && f.OwnerType == ownerType
}

// IsInFolder は指定フォルダ内にあるかどうかを判定します
func (f *File) IsInFolder(folderID uuid.UUID) bool {
	return f.FolderID != nil && *f.FolderID == folderID
}

// IsAtRoot はルートレベルにあるかどうかを判定します
func (f *File) IsAtRoot() bool {
	return f.FolderID == nil
}

// ToArchived はアーカイブ用のデータを生成します
func (f *File) ToArchived(originalPath string, archivedBy uuid.UUID) *ArchivedFile {
	return NewArchivedFile(
		f.ID,
		f.FolderID,
		originalPath,
		f.Name,
		f.MimeType,
		f.Size,
		f.OwnerID,
		f.OwnerType,
		f.StorageKey,
		archivedBy,
	)
}
