package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database/sqlcgen"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ShareLinkAccessRepository は共有リンクアクセスログリポジトリの実装です
type ShareLinkAccessRepository struct {
	*database.BaseRepository
}

// NewShareLinkAccessRepository は新しいShareLinkAccessRepositoryを作成します
func NewShareLinkAccessRepository(txManager *database.TxManager) *ShareLinkAccessRepository {
	return &ShareLinkAccessRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create はアクセスログを作成します
func (r *ShareLinkAccessRepository) Create(ctx context.Context, access *entity.ShareLinkAccess) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	var userID pgtype.UUID
	if access.UserID != nil {
		userID = pgtype.UUID{Bytes: *access.UserID, Valid: true}
	}

	var ipAddress, userAgent *string
	if access.IPAddress != "" {
		ipAddress = &access.IPAddress
	}
	if access.UserAgent != "" {
		userAgent = &access.UserAgent
	}

	_, err := queries.CreateShareLinkAccess(ctx, sqlcgen.CreateShareLinkAccessParams{
		ID:          access.ID,
		ShareLinkID: access.ShareLinkID,
		AccessedAt:  access.AccessedAt,
		IpAddress:   ipAddress,
		UserAgent:   userAgent,
		UserID:      userID,
		Action:      access.Action.String(),
	})

	return r.HandleError(err)
}

// FindByID はIDでアクセスログを検索します
func (r *ShareLinkAccessRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.ShareLinkAccess, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetShareLinkAccessByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("share_link_access")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row)
}

// FindByShareLinkID は共有リンクIDでアクセスログを検索します
func (r *ShareLinkAccessRepository) FindByShareLinkID(ctx context.Context, shareLinkID uuid.UUID) ([]*entity.ShareLinkAccess, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListShareLinkAccessesByLinkID(ctx, shareLinkID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// FindByShareLinkIDWithPagination は共有リンクIDでアクセスログをページングで検索します
func (r *ShareLinkAccessRepository) FindByShareLinkIDWithPagination(ctx context.Context, shareLinkID uuid.UUID, limit, offset int) ([]*entity.ShareLinkAccess, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListShareLinkAccessesByLinkIDWithPagination(ctx, sqlcgen.ListShareLinkAccessesByLinkIDWithPaginationParams{
		ShareLinkID: shareLinkID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// CountByShareLinkID は共有リンクIDでアクセスログをカウントします
func (r *ShareLinkAccessRepository) CountByShareLinkID(ctx context.Context, shareLinkID uuid.UUID) (int, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	count, err := queries.CountShareLinkAccessesByLinkID(ctx, shareLinkID)
	if err != nil {
		return 0, r.HandleError(err)
	}

	return int(count), nil
}

// DeleteByShareLinkID は共有リンクIDでアクセスログを一括削除します
func (r *ShareLinkAccessRepository) DeleteByShareLinkID(ctx context.Context, shareLinkID uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteShareLinkAccessesByLinkID(ctx, shareLinkID)
	return r.HandleError(err)
}

// toEntity はsqlcgen.ShareLinkAccessをentity.ShareLinkAccessに変換します
func (r *ShareLinkAccessRepository) toEntity(row sqlcgen.ShareLinkAccess) (*entity.ShareLinkAccess, error) {
	var userID *uuid.UUID
	if row.UserID.Valid {
		id := uuid.UUID(row.UserID.Bytes)
		userID = &id
	}

	var ipAddress, userAgent string
	if row.IpAddress != nil {
		ipAddress = *row.IpAddress
	}
	if row.UserAgent != nil {
		userAgent = *row.UserAgent
	}

	action := entity.AccessAction(row.Action)
	if !action.IsValid() {
		return nil, entity.ErrInvalidAccessAction
	}

	return entity.ReconstructShareLinkAccess(
		row.ID,
		row.ShareLinkID,
		row.AccessedAt,
		ipAddress,
		userAgent,
		userID,
		action,
	), nil
}

// toEntities は複数のsqlcgen.ShareLinkAccessをentity.ShareLinkAccessに変換します
func (r *ShareLinkAccessRepository) toEntities(rows []sqlcgen.ShareLinkAccess) ([]*entity.ShareLinkAccess, error) {
	accesses := make([]*entity.ShareLinkAccess, 0, len(rows))
	for _, row := range rows {
		access, err := r.toEntity(row)
		if err != nil {
			return nil, err
		}
		accesses = append(accesses, access)
	}
	return accesses, nil
}

// インターフェースの実装を保証
var _ repository.ShareLinkAccessRepository = (*ShareLinkAccessRepository)(nil)
