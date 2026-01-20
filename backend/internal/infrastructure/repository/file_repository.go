package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database/sqlcgen"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// FileRepository はファイルリポジトリの実装です
type FileRepository struct {
	*database.BaseRepository
}

// NewFileRepository は新しいFileRepositoryを作成します
func NewFileRepository(txManager *database.TxManager) *FileRepository {
	return &FileRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create はファイルを作成します
func (r *FileRepository) Create(ctx context.Context, file *entity.File) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	_, err := queries.CreateFile(ctx, sqlcgen.CreateFileParams{
		ID:             file.ID,
		OwnerID:        file.OwnerID,
		OwnerType:      sqlcgen.OwnerType(file.OwnerType),
		FolderID:       uuidToPgtype(file.FolderID),
		Name:           file.Name.String(),
		MimeType:       file.MimeType.String(),
		Size:           file.Size,
		StorageKey:     file.StorageKey.String(),
		CurrentVersion: int32(file.CurrentVersion),
		Status:         sqlcgen.FileStatus(file.Status),
		CreatedAt:      file.CreatedAt,
		UpdatedAt:      file.UpdatedAt,
	})

	return r.HandleError(err)
}

// FindByID はIDでファイルを検索します
func (r *FileRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.File, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetFileByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("file")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row), nil
}

// Update はファイルを更新します
func (r *FileRepository) Update(ctx context.Context, file *entity.File) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	name := file.Name.String()
	size := file.Size
	currentVersion := int32(file.CurrentVersion)
	_, err := queries.UpdateFile(ctx, sqlcgen.UpdateFileParams{
		ID:             file.ID,
		FolderID:       uuidToPgtype(file.FolderID),
		Name:           &name,
		Size:           &size,
		CurrentVersion: &currentVersion,
	})

	return r.HandleError(err)
}

// Delete はファイルを削除します
func (r *FileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteFile(ctx, id)
	return r.HandleError(err)
}

// FindByFolderID はフォルダIDでファイルを検索します
func (r *FileRepository) FindByFolderID(ctx context.Context, folderID *uuid.UUID, ownerID uuid.UUID, ownerType valueobject.OwnerType) ([]*entity.File, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	// folderID が nil の場合はルートレベルのファイルを検索
	if folderID == nil {
		rows, err := queries.ListRootFilesByOwner(ctx, sqlcgen.ListRootFilesByOwnerParams{
			OwnerID:   ownerID,
			OwnerType: sqlcgen.OwnerType(ownerType),
		})
		if err != nil {
			return nil, r.HandleError(err)
		}
		return r.toEntities(rows), nil
	}

	rows, err := queries.ListFilesByFolderID(ctx, sqlcgen.ListFilesByFolderIDParams{
		FolderID:  uuidToPgtype(folderID),
		OwnerID:   ownerID,
		OwnerType: sqlcgen.OwnerType(ownerType),
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// FindByNameAndFolder はフォルダ内で名前でファイルを検索します
func (r *FileRepository) FindByNameAndFolder(ctx context.Context, name valueobject.FileName, folderID *uuid.UUID, ownerID uuid.UUID, ownerType valueobject.OwnerType) (*entity.File, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	// folderID が nil の場合はルートレベルで検索
	if folderID == nil {
		row, err := queries.GetFileByNameAtRoot(ctx, sqlcgen.GetFileByNameAtRootParams{
			OwnerID:   ownerID,
			OwnerType: sqlcgen.OwnerType(ownerType),
			Name:      name.String(),
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, apperror.NewNotFoundError("file")
			}
			return nil, r.HandleError(err)
		}
		return r.toEntity(row), nil
	}

	row, err := queries.GetFileByNameAndFolder(ctx, sqlcgen.GetFileByNameAndFolderParams{
		FolderID:  uuidToPgtype(folderID),
		OwnerID:   ownerID,
		OwnerType: sqlcgen.OwnerType(ownerType),
		Name:      name.String(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("file")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row), nil
}

// FindByOwner はオーナーの全ファイルを検索します
func (r *FileRepository) FindByOwner(ctx context.Context, ownerID uuid.UUID, ownerType valueobject.OwnerType) ([]*entity.File, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListFilesByOwner(ctx, sqlcgen.ListFilesByOwnerParams{
		OwnerID:   ownerID,
		OwnerType: sqlcgen.OwnerType(ownerType),
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// FindByStorageKey はストレージキーでファイルを検索します
func (r *FileRepository) FindByStorageKey(ctx context.Context, storageKey valueobject.StorageKey) (*entity.File, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetFileByStorageKey(ctx, storageKey.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("file")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row), nil
}

// ExistsByNameAndFolder はフォルダ内で同名ファイルの存在チェックをします
func (r *FileRepository) ExistsByNameAndFolder(ctx context.Context, name valueobject.FileName, folderID *uuid.UUID, ownerID uuid.UUID, ownerType valueobject.OwnerType) (bool, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	// folderID が nil の場合はルートレベルでチェック
	if folderID == nil {
		exists, err := queries.FileExistsByNameAtRoot(ctx, sqlcgen.FileExistsByNameAtRootParams{
			OwnerID:   ownerID,
			OwnerType: sqlcgen.OwnerType(ownerType),
			Name:      name.String(),
		})
		return exists, r.HandleError(err)
	}

	exists, err := queries.FileExistsByNameAndFolder(ctx, sqlcgen.FileExistsByNameAndFolderParams{
		FolderID:  uuidToPgtype(folderID),
		OwnerID:   ownerID,
		OwnerType: sqlcgen.OwnerType(ownerType),
		Name:      name.String(),
	})

	return exists, r.HandleError(err)
}

// UpdateStatus はファイルステータスを更新します
func (r *FileRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.FileStatus) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.UpdateFileStatus(ctx, sqlcgen.UpdateFileStatusParams{
		ID:     id,
		Status: sqlcgen.FileStatus(status),
	})

	return r.HandleError(err)
}

// FindByFolderIDs は複数フォルダIDでファイルを検索します
func (r *FileRepository) FindByFolderIDs(ctx context.Context, folderIDs []uuid.UUID) ([]*entity.File, error) {
	if len(folderIDs) == 0 {
		return []*entity.File{}, nil
	}

	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListFilesByFolderIDs(ctx, folderIDs)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// BulkDelete はファイルを一括削除します
func (r *FileRepository) BulkDelete(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteFilesBulk(ctx, ids)
	return r.HandleError(err)
}

// FindUploadFailed はアップロード失敗ファイルを検索します
func (r *FileRepository) FindUploadFailed(ctx context.Context) ([]*entity.File, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListUploadFailedFiles(ctx)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// toEntity はsqlcgen.Fileをentity.Fileに変換します
func (r *FileRepository) toEntity(row sqlcgen.File) *entity.File {
	name, _ := valueobject.NewFileName(row.Name)
	mimeType, _ := valueobject.NewMimeType(row.MimeType)
	storageKey, _ := valueobject.NewStorageKeyFromString(row.StorageKey)
	ownerType := valueobject.OwnerType(row.OwnerType)

	return entity.ReconstructFile(
		row.ID,
		row.OwnerID,
		ownerType,
		pgtypeToUUID(row.FolderID),
		name,
		mimeType,
		row.Size,
		storageKey,
		int(row.CurrentVersion),
		entity.FileStatus(row.Status),
		row.CreatedAt,
		row.UpdatedAt,
	)
}

// toEntities はsqlcgen.File配列をentity.File配列に変換します
func (r *FileRepository) toEntities(rows []sqlcgen.File) []*entity.File {
	entities := make([]*entity.File, len(rows))
	for i, row := range rows {
		entities[i] = r.toEntity(row)
	}
	return entities
}

// インターフェースの実装を保証
var _ repository.FileRepository = (*FileRepository)(nil)
