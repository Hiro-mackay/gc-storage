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

// UploadSessionRepository はアップロードセッションリポジトリの実装です
type UploadSessionRepository struct {
	*database.BaseRepository
}

// NewUploadSessionRepository は新しいUploadSessionRepositoryを作成します
func NewUploadSessionRepository(txManager *database.TxManager) *UploadSessionRepository {
	return &UploadSessionRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create はアップロードセッションを作成します
func (r *UploadSessionRepository) Create(ctx context.Context, session *entity.UploadSession) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	_, err := queries.CreateUploadSession(ctx, sqlcgen.CreateUploadSessionParams{
		ID:            session.ID,
		FileID:        session.FileID,
		OwnerID:       session.OwnerID,
		CreatedBy:     session.CreatedBy,
		FolderID:      session.FolderID,
		FileName:      session.FileName.String(),
		MimeType:      session.MimeType.String(),
		TotalSize:     session.TotalSize,
		StorageKey:    session.StorageKey.String(),
		MinioUploadID: session.MinioUploadID,
		IsMultipart:   session.IsMultipart,
		TotalParts:    int32(session.TotalParts),
		UploadedParts: int32(session.UploadedParts),
		Status:        sqlcgen.UploadSessionStatus(session.Status),
		CreatedAt:     session.CreatedAt,
		UpdatedAt:     session.UpdatedAt,
		ExpiresAt:     session.ExpiresAt,
	})

	return r.HandleError(err)
}

// FindByID はIDでアップロードセッションを検索します
func (r *UploadSessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.UploadSession, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetUploadSessionByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("upload session")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row), nil
}

// Update はアップロードセッションを更新します
func (r *UploadSessionRepository) Update(ctx context.Context, session *entity.UploadSession) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	uploadedParts := int32(session.UploadedParts)
	_, err := queries.UpdateUploadSession(ctx, sqlcgen.UpdateUploadSessionParams{
		ID:            session.ID,
		MinioUploadID: session.MinioUploadID,
		UploadedParts: &uploadedParts,
		Status:        sqlcgen.NullUploadSessionStatus{UploadSessionStatus: sqlcgen.UploadSessionStatus(session.Status), Valid: true},
	})

	return r.HandleError(err)
}

// Delete はアップロードセッションを削除します
func (r *UploadSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteUploadSession(ctx, id)
	return r.HandleError(err)
}

// FindByFileID はファイルIDでアップロードセッションを検索します
func (r *UploadSessionRepository) FindByFileID(ctx context.Context, fileID uuid.UUID) (*entity.UploadSession, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetUploadSessionByFileID(ctx, fileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("upload session")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row), nil
}

// FindByStorageKey はストレージキーでアップロードセッションを検索します
func (r *UploadSessionRepository) FindByStorageKey(ctx context.Context, storageKey valueobject.StorageKey) (*entity.UploadSession, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetUploadSessionByStorageKey(ctx, storageKey.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("upload session")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row), nil
}

// FindExpired は期限切れのアップロードセッションを検索します
func (r *UploadSessionRepository) FindExpired(ctx context.Context) ([]*entity.UploadSession, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListExpiredUploadSessions(ctx)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// UpdateStatus はステータスのみを更新します
func (r *UploadSessionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.UploadSessionStatus) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.UpdateUploadSessionStatus(ctx, sqlcgen.UpdateUploadSessionStatusParams{
		ID:     id,
		Status: sqlcgen.UploadSessionStatus(status),
	})

	return r.HandleError(err)
}

// toEntity はsqlcgen.UploadSessionをentity.UploadSessionに変換します
func (r *UploadSessionRepository) toEntity(row sqlcgen.UploadSession) *entity.UploadSession {
	fileName, _ := valueobject.NewFileName(row.FileName)
	mimeType, _ := valueobject.NewMimeType(row.MimeType)
	storageKey, _ := valueobject.NewStorageKeyFromString(row.StorageKey)

	return entity.ReconstructUploadSession(
		row.ID,
		row.FileID,
		row.OwnerID,
		row.CreatedBy,
		row.FolderID,
		fileName,
		mimeType,
		row.TotalSize,
		storageKey,
		row.MinioUploadID,
		row.IsMultipart,
		int(row.TotalParts),
		int(row.UploadedParts),
		entity.UploadSessionStatus(row.Status),
		row.CreatedAt,
		row.UpdatedAt,
		row.ExpiresAt,
	)
}

// toEntities はsqlcgen.UploadSession配列をentity.UploadSession配列に変換します
func (r *UploadSessionRepository) toEntities(rows []sqlcgen.UploadSession) []*entity.UploadSession {
	entities := make([]*entity.UploadSession, len(rows))
	for i, row := range rows {
		entities[i] = r.toEntity(row)
	}
	return entities
}

// インターフェースの実装を保証
var _ repository.UploadSessionRepository = (*UploadSessionRepository)(nil)
