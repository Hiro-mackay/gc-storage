package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database/sqlcgen"
)

// UploadPartRepository はアップロードパーツリポジトリの実装です
type UploadPartRepository struct {
	*database.BaseRepository
}

// NewUploadPartRepository は新しいUploadPartRepositoryを作成します
func NewUploadPartRepository(txManager *database.TxManager) *UploadPartRepository {
	return &UploadPartRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create はアップロードパーツを作成します
func (r *UploadPartRepository) Create(ctx context.Context, part *entity.UploadPart) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	_, err := queries.CreateUploadPart(ctx, sqlcgen.CreateUploadPartParams{
		ID:         part.ID,
		SessionID:  part.SessionID,
		PartNumber: int32(part.PartNumber),
		Size:       part.Size,
		Etag:       part.ETag,
		UploadedAt: part.UploadedAt,
	})

	return r.HandleError(err)
}

// FindBySessionID はセッションIDでアップロードパーツを検索します
func (r *UploadPartRepository) FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*entity.UploadPart, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListUploadPartsBySessionID(ctx, sessionID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// DeleteBySessionID はセッションIDで全パーツを削除します
func (r *UploadPartRepository) DeleteBySessionID(ctx context.Context, sessionID uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteUploadPartsBySessionID(ctx, sessionID)
	return r.HandleError(err)
}

// toEntity はsqlcgen.UploadPartをentity.UploadPartに変換します
func (r *UploadPartRepository) toEntity(row sqlcgen.UploadPart) *entity.UploadPart {
	return entity.ReconstructUploadPart(
		row.ID,
		row.SessionID,
		int(row.PartNumber),
		row.Size,
		row.Etag,
		row.UploadedAt,
	)
}

// toEntities はsqlcgen.UploadPart配列をentity.UploadPart配列に変換します
func (r *UploadPartRepository) toEntities(rows []sqlcgen.UploadPart) []*entity.UploadPart {
	entities := make([]*entity.UploadPart, len(rows))
	for i, row := range rows {
		entities[i] = r.toEntity(row)
	}
	return entities
}

// インターフェースの実装を保証
var _ repository.UploadPartRepository = (*UploadPartRepository)(nil)
