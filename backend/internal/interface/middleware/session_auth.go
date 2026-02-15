package middleware

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// SessionAuthMiddleware はセッションベース認証ミドルウェアを提供します
type SessionAuthMiddleware struct {
	sessionRepo repository.SessionRepository
	userRepo    repository.UserRepository
}

// NewSessionAuthMiddleware は新しいSessionAuthMiddlewareを作成します
func NewSessionAuthMiddleware(
	sessionRepo repository.SessionRepository,
	userRepo repository.UserRepository,
) *SessionAuthMiddleware {
	return &SessionAuthMiddleware{
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
	}
}

// Authenticate は認証ミドルウェアを返します
func (m *SessionAuthMiddleware) Authenticate() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 1. Cookieからセッション IDを取得
			cookie, err := c.Cookie("session_id")
			if err != nil {
				return apperror.NewUnauthorizedError("session not found")
			}

			// 2. Redisからセッションを取得
			session, err := m.sessionRepo.FindByID(c.Request().Context(), cookie.Value)
			if err != nil {
				return apperror.NewUnauthorizedError("invalid session")
			}

			// 3. セッションの有効期限をチェック
			if session.IsExpired() {
				m.sessionRepo.Delete(c.Request().Context(), session.ID)
				return apperror.NewUnauthorizedError("session expired")
			}

			// 4. ユーザーの状態をチェック
			user, err := m.userRepo.FindByID(c.Request().Context(), session.UserID)
			if err != nil {
				return apperror.NewUnauthorizedError("user not found")
			}

			if user.Status != entity.UserStatusActive {
				return apperror.NewUnauthorizedError("account is not active")
			}

			// 5. セッションをリフレッシュ（スライディングウィンドウ）
			session.Refresh()
			if err := m.sessionRepo.Save(c.Request().Context(), session); err != nil {
				slog.Warn("failed to refresh session", "session_id", session.ID, "error", err)
			}

			// 6. コンテキストにユーザー情報を設定
			c.Set(ContextKeyUserID, user.ID.String())
			c.Set(ContextKeySessionID, session.ID)
			c.Set(ContextKeyUser, user)

			// リクエストコンテキストにも設定（UseCase層で使用）
			ctx := c.Request().Context()
			ctx = context.WithValue(ctx, ContextKeyUserID, user.ID.String())
			ctx = context.WithValue(ctx, ContextKeySessionID, session.ID)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// OptionalAuth はオプショナル認証ミドルウェアを返します
// セッションがあれば検証し、なくてもエラーにしない
func (m *SessionAuthMiddleware) OptionalAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie("session_id")
			if err != nil {
				return next(c)
			}

			session, err := m.sessionRepo.FindByID(c.Request().Context(), cookie.Value)
			if err != nil {
				return next(c)
			}

			if session.IsExpired() {
				return next(c)
			}

			user, err := m.userRepo.FindByID(c.Request().Context(), session.UserID)
			if err != nil {
				return next(c)
			}

			if user.Status != entity.UserStatusActive {
				return next(c)
			}

			// セッションをリフレッシュ
			session.Refresh()
			if err := m.sessionRepo.Save(c.Request().Context(), session); err != nil {
				slog.Warn("failed to refresh session (optional auth)", "session_id", session.ID, "error", err)
			}

			c.Set(ContextKeyUserID, user.ID.String())
			c.Set(ContextKeySessionID, session.ID)
			c.Set(ContextKeyUser, user)

			ctx := c.Request().Context()
			ctx = context.WithValue(ctx, ContextKeyUserID, user.ID.String())
			ctx = context.WithValue(ctx, ContextKeySessionID, session.ID)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}
