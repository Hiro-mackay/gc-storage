package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database/sqlcgen"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// FileVersionRepository はファイルバージョンリポジトリの実装です
type FileVersionRepository struct {
	*database.BaseRepository
}

// NewFileVersionRepository は新しいFileVersionRepositoryを作成します
func NewFileVersionRepository(txManager *database.TxManager) *FileVersionRepository {
	return &FileVersionRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create はファイルバージョンを作成します
func (r *FileVersionRepository) Create(ctx context.Context, version *entity.FileVersion) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	var minioVersionID *string
	if version.MinioVersionID != "" {
		minioVersionID = &version.MinioVersionID
	}

	_, err := queries.CreateFileVersion(ctx, sqlcgen.CreateFileVersionParams{
		ID:             version.ID,
		FileID:         version.FileID,
		VersionNumber:  int32(version.VersionNumber),
		MinioVersionID: minioVersionID,
		Size:           version.Size,
		Checksum:       version.Checksum,
		UploadedBy:     version.UploadedBy,
		CreatedAt:      version.CreatedAt,
	})

	return r.HandleError(err)
}

// FindByID はIDでファイルバージョンを検索します
func (r *FileVersionRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.FileVersion, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetFileVersionByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("file version")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row), nil
}

// Delete はファイルバージョンを削除します
func (r *FileVersionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteFileVersion(ctx, id)
	return r.HandleError(err)
}

// FindByFileID はファイルIDでバージョンを検索します
func (r *FileVersionRepository) FindByFileID(ctx context.Context, fileID uuid.UUID) ([]*entity.FileVersion, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListFileVersionsByFileID(ctx, fileID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// FindByFileAndVersion はファイルIDとバージョン番号でバージョンを検索します
func (r *FileVersionRepository) FindByFileAndVersion(ctx context.Context, fileID uuid.UUID, versionNumber int) (*entity.FileVersion, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetFileVersionByFileAndVersion(ctx, sqlcgen.GetFileVersionByFileAndVersionParams{
		FileID:        fileID,
		VersionNumber: int32(versionNumber),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("file version")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row), nil
}

// FindLatestByFileID はファイルの最新バージョンを検索します
func (r *FileVersionRepository) FindLatestByFileID(ctx context.Context, fileID uuid.UUID) (*entity.FileVersion, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetLatestFileVersion(ctx, fileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("file version")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row), nil
}

// BulkCreate はファイルバージョンを一括作成します
func (r *FileVersionRepository) BulkCreate(ctx context.Context, versions []*entity.FileVersion) error {
	if len(versions) == 0 {
		return nil
	}

	querier := r.Querier(ctx)

	// CopyFrom を使用してバルクインサート
	params := make([]sqlcgen.CreateFileVersionsBulkParams, len(versions))
	for i, v := range versions {
		var minioVersionID *string
		if v.MinioVersionID != "" {
			minioVersionID = &v.MinioVersionID
		}
		params[i] = sqlcgen.CreateFileVersionsBulkParams{
			ID:             v.ID,
			FileID:         v.FileID,
			VersionNumber:  int32(v.VersionNumber),
			MinioVersionID: minioVersionID,
			Size:           v.Size,
			Checksum:       v.Checksum,
			UploadedBy:     v.UploadedBy,
			CreatedAt:      v.CreatedAt,
		}
	}

	queries := sqlcgen.New(querier)
	_, err := queries.CreateFileVersionsBulk(ctx, params)
	return r.HandleError(err)
}

// DeleteByFileID はファイルIDで全バージョンを削除します
func (r *FileVersionRepository) DeleteByFileID(ctx context.Context, fileID uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteFileVersionsByFileID(ctx, fileID)
	return r.HandleError(err)
}

// FindByFileIDs は複数ファイルIDでバージョンを検索します
func (r *FileVersionRepository) FindByFileIDs(ctx context.Context, fileIDs []uuid.UUID) ([]*entity.FileVersion, error) {
	if len(fileIDs) == 0 {
		return []*entity.FileVersion{}, nil
	}

	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListFileVersionsByFileIDs(ctx, fileIDs)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// CountByFileID はファイルのバージョン数をカウントします
func (r *FileVersionRepository) CountByFileID(ctx context.Context, fileID uuid.UUID) (int, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	count, err := queries.CountFileVersionsByFileID(ctx, fileID)
	if err != nil {
		return 0, r.HandleError(err)
	}

	return int(count), nil
}

// toEntity はsqlcgen.FileVersionをentity.FileVersionに変換します
func (r *FileVersionRepository) toEntity(row sqlcgen.FileVersion) *entity.FileVersion {
	minioVersionID := ""
	if row.MinioVersionID != nil {
		minioVersionID = *row.MinioVersionID
	}

	return entity.ReconstructFileVersion(
		row.ID,
		row.FileID,
		int(row.VersionNumber),
		minioVersionID,
		row.Size,
		row.Checksum,
		row.UploadedBy,
		row.CreatedAt,
	)
}

// toEntities はsqlcgen.FileVersion配列をentity.FileVersion配列に変換します
func (r *FileVersionRepository) toEntities(rows []sqlcgen.FileVersion) []*entity.FileVersion {
	entities := make([]*entity.FileVersion, len(rows))
	for i, row := range rows {
		entities[i] = r.toEntity(row)
	}
	return entities
}

// インターフェースの実装を保証
var _ repository.FileVersionRepository = (*FileVersionRepository)(nil)
