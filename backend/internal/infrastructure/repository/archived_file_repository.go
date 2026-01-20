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

// ArchivedFileRepository はアーカイブファイルリポジトリの実装です
type ArchivedFileRepository struct {
	*database.BaseRepository
}

// NewArchivedFileRepository は新しいArchivedFileRepositoryを作成します
func NewArchivedFileRepository(txManager *database.TxManager) *ArchivedFileRepository {
	return &ArchivedFileRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create はアーカイブファイルを作成します
func (r *ArchivedFileRepository) Create(ctx context.Context, archivedFile *entity.ArchivedFile) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	_, err := queries.CreateArchivedFile(ctx, sqlcgen.CreateArchivedFileParams{
		ID:               archivedFile.ID,
		OriginalFileID:   archivedFile.OriginalFileID,
		OriginalFolderID: uuidToPgtype(archivedFile.OriginalFolderID),
		OriginalPath:     archivedFile.OriginalPath,
		Name:             archivedFile.Name.String(),
		MimeType:         archivedFile.MimeType.String(),
		Size:             archivedFile.Size,
		OwnerID:          archivedFile.OwnerID,
		OwnerType:        sqlcgen.OwnerType(archivedFile.OwnerType),
		StorageKey:       archivedFile.StorageKey.String(),
		ArchivedAt:       archivedFile.ArchivedAt,
		ArchivedBy:       archivedFile.ArchivedBy,
		ExpiresAt:        archivedFile.ExpiresAt,
	})

	return r.HandleError(err)
}

// FindByID はIDでアーカイブファイルを検索します
func (r *ArchivedFileRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.ArchivedFile, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetArchivedFileByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("archived file")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row), nil
}

// Delete はアーカイブファイルを削除します
func (r *ArchivedFileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteArchivedFile(ctx, id)
	return r.HandleError(err)
}

// FindByOwner はオーナーのアーカイブファイルを検索します
func (r *ArchivedFileRepository) FindByOwner(ctx context.Context, ownerID uuid.UUID, ownerType valueobject.OwnerType) ([]*entity.ArchivedFile, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListArchivedFilesByOwner(ctx, sqlcgen.ListArchivedFilesByOwnerParams{
		OwnerID:   ownerID,
		OwnerType: sqlcgen.OwnerType(ownerType),
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// FindExpired は期限切れのアーカイブファイルを検索します
func (r *ArchivedFileRepository) FindExpired(ctx context.Context) ([]*entity.ArchivedFile, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListExpiredArchivedFiles(ctx)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// FindByOriginalFileID は元ファイルIDでアーカイブファイルを検索します
func (r *ArchivedFileRepository) FindByOriginalFileID(ctx context.Context, originalFileID uuid.UUID) (*entity.ArchivedFile, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetArchivedFileByOriginalFileID(ctx, originalFileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("archived file")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row), nil
}

// toEntity はsqlcgen.ArchivedFileをentity.ArchivedFileに変換します
func (r *ArchivedFileRepository) toEntity(row sqlcgen.ArchivedFile) *entity.ArchivedFile {
	name, _ := valueobject.NewFileName(row.Name)
	mimeType, _ := valueobject.NewMimeType(row.MimeType)
	storageKey, _ := valueobject.NewStorageKeyFromString(row.StorageKey)
	ownerType := valueobject.OwnerType(row.OwnerType)

	return entity.ReconstructArchivedFile(
		row.ID,
		row.OriginalFileID,
		pgtypeToUUID(row.OriginalFolderID),
		row.OriginalPath,
		name,
		mimeType,
		row.Size,
		row.OwnerID,
		ownerType,
		storageKey,
		row.ArchivedAt,
		row.ArchivedBy,
		row.ExpiresAt,
	)
}

// toEntities はsqlcgen.ArchivedFile配列をentity.ArchivedFile配列に変換します
func (r *ArchivedFileRepository) toEntities(rows []sqlcgen.ArchivedFile) []*entity.ArchivedFile {
	entities := make([]*entity.ArchivedFile, len(rows))
	for i, row := range rows {
		entities[i] = r.toEntity(row)
	}
	return entities
}

// インターフェースの実装を保証
var _ repository.ArchivedFileRepository = (*ArchivedFileRepository)(nil)
