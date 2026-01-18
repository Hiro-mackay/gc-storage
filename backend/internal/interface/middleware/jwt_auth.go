package middleware

import (
	"context"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/cache"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/jwt"
)

// JWTAuthMiddleware はJWT認証ミドルウェアを提供します
type JWTAuthMiddleware struct {
	jwtService   *jwt.JWTService
	jwtBlacklist *cache.JWTBlacklist
}

// NewJWTAuthMiddleware は新しいJWTAuthMiddlewareを作成します
func NewJWTAuthMiddleware(jwtService *jwt.JWTService, jwtBlacklist *cache.JWTBlacklist) *JWTAuthMiddleware {
	return &JWTAuthMiddleware{
		jwtService:   jwtService,
		jwtBlacklist: jwtBlacklist,
	}
}

// Authenticate は認証ミドルウェアを返します
func (m *JWTAuthMiddleware) Authenticate() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Authorizationヘッダーを取得
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return apperror.NewUnauthorizedError("authorization header required")
			}

			// Bearer トークンを抽出
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return apperror.NewUnauthorizedError("invalid authorization header format")
			}

			token := parts[1]

			// トークンを検証
			claims, err := m.jwtService.ValidateAccessToken(token)
			if err != nil {
				return apperror.NewUnauthorizedError("invalid or expired token")
			}

			// ブラックリストチェック
			if m.jwtBlacklist != nil {
				isBlacklisted, err := m.jwtBlacklist.IsBlacklisted(c.Request().Context(), claims.ID)
				if err != nil {
					// エラーの場合はログを出力して続行
				} else if isBlacklisted {
					return apperror.NewUnauthorizedError("token has been revoked")
				}
			}

			// コンテキストにユーザー情報を設定
			c.Set(ContextKeyUserID, claims.UserID.String())
			c.Set(ContextKeySessionID, claims.SessionID)
			c.Set(ContextKeyAccessClaims, claims)

			// リクエストコンテキストにも設定（UseCase層で使用）
			ctx := c.Request().Context()
			ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID.String())
			ctx = context.WithValue(ctx, ContextKeySessionID, claims.SessionID)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// OptionalAuth はオプショナル認証ミドルウェアを返します
// トークンがあれば検証し、なくてもエラーにしない
func (m *JWTAuthMiddleware) OptionalAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return next(c)
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return next(c)
			}

			token := parts[1]
			claims, err := m.jwtService.ValidateAccessToken(token)
			if err != nil {
				return next(c)
			}

			// ブラックリストチェック
			if m.jwtBlacklist != nil {
				isBlacklisted, _ := m.jwtBlacklist.IsBlacklisted(c.Request().Context(), claims.ID)
				if isBlacklisted {
					return next(c)
				}
			}

			c.Set(ContextKeyUserID, claims.UserID.String())
			c.Set(ContextKeySessionID, claims.SessionID)

			ctx := c.Request().Context()
			ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID.String())
			ctx = context.WithValue(ctx, ContextKeySessionID, claims.SessionID)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}
