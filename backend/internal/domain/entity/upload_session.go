package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// アップロードセッションステータス
type UploadSessionStatus string

const (
	UploadSessionStatusPending    UploadSessionStatus = "pending"
	UploadSessionStatusInProgress UploadSessionStatus = "in_progress"
	UploadSessionStatusCompleted  UploadSessionStatus = "completed"
	UploadSessionStatusAborted    UploadSessionStatus = "aborted"
	UploadSessionStatusExpired    UploadSessionStatus = "expired"
)

// アップロード関連定数
const (
	UploadSessionTTL     = 24 * time.Hour
	MultipartThreshold   = 5 * 1024 * 1024  // 5MB
	MinPartSize          = 5 * 1024 * 1024  // 5MB
	MaxPartSize          = 5 * 1024 * 1024 * 1024 // 5GB
	MaxMultipartParts    = 10000
)

// アップロードセッション関連エラー
var (
	ErrUploadSessionExpired       = errors.New("upload session expired")
	ErrUploadSessionCompleted     = errors.New("upload session already completed")
	ErrUploadSessionAborted       = errors.New("upload session already aborted")
	ErrUploadSessionInvalidStatus = errors.New("invalid upload session status")
)

// UploadSession はアップロードセッションエンティティ
// Note: owner_typeは削除。セッションは常にユーザーが所有者。グループはPermissionGrantでアクセス。
type UploadSession struct {
	ID            uuid.UUID
	FileID        uuid.UUID              // 作成予定のファイルID（事前生成）
	OwnerID       uuid.UUID              // 現在の所有者ID
	CreatedBy     uuid.UUID              // 作成者ID（アップロード者）
	FolderID      uuid.UUID              // 必須 - アップロード先フォルダID
	FileName      valueobject.FileName
	MimeType      valueobject.MimeType
	TotalSize     int64
	StorageKey    valueobject.StorageKey
	MinioUploadID *string // MinIOマルチパートアップロードID
	IsMultipart   bool
	TotalParts    int
	UploadedParts int
	Status        UploadSessionStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ExpiresAt     time.Time
}

// NewUploadSession は新しいアップロードセッションを作成します
// 新規作成時は owner_id = created_by = 作成者（createdBy引数）となります
func NewUploadSession(
	fileID uuid.UUID,
	createdBy uuid.UUID,
	folderID uuid.UUID,
	fileName valueobject.FileName,
	mimeType valueobject.MimeType,
	totalSize int64,
	minioUploadID *string,
) *UploadSession {
	isMultipart := totalSize >= MultipartThreshold
	totalParts := 1
	if isMultipart {
		totalParts = CalculatePartCount(totalSize)
	}

	now := time.Now()
	return &UploadSession{
		ID:            uuid.New(),
		FileID:        fileID,
		OwnerID:       createdBy, // 新規作成時は owner_id = created_by
		CreatedBy:     createdBy,
		FolderID:      folderID,
		FileName:      fileName,
		MimeType:      mimeType,
		TotalSize:     totalSize,
		StorageKey:    valueobject.NewStorageKey(fileID),
		MinioUploadID: minioUploadID,
		IsMultipart:   isMultipart,
		TotalParts:    totalParts,
		UploadedParts: 0,
		Status:        UploadSessionStatusPending,
		CreatedAt:     now,
		UpdatedAt:     now,
		ExpiresAt:     now.Add(UploadSessionTTL),
	}
}

// ReconstructUploadSession はDBからアップロードセッションを復元します
func ReconstructUploadSession(
	id uuid.UUID,
	fileID uuid.UUID,
	ownerID uuid.UUID,
	createdBy uuid.UUID,
	folderID uuid.UUID,
	fileName valueobject.FileName,
	mimeType valueobject.MimeType,
	totalSize int64,
	storageKey valueobject.StorageKey,
	minioUploadID *string,
	isMultipart bool,
	totalParts int,
	uploadedParts int,
	status UploadSessionStatus,
	createdAt time.Time,
	updatedAt time.Time,
	expiresAt time.Time,
) *UploadSession {
	return &UploadSession{
		ID:            id,
		FileID:        fileID,
		OwnerID:       ownerID,
		CreatedBy:     createdBy,
		FolderID:      folderID,
		FileName:      fileName,
		MimeType:      mimeType,
		TotalSize:     totalSize,
		StorageKey:    storageKey,
		MinioUploadID: minioUploadID,
		IsMultipart:   isMultipart,
		TotalParts:    totalParts,
		UploadedParts: uploadedParts,
		Status:        status,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		ExpiresAt:     expiresAt,
	}
}

// Complete はアップロードを完了状態にします
func (us *UploadSession) Complete() error {
	if us.Status == UploadSessionStatusCompleted {
		return ErrUploadSessionCompleted
	}
	if us.Status == UploadSessionStatusAborted {
		return ErrUploadSessionAborted
	}
	if us.IsExpired() {
		return ErrUploadSessionExpired
	}

	us.Status = UploadSessionStatusCompleted
	us.UpdatedAt = time.Now()
	return nil
}

// Abort はアップロードを中断状態にします
func (us *UploadSession) Abort() error {
	if us.Status == UploadSessionStatusCompleted {
		return ErrUploadSessionCompleted
	}
	if us.Status == UploadSessionStatusAborted {
		return ErrUploadSessionAborted
	}

	us.Status = UploadSessionStatusAborted
	us.UpdatedAt = time.Now()
	return nil
}

// MarkExpired はセッションを期限切れにします
func (us *UploadSession) MarkExpired() {
	us.Status = UploadSessionStatusExpired
	us.UpdatedAt = time.Now()
}

// IncrementUploadedParts はアップロード済みパーツ数をインクリメントします
func (us *UploadSession) IncrementUploadedParts() {
	us.UploadedParts++
	if us.Status == UploadSessionStatusPending && us.UploadedParts > 0 {
		us.Status = UploadSessionStatusInProgress
	}
	us.UpdatedAt = time.Now()
}

// IsExpired はセッションが期限切れかどうかを判定します
func (us *UploadSession) IsExpired() bool {
	return time.Now().After(us.ExpiresAt)
}

// IsPending はペンディング状態かどうかを判定します
func (us *UploadSession) IsPending() bool {
	return us.Status == UploadSessionStatusPending
}

// IsInProgress は進行中かどうかを判定します
func (us *UploadSession) IsInProgress() bool {
	return us.Status == UploadSessionStatusInProgress
}

// IsCompleted は完了済みかどうかを判定します
func (us *UploadSession) IsCompleted() bool {
	return us.Status == UploadSessionStatusCompleted
}

// IsAborted は中断済みかどうかを判定します
func (us *UploadSession) IsAborted() bool {
	return us.Status == UploadSessionStatusAborted
}

// CanAcceptUpload はアップロードを受け付けられるかどうかを判定します
func (us *UploadSession) CanAcceptUpload() bool {
	return (us.Status == UploadSessionStatusPending || us.Status == UploadSessionStatusInProgress) && !us.IsExpired()
}

// AllPartsUploaded は全パーツがアップロード済みかどうかを判定します
func (us *UploadSession) AllPartsUploaded() bool {
	return us.UploadedParts >= us.TotalParts
}

// Progress はアップロード進捗を返します（0-100）
func (us *UploadSession) Progress() int {
	if us.TotalParts == 0 {
		if us.Status == UploadSessionStatusCompleted {
			return 100
		}
		return 0
	}

	return (us.UploadedParts * 100) / us.TotalParts
}

// IsOwnedBy は指定ユーザーが所有者かどうかを判定します
// Note: セッションは常にユーザーが所有者（グループはPermissionGrantでアクセス）
func (us *UploadSession) IsOwnedBy(ownerID uuid.UUID) bool {
	return us.OwnerID == ownerID
}

// IsCreatedBy は指定ユーザーが作成者かどうかを判定します
func (us *UploadSession) IsCreatedBy(userID uuid.UUID) bool {
	return us.CreatedBy == userID
}

// CalculatePartCount はファイルサイズからパート数を計算します
func CalculatePartCount(fileSize int64) int {
	if fileSize <= MinPartSize {
		return 1
	}

	partCount := int(fileSize / MinPartSize)
	if fileSize%MinPartSize > 0 {
		partCount++
	}

	if partCount > MaxMultipartParts {
		partCount = MaxMultipartParts
	}

	return partCount
}

// CalculatePartSize は各パートのサイズを計算します
func CalculatePartSize(fileSize int64, partNumber, totalParts int) int64 {
	if partNumber < totalParts {
		return MinPartSize
	}
	// 最後のパートは残りサイズ
	return fileSize - MinPartSize*int64(totalParts-1)
}
