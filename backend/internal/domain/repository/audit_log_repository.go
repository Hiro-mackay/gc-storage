package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// AuditLogRepository は監査ログの永続化インターフェースです
type AuditLogRepository interface {
	// Create は監査ログを作成します
	Create(ctx context.Context, log *entity.AuditLog) error
	// ListByUserID はユーザーIDで監査ログを取得します
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.AuditLog, error)
	// ListByResource はリソースタイプとIDで監査ログを取得します
	ListByResource(ctx context.Context, resourceType entity.AuditResourceType, resourceID uuid.UUID, limit, offset int) ([]*entity.AuditLog, error)
	// CountByUserID はユーザーIDで監査ログ数を取得します
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
}
