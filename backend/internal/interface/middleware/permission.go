package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// PermissionMiddleware は権限チェックミドルウェアです
type PermissionMiddleware struct {
	resolver authz.PermissionResolver
}

// NewPermissionMiddleware は新しいPermissionMiddlewareを作成します
func NewPermissionMiddleware(resolver authz.PermissionResolver) *PermissionMiddleware {
	return &PermissionMiddleware{
		resolver: resolver,
	}
}

// RequirePermission は指定された権限を持っているか確認します
func (m *PermissionMiddleware) RequirePermission(resourceType authz.ResourceType, permission authz.Permission, resourceIDParam string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetAccessClaims(c)
			if claims == nil {
				return apperror.NewUnauthorizedError("invalid token")
			}

			resourceID, err := uuid.Parse(c.Param(resourceIDParam))
			if err != nil {
				return apperror.NewValidationError("invalid resource ID", nil)
			}

			hasPermission, err := m.resolver.HasPermission(c.Request().Context(), claims.UserID, resourceType, resourceID, permission)
			if err != nil {
				return err
			}

			if !hasPermission {
				return apperror.NewForbiddenError("you do not have permission to access this resource")
			}

			return next(c)
		}
	}
}

// RequireAnyPermission はいずれかの権限を持っているか確認します
func (m *PermissionMiddleware) RequireAnyPermission(resourceType authz.ResourceType, permissions []authz.Permission, resourceIDParam string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetAccessClaims(c)
			if claims == nil {
				return apperror.NewUnauthorizedError("invalid token")
			}

			resourceID, err := uuid.Parse(c.Param(resourceIDParam))
			if err != nil {
				return apperror.NewValidationError("invalid resource ID", nil)
			}

			for _, permission := range permissions {
				hasPermission, err := m.resolver.HasPermission(c.Request().Context(), claims.UserID, resourceType, resourceID, permission)
				if err != nil {
					return err
				}
				if hasPermission {
					return next(c)
				}
			}

			return apperror.NewForbiddenError("you do not have permission to access this resource")
		}
	}
}

// RequireOwner はオーナーであることを確認します
func (m *PermissionMiddleware) RequireOwner(resourceType authz.ResourceType, resourceIDParam string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetAccessClaims(c)
			if claims == nil {
				return apperror.NewUnauthorizedError("invalid token")
			}

			resourceID, err := uuid.Parse(c.Param(resourceIDParam))
			if err != nil {
				return apperror.NewValidationError("invalid resource ID", nil)
			}

			isOwner, err := m.resolver.IsOwner(c.Request().Context(), claims.UserID, resourceType, resourceID)
			if err != nil {
				return err
			}

			if !isOwner {
				return apperror.NewForbiddenError("only the owner can perform this action")
			}

			return next(c)
		}
	}
}

// RequireRole は指定されたロール以上を持っているか確認します
func (m *PermissionMiddleware) RequireRole(resourceType authz.ResourceType, minRole authz.Role, resourceIDParam string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetAccessClaims(c)
			if claims == nil {
				return apperror.NewUnauthorizedError("invalid token")
			}

			resourceID, err := uuid.Parse(c.Param(resourceIDParam))
			if err != nil {
				return apperror.NewValidationError("invalid resource ID", nil)
			}

			effectiveRole, err := m.resolver.GetEffectiveRole(c.Request().Context(), claims.UserID, resourceType, resourceID)
			if err != nil {
				return err
			}

			if !effectiveRole.Includes(minRole) {
				return apperror.NewForbiddenError("you do not have sufficient permissions")
			}

			return next(c)
		}
	}
}
