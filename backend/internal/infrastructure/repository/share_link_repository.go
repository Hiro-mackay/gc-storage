package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database/sqlcgen"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ShareLinkRepository は共有リンクリポジトリの実装です
type ShareLinkRepository struct {
	*database.BaseRepository
}

// NewShareLinkRepository は新しいShareLinkRepositoryを作成します
func NewShareLinkRepository(txManager *database.TxManager) *ShareLinkRepository {
	return &ShareLinkRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create は共有リンクを作成します
func (r *ShareLinkRepository) Create(ctx context.Context, link *entity.ShareLink) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	var expiresAt pgtype.Timestamptz
	if link.ExpiresAt != nil {
		expiresAt = pgtype.Timestamptz{Time: *link.ExpiresAt, Valid: true}
	}

	var maxAccessCount *int32
	if link.MaxAccessCount != nil {
		count := int32(*link.MaxAccessCount)
		maxAccessCount = &count
	}

	var passwordHash *string
	if link.PasswordHash != "" {
		passwordHash = &link.PasswordHash
	}

	_, err := queries.CreateShareLink(ctx, sqlcgen.CreateShareLinkParams{
		ID:             link.ID,
		Token:          link.Token.String(),
		ResourceType:   link.ResourceType.String(),
		ResourceID:     link.ResourceID,
		CreatedBy:      link.CreatedBy,
		Permission:     link.Permission.String(),
		PasswordHash:   passwordHash,
		ExpiresAt:      expiresAt,
		MaxAccessCount: maxAccessCount,
		AccessCount:    int32(link.AccessCount),
		Status:         link.Status.String(),
		CreatedAt:      link.CreatedAt,
		UpdatedAt:      link.UpdatedAt,
	})

	return r.HandleError(err)
}

// FindByID はIDで共有リンクを検索します
func (r *ShareLinkRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.ShareLink, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetShareLinkByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("share_link")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row)
}

// FindByToken はトークンで共有リンクを検索します
func (r *ShareLinkRepository) FindByToken(ctx context.Context, token valueobject.ShareToken) (*entity.ShareLink, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetShareLinkByToken(ctx, token.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("share_link")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row)
}

// Update は共有リンクを更新します
func (r *ShareLinkRepository) Update(ctx context.Context, link *entity.ShareLink) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	permission := link.Permission.String()
	status := link.Status.String()
	accessCount := int32(link.AccessCount)

	var expiresAt pgtype.Timestamptz
	if link.ExpiresAt != nil {
		expiresAt = pgtype.Timestamptz{Time: *link.ExpiresAt, Valid: true}
	}

	var maxAccessCount *int32
	if link.MaxAccessCount != nil {
		count := int32(*link.MaxAccessCount)
		maxAccessCount = &count
	}

	_, err := queries.UpdateShareLink(ctx, sqlcgen.UpdateShareLinkParams{
		ID:             link.ID,
		Permission:     &permission,
		PasswordHash:   &link.PasswordHash,
		ExpiresAt:      expiresAt,
		MaxAccessCount: maxAccessCount,
		AccessCount:    &accessCount,
		Status:         &status,
	})

	return r.HandleError(err)
}

// Delete は共有リンクを削除します
func (r *ShareLinkRepository) Delete(ctx context.Context, id uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteShareLink(ctx, id)
	return r.HandleError(err)
}

// FindByResource はリソースで共有リンクを検索します
func (r *ShareLinkRepository) FindByResource(ctx context.Context, resourceType authz.ResourceType, resourceID uuid.UUID) ([]*entity.ShareLink, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListShareLinksByResource(ctx, sqlcgen.ListShareLinksByResourceParams{
		ResourceType: resourceType.String(),
		ResourceID:   resourceID,
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// FindActiveByResource はリソースでアクティブな共有リンクを検索します
func (r *ShareLinkRepository) FindActiveByResource(ctx context.Context, resourceType authz.ResourceType, resourceID uuid.UUID) ([]*entity.ShareLink, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListActiveShareLinksByResource(ctx, sqlcgen.ListActiveShareLinksByResourceParams{
		ResourceType: resourceType.String(),
		ResourceID:   resourceID,
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// FindByCreator は作成者で共有リンクを検索します
func (r *ShareLinkRepository) FindByCreator(ctx context.Context, createdBy uuid.UUID) ([]*entity.ShareLink, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListShareLinksByCreator(ctx, createdBy)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// FindExpired は期限切れの共有リンクを検索します
func (r *ShareLinkRepository) FindExpired(ctx context.Context) ([]*entity.ShareLink, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListExpiredShareLinks(ctx)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// UpdateStatusBatch は一括でステータスを更新します
func (r *ShareLinkRepository) UpdateStatusBatch(ctx context.Context, ids []uuid.UUID, status valueobject.ShareLinkStatus) (int64, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	count, err := queries.UpdateShareLinksStatusBatch(ctx, sqlcgen.UpdateShareLinksStatusBatchParams{
		Column1: ids,
		Status:  status.String(),
	})
	if err != nil {
		return 0, r.HandleError(err)
	}

	return count, nil
}

// toEntity はsqlcgen.ShareLinkをentity.ShareLinkに変換します
func (r *ShareLinkRepository) toEntity(row sqlcgen.ShareLink) (*entity.ShareLink, error) {
	token, err := valueobject.ReconstructShareToken(row.Token)
	if err != nil {
		return nil, err
	}

	resourceType, err := authz.NewResourceType(row.ResourceType)
	if err != nil {
		return nil, err
	}

	permission, err := valueobject.NewSharePermission(row.Permission)
	if err != nil {
		return nil, err
	}

	status, err := valueobject.NewShareLinkStatus(row.Status)
	if err != nil {
		return nil, err
	}

	var expiresAt *time.Time
	if row.ExpiresAt.Valid {
		expiresAt = &row.ExpiresAt.Time
	}

	var maxAccessCount *int
	if row.MaxAccessCount != nil {
		count := int(*row.MaxAccessCount)
		maxAccessCount = &count
	}

	var passwordHash string
	if row.PasswordHash != nil {
		passwordHash = *row.PasswordHash
	}

	return entity.ReconstructShareLink(
		row.ID,
		token,
		resourceType,
		row.ResourceID,
		row.CreatedBy,
		permission,
		passwordHash,
		expiresAt,
		maxAccessCount,
		int(row.AccessCount),
		status,
		row.CreatedAt,
		row.UpdatedAt,
	), nil
}

// toEntities は複数のsqlcgen.ShareLinkをentity.ShareLinkに変換します
func (r *ShareLinkRepository) toEntities(rows []sqlcgen.ShareLink) ([]*entity.ShareLink, error) {
	links := make([]*entity.ShareLink, 0, len(rows))
	for _, row := range rows {
		link, err := r.toEntity(row)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}

// インターフェースの実装を保証
var _ repository.ShareLinkRepository = (*ShareLinkRepository)(nil)
