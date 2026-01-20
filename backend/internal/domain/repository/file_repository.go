package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// FileRepository はファイルリポジトリのインターフェース
type FileRepository interface {
	// 基本CRUD
	Create(ctx context.Context, file *entity.File) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.File, error)
	Update(ctx context.Context, file *entity.File) error
	Delete(ctx context.Context, id uuid.UUID) error

	// 検索
	FindByFolderID(ctx context.Context, folderID *uuid.UUID, ownerID uuid.UUID, ownerType valueobject.OwnerType) ([]*entity.File, error)
	FindByNameAndFolder(ctx context.Context, name valueobject.FileName, folderID *uuid.UUID, ownerID uuid.UUID, ownerType valueobject.OwnerType) (*entity.File, error)
	FindByOwner(ctx context.Context, ownerID uuid.UUID, ownerType valueobject.OwnerType) ([]*entity.File, error)
	FindByStorageKey(ctx context.Context, storageKey valueobject.StorageKey) (*entity.File, error)

	// 存在チェック
	ExistsByNameAndFolder(ctx context.Context, name valueobject.FileName, folderID *uuid.UUID, ownerID uuid.UUID, ownerType valueobject.OwnerType) (bool, error)

	// ステータス更新
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.FileStatus) error

	// 一括操作（フォルダ削除用）
	FindByFolderIDs(ctx context.Context, folderIDs []uuid.UUID) ([]*entity.File, error)
	BulkDelete(ctx context.Context, ids []uuid.UUID) error

	// アップロード失敗ファイル（クリーンアップ用）
	FindUploadFailed(ctx context.Context) ([]*entity.File, error)
}

// FileVersionRepository はファイルバージョンリポジトリのインターフェース
type FileVersionRepository interface {
	// 基本CRUD
	Create(ctx context.Context, version *entity.FileVersion) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.FileVersion, error)
	Delete(ctx context.Context, id uuid.UUID) error

	// 検索
	FindByFileID(ctx context.Context, fileID uuid.UUID) ([]*entity.FileVersion, error)
	FindByFileAndVersion(ctx context.Context, fileID uuid.UUID, versionNumber int) (*entity.FileVersion, error)
	FindLatestByFileID(ctx context.Context, fileID uuid.UUID) (*entity.FileVersion, error)

	// 一括操作
	BulkCreate(ctx context.Context, versions []*entity.FileVersion) error
	DeleteByFileID(ctx context.Context, fileID uuid.UUID) error
	FindByFileIDs(ctx context.Context, fileIDs []uuid.UUID) ([]*entity.FileVersion, error)

	// カウント
	CountByFileID(ctx context.Context, fileID uuid.UUID) (int, error)
}

// ArchivedFileRepository はゴミ箱ファイルリポジトリのインターフェース
type ArchivedFileRepository interface {
	// 基本CRUD
	Create(ctx context.Context, archivedFile *entity.ArchivedFile) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.ArchivedFile, error)
	Delete(ctx context.Context, id uuid.UUID) error

	// 検索
	FindByOwner(ctx context.Context, ownerID uuid.UUID, ownerType valueobject.OwnerType) ([]*entity.ArchivedFile, error)
	FindExpired(ctx context.Context) ([]*entity.ArchivedFile, error)
	FindByOriginalFileID(ctx context.Context, originalFileID uuid.UUID) (*entity.ArchivedFile, error)
}

// ArchivedFileVersionRepository はゴミ箱ファイルバージョンリポジトリのインターフェース
type ArchivedFileVersionRepository interface {
	// 基本操作
	BulkCreate(ctx context.Context, versions []*entity.ArchivedFileVersion) error
	FindByArchivedFileID(ctx context.Context, archivedFileID uuid.UUID) ([]*entity.ArchivedFileVersion, error)
	DeleteByArchivedFileID(ctx context.Context, archivedFileID uuid.UUID) error
}

// UploadSessionRepository はアップロードセッションリポジトリのインターフェース
type UploadSessionRepository interface {
	// 基本CRUD
	Create(ctx context.Context, session *entity.UploadSession) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.UploadSession, error)
	Update(ctx context.Context, session *entity.UploadSession) error
	Delete(ctx context.Context, id uuid.UUID) error

	// 検索
	FindByFileID(ctx context.Context, fileID uuid.UUID) (*entity.UploadSession, error)
	FindByStorageKey(ctx context.Context, storageKey valueobject.StorageKey) (*entity.UploadSession, error)

	// 期限切れ検索（クリーンアップ用）
	FindExpired(ctx context.Context) ([]*entity.UploadSession, error)

	// ステータス更新
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.UploadSessionStatus) error
}

// UploadPartRepository はアップロードパーツリポジトリのインターフェース
type UploadPartRepository interface {
	// 基本操作
	Create(ctx context.Context, part *entity.UploadPart) error
	FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*entity.UploadPart, error)
	DeleteBySessionID(ctx context.Context, sessionID uuid.UUID) error
}
