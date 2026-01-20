package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database/sqlcgen"
)

// ArchivedFileVersionRepository はアーカイブファイルバージョンリポジトリの実装です
type ArchivedFileVersionRepository struct {
	*database.BaseRepository
}

// NewArchivedFileVersionRepository は新しいArchivedFileVersionRepositoryを作成します
func NewArchivedFileVersionRepository(txManager *database.TxManager) *ArchivedFileVersionRepository {
	return &ArchivedFileVersionRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// BulkCreate はアーカイブファイルバージョンを一括作成します
func (r *ArchivedFileVersionRepository) BulkCreate(ctx context.Context, versions []*entity.ArchivedFileVersion) error {
	if len(versions) == 0 {
		return nil
	}

	querier := r.Querier(ctx)

	// CopyFrom を使用してバルクインサート
	params := make([]sqlcgen.CreateArchivedFileVersionsBulkParams, len(versions))
	for i, v := range versions {
		params[i] = sqlcgen.CreateArchivedFileVersionsBulkParams{
			ID:                v.ID,
			ArchivedFileID:    v.ArchivedFileID,
			OriginalVersionID: v.OriginalVersionID,
			VersionNumber:     int32(v.VersionNumber),
			MinioVersionID:    v.MinioVersionID,
			Size:              v.Size,
			Checksum:          v.Checksum,
			UploadedBy:        v.UploadedBy,
			CreatedAt:         v.CreatedAt,
		}
	}

	queries := sqlcgen.New(querier)
	_, err := queries.CreateArchivedFileVersionsBulk(ctx, params)
	return r.HandleError(err)
}

// FindByArchivedFileID はアーカイブファイルIDでバージョンを検索します
func (r *ArchivedFileVersionRepository) FindByArchivedFileID(ctx context.Context, archivedFileID uuid.UUID) ([]*entity.ArchivedFileVersion, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListArchivedFileVersionsByArchivedFileID(ctx, archivedFileID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// DeleteByArchivedFileID はアーカイブファイルIDで全バージョンを削除します
func (r *ArchivedFileVersionRepository) DeleteByArchivedFileID(ctx context.Context, archivedFileID uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteArchivedFileVersionsByArchivedFileID(ctx, archivedFileID)
	return r.HandleError(err)
}

// toEntity はsqlcgen.ArchivedFileVersionをentity.ArchivedFileVersionに変換します
func (r *ArchivedFileVersionRepository) toEntity(row sqlcgen.ArchivedFileVersion) *entity.ArchivedFileVersion {
	return entity.ReconstructArchivedFileVersion(
		row.ID,
		row.ArchivedFileID,
		row.OriginalVersionID,
		int(row.VersionNumber),
		row.MinioVersionID,
		row.Size,
		row.Checksum,
		row.UploadedBy,
		row.CreatedAt,
	)
}

// toEntities はsqlcgen.ArchivedFileVersion配列をentity.ArchivedFileVersion配列に変換します
func (r *ArchivedFileVersionRepository) toEntities(rows []sqlcgen.ArchivedFileVersion) []*entity.ArchivedFileVersion {
	entities := make([]*entity.ArchivedFileVersion, len(rows))
	for i, row := range rows {
		entities[i] = r.toEntity(row)
	}
	return entities
}

// インターフェースの実装を保証
var _ repository.ArchivedFileVersionRepository = (*ArchivedFileVersionRepository)(nil)
