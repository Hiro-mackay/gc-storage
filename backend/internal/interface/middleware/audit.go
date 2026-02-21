package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
)

const ContextKeyAuditService = "audit_service"

// AuditMiddleware は監査ログサービスをコンテキストに注入するミドルウェアです
type AuditMiddleware struct {
	auditService service.AuditService
}

// NewAuditMiddleware は新しいAuditMiddlewareを作成します
func NewAuditMiddleware(auditService service.AuditService) *AuditMiddleware {
	return &AuditMiddleware{auditService: auditService}
}

// Inject はAuditServiceをEchoコンテキストに注入するミドルウェアを返します
func (m *AuditMiddleware) Inject() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(ContextKeyAuditService, m.auditService)
			return next(c)
		}
	}
}

// GetAuditService はEchoコンテキストからAuditServiceを取得します
func GetAuditService(c echo.Context) service.AuditService {
	if svc, ok := c.Get(ContextKeyAuditService).(service.AuditService); ok {
		return svc
	}
	return nil
}

// AuditHelper は監査ログ記録のヘルパー関数です
func AuditHelper(c echo.Context, action string, resourceType string, resourceID *uuid.UUID, details map[string]interface{}) {
	svc := GetAuditService(c)
	if svc == nil {
		return
	}

	var userID *uuid.UUID
	if uid, err := GetUserUUID(c); err == nil && uid != uuid.Nil {
		userID = &uid
	}

	svc.Log(c.Request().Context(), service.AuditEntry{
		UserID:       userID,
		Action:       entity.AuditAction(action),
		ResourceType: entity.AuditResourceType(resourceType),
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    c.RealIP(),
		UserAgent:    c.Request().UserAgent(),
		RequestID:    GetRequestID(c),
	})
}
