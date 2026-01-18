package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/pkg/jwt"
)

const (
	ContextKeyUserID       = "user_id"
	ContextKeySessionID    = "session_id"
	ContextKeyAccessClaims = "access_claims"
)

// GetUserID はコンテキストからユーザーIDを取得します
func GetUserID(c echo.Context) string {
	if id, ok := c.Get(ContextKeyUserID).(string); ok {
		return id
	}
	return ""
}

// GetUserUUID はコンテキストからユーザーIDをUUIDとして取得します
func GetUserUUID(c echo.Context) (uuid.UUID, error) {
	userID := GetUserID(c)
	if userID == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(userID)
}

// GetSessionID はコンテキストからセッションIDを取得します
func GetSessionID(c echo.Context) string {
	if id, ok := c.Get(ContextKeySessionID).(string); ok {
		return id
	}
	return ""
}

// SetUserID はコンテキストにユーザーIDを設定します
func SetUserID(c echo.Context, userID string) {
	c.Set(ContextKeyUserID, userID)
}

// SetSessionID はコンテキストにセッションIDを設定します
func SetSessionID(c echo.Context, sessionID string) {
	c.Set(ContextKeySessionID, sessionID)
}

// GetAccessClaims はコンテキストからアクセストークンクレームを取得します
func GetAccessClaims(c echo.Context) *jwt.AccessTokenClaims {
	if claims, ok := c.Get(ContextKeyAccessClaims).(*jwt.AccessTokenClaims); ok {
		return claims
	}
	return nil
}

// SetAccessClaims はコンテキストにアクセストークンクレームを設定します
func SetAccessClaims(c echo.Context, claims *jwt.AccessTokenClaims) {
	c.Set(ContextKeyAccessClaims, claims)
}
