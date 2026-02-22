package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

const (
	// CSRFCookieName はCSRFトークンのCookie名です
	CSRFCookieName = "csrf_token"
	// CSRFHeaderName はCSRFトークンのヘッダー名です
	CSRFHeaderName = "X-CSRF-Token"
	// csrfTokenBytes はCSRFトークンのバイト数です（32バイト = 256ビット）
	csrfTokenBytes = 32
)

// GenerateCSRFToken は暗号学的に安全なCSRFトークンを生成します
func GenerateCSRFToken() (string, error) {
	b := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SecureCookies はCookieのSecureフラグを制御するグローバル設定です
// ローカル開発（HTTP）では false、本番（HTTPS）では true に設定します
var SecureCookies = false

// SetCSRFCookie はCSRFトークンCookieを設定します（JavaScriptから読み取り可能）
func SetCSRFCookie(c echo.Context, token string) {
	c.SetCookie(&http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // JSから読み取り可能にする（double-submit cookie pattern）
		Secure:   SecureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60, // 7日
	})
}

// ClearCSRFCookie はCSRFトークンCookieを削除します
func ClearCSRFCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     CSRFCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		Secure:   SecureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// CSRF はCSRF保護ミドルウェアを返します（double-submit cookie pattern）
// セッションCookieを持つリクエストの状態変更メソッド（POST, PUT, PATCH, DELETE）に対して
// CSRFトークンの検証を行います
func CSRF() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 安全なメソッドはスキップ
			method := strings.ToUpper(c.Request().Method)
			if method == "GET" || method == "HEAD" || method == "OPTIONS" {
				return next(c)
			}

			// セッションCookieがない場合はスキップ（公開エンドポイント）
			if _, err := c.Cookie("session_id"); err != nil {
				return next(c)
			}

			// CSRFトークンをCookieから取得
			csrfCookie, err := c.Cookie(CSRFCookieName)
			if err != nil || csrfCookie.Value == "" {
				return apperror.NewForbiddenError("CSRF token missing")
			}

			// CSRFトークンをヘッダーから取得
			headerToken := c.Request().Header.Get(CSRFHeaderName)
			if headerToken == "" {
				return apperror.NewForbiddenError("CSRF token header missing")
			}

			// 定数時間比較でトークンを検証
			if subtle.ConstantTimeCompare([]byte(csrfCookie.Value), []byte(headerToken)) != 1 {
				return apperror.NewForbiddenError("CSRF token mismatch")
			}

			return next(c)
		}
	}
}
