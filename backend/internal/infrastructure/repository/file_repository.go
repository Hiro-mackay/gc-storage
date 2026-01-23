package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

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
		FolderID:       file.FolderID,
		OwnerID:        file.OwnerID,
		CreatedBy:      file.CreatedBy,
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
		FolderID:       pgtype.UUID{Bytes: file.FolderID, Valid: true},
		OwnerID:        pgtype.UUID{Bytes: file.OwnerID, Valid: true},
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
func (r *FileRepository) FindByFolderID(ctx context.Context, folderID uuid.UUID) ([]*entity.File, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListFilesByFolderID(ctx, folderID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// FindByNameAndFolder はフォルダ内で名前でファイルを検索します
func (r *FileRepository) FindByNameAndFolder(ctx context.Context, name valueobject.FileName, folderID uuid.UUID) (*entity.File, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetFileByNameAndFolder(ctx, sqlcgen.GetFileByNameAndFolderParams{
		FolderID: folderID,
		Name:     name.String(),
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
func (r *FileRepository) FindByOwner(ctx context.Context, ownerID uuid.UUID) ([]*entity.File, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListFilesByOwner(ctx, ownerID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// FindByCreatedBy は作成者の全ファイルを検索します
func (r *FileRepository) FindByCreatedBy(ctx context.Context, createdBy uuid.UUID) ([]*entity.File, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListFilesByCreatedBy(ctx, createdBy)
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
func (r *FileRepository) ExistsByNameAndFolder(ctx context.Context, name valueobject.FileName, folderID uuid.UUID) (bool, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	exists, err := queries.FileExistsByNameAndFolder(ctx, sqlcgen.FileExistsByNameAndFolderParams{
		FolderID: folderID,
		Name:     name.String(),
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

	return entity.ReconstructFile(
		row.ID,
		row.FolderID,
		row.OwnerID,
		row.CreatedBy,
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
