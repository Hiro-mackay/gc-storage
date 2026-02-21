package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// AuditService は監査ログを記録するサービスインターフェースです
type AuditService interface {
	// Log は監査ログを非同期で記録します
	Log(ctx context.Context, entry AuditEntry)
}

// AuditEntry は監査ログの記録に必要な情報を定義します
type AuditEntry struct {
	UserID       *uuid.UUID
	Action       entity.AuditAction
	ResourceType entity.AuditResourceType
	ResourceID   *uuid.UUID
	Details      map[string]interface{}
	IPAddress    string
	UserAgent    string
	RequestID    string
}
