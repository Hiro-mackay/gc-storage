package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/request"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/response"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/presenter"
	authcmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/command"
	authqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// AuthHandler は認証関連のHTTPハンドラーです
type AuthHandler struct {
	// Commands
	registerCommand     *authcmd.RegisterCommand
	loginCommand        *authcmd.LoginCommand
	refreshTokenCommand *authcmd.RefreshTokenCommand
	logoutCommand       *authcmd.LogoutCommand

	// Queries
	getUserQuery *authqry.GetUserQuery
}

// NewAuthHandler は新しいAuthHandlerを作成します
func NewAuthHandler(
	registerCommand *authcmd.RegisterCommand,
	loginCommand *authcmd.LoginCommand,
	refreshTokenCommand *authcmd.RefreshTokenCommand,
	logoutCommand *authcmd.LogoutCommand,
	getUserQuery *authqry.GetUserQuery,
) *AuthHandler {
	return &AuthHandler{
		registerCommand:     registerCommand,
		loginCommand:        loginCommand,
		refreshTokenCommand: refreshTokenCommand,
		logoutCommand:       logoutCommand,
		getUserQuery:        getUserQuery,
	}
}

// Register はユーザー登録を処理します
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c echo.Context) error {
	var req request.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	output, err := h.registerCommand.Execute(c.Request().Context(), authcmd.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		return err
	}

	return presenter.Created(c, response.RegisterResponse{
		UserID:  output.UserID.String(),
		Message: "Registration successful. Please check your email to verify your account.",
	})
}

// Login はログインを処理します
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c echo.Context) error {
	var req request.LoginRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	output, err := h.loginCommand.Execute(c.Request().Context(), authcmd.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: c.Request().UserAgent(),
		IPAddress: c.RealIP(),
	})
	if err != nil {
		return err
	}

	// リフレッシュトークンをHttpOnly Cookieに設定
	h.setRefreshTokenCookie(c, output.RefreshToken)

	return presenter.OK(c, response.LoginResponse{
		AccessToken: output.AccessToken,
		ExpiresIn:   output.ExpiresIn,
		User:        response.ToUserResponse(output.User),
	})
}

// Refresh はトークンリフレッシュを処理します
// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c echo.Context) error {
	// Cookieからリフレッシュトークンを取得
	cookie, err := c.Cookie("refresh_token")
	if err != nil {
		return apperror.NewUnauthorizedError("refresh token not found")
	}

	output, err := h.refreshTokenCommand.Execute(c.Request().Context(), authcmd.RefreshTokenInput{
		RefreshToken: cookie.Value,
	})
	if err != nil {
		return err
	}

	// 新しいリフレッシュトークンをCookieに設定
	h.setRefreshTokenCookie(c, output.RefreshToken)

	return presenter.OK(c, response.RefreshResponse{
		AccessToken: output.AccessToken,
		ExpiresIn:   output.ExpiresIn,
	})
}

// Logout はログアウトを処理します
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c echo.Context) error {
	sessionID := middleware.GetSessionID(c)
	accessClaims := middleware.GetAccessClaims(c)

	if err := h.logoutCommand.Execute(c.Request().Context(), sessionID, accessClaims); err != nil {
		// エラーでも成功扱い
	}

	// Cookieを削除
	h.clearRefreshTokenCookie(c)

	return presenter.OK(c, map[string]string{"message": "logged out successfully"})
}

// Me は現在のユーザー情報を取得します
// GET /api/v1/me
func (h *AuthHandler) Me(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	output, err := h.getUserQuery.Execute(c.Request().Context(), authqry.GetUserInput{
		UserID: uuid.MustParse(claims.UserID.String()),
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToUserResponse(output.User))
}

func (h *AuthHandler) setRefreshTokenCookie(c echo.Context, token string) {
	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7日
	})
}

func (h *AuthHandler) clearRefreshTokenCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}
