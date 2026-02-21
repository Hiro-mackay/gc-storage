package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database/sqlcgen"
)

// AuditLogRepository は監査ログリポジトリの実装です
type AuditLogRepository struct {
	*database.BaseRepository
}

// NewAuditLogRepository は新しいAuditLogRepositoryを作成します
func NewAuditLogRepository(txManager *database.TxManager) *AuditLogRepository {
	return &AuditLogRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create は監査ログを作成します
func (r *AuditLogRepository) Create(ctx context.Context, log *entity.AuditLog) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	var userID pgtype.UUID
	if log.UserID != nil {
		userID = pgtype.UUID{Bytes: *log.UserID, Valid: true}
	}

	var resourceID pgtype.UUID
	if log.ResourceID != nil {
		resourceID = pgtype.UUID{Bytes: *log.ResourceID, Valid: true}
	}

	var details []byte
	if log.Details != nil {
		var err error
		details, err = json.Marshal(log.Details)
		if err != nil {
			return err
		}
	}

	var ipAddress *string
	if log.IPAddress != "" {
		ipAddress = &log.IPAddress
	}

	var userAgent *string
	if log.UserAgent != "" {
		userAgent = &log.UserAgent
	}

	var requestID *string
	if log.RequestID != "" {
		requestID = &log.RequestID
	}

	_, err := queries.CreateAuditLog(ctx, sqlcgen.CreateAuditLogParams{
		UserID:       userID,
		Action:       string(log.Action),
		ResourceType: string(log.ResourceType),
		ResourceID:   resourceID,
		Details:      details,
		IpAddress:    ipAddress,
		UserAgent:    userAgent,
		RequestID:    requestID,
	})

	return r.HandleError(err)
}

// ListByUserID はユーザーIDで監査ログを取得します
func (r *AuditLogRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.AuditLog, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListAuditLogsByUserID(ctx, sqlcgen.ListAuditLogsByUserIDParams{
		UserID:    pgtype.UUID{Bytes: userID, Valid: true},
		LimitVal:  int32(limit),
		OffsetVal: int32(offset),
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// ListByResource はリソースタイプとIDで監査ログを取得します
func (r *AuditLogRepository) ListByResource(ctx context.Context, resourceType entity.AuditResourceType, resourceID uuid.UUID, limit, offset int) ([]*entity.AuditLog, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListAuditLogsByResource(ctx, sqlcgen.ListAuditLogsByResourceParams{
		ResourceType: string(resourceType),
		ResourceID:   pgtype.UUID{Bytes: resourceID, Valid: true},
		LimitVal:     int32(limit),
		OffsetVal:    int32(offset),
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// CountByUserID はユーザーIDで監査ログ数を取得します
func (r *AuditLogRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	count, err := queries.CountAuditLogsByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return 0, r.HandleError(err)
	}

	return int(count), nil
}

// toEntities はsqlcgen.AuditLogのスライスをentity.AuditLogのスライスに変換します
func (r *AuditLogRepository) toEntities(rows []sqlcgen.AuditLog) []*entity.AuditLog {
	entities := make([]*entity.AuditLog, len(rows))
	for i, row := range rows {
		entities[i] = r.toEntity(row)
	}
	return entities
}

// toEntity はsqlcgen.AuditLogをentity.AuditLogに変換します
func (r *AuditLogRepository) toEntity(row sqlcgen.AuditLog) *entity.AuditLog {
	var userID *uuid.UUID
	if row.UserID.Valid {
		id := uuid.UUID(row.UserID.Bytes)
		userID = &id
	}

	var resourceID *uuid.UUID
	if row.ResourceID.Valid {
		id := uuid.UUID(row.ResourceID.Bytes)
		resourceID = &id
	}

	var details map[string]interface{}
	if row.Details != nil {
		_ = json.Unmarshal(row.Details, &details)
	}

	var ipAddress string
	if row.IpAddress != nil {
		ipAddress = *row.IpAddress
	}

	var userAgent string
	if row.UserAgent != nil {
		userAgent = *row.UserAgent
	}

	var requestID string
	if row.RequestID != nil {
		requestID = *row.RequestID
	}

	return &entity.AuditLog{
		ID:           row.ID,
		UserID:       userID,
		Action:       entity.AuditAction(row.Action),
		ResourceType: entity.AuditResourceType(row.ResourceType),
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		RequestID:    requestID,
		CreatedAt:    row.CreatedAt,
	}
}

// インターフェースの実装を保証
var _ repository.AuditLogRepository = (*AuditLogRepository)(nil)
