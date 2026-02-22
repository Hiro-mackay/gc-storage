package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// contextKey is a custom type for context.WithValue keys to avoid collisions
type contextKey string

const (
	ContextKeyUserID    = "user_id"
	ContextKeySessionID = "session_id"
	ContextKeyUser      = "user"

	// Typed keys for context.WithValue (prevents SA1029)
	ctxKeyUserID    contextKey = ContextKeyUserID
	ctxKeySessionID contextKey = ContextKeySessionID
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

// GetUser はコンテキストからユーザーを取得します
func GetUser(c echo.Context) *entity.User {
	if user, ok := c.Get(ContextKeyUser).(*entity.User); ok {
		return user
	}
	return nil
}

// SetUserID はコンテキストにユーザーIDを設定します
func SetUserID(c echo.Context, userID string) {
	c.Set(ContextKeyUserID, userID)
}

// SetSessionID はコンテキストにセッションIDを設定します
func SetSessionID(c echo.Context, sessionID string) {
	c.Set(ContextKeySessionID, sessionID)
}

// SetUser はコンテキストにユーザーを設定します
func SetUser(c echo.Context, user *entity.User) {
	c.Set(ContextKeyUser, user)
}

// AccessTokenClaims は後方互換性のためのクレーム構造体です
// 新しいコードでは GetUser() を使用してください
type AccessTokenClaims struct {
	UserID    uuid.UUID
	SessionID string
}

// GetAccessClaims は後方互換性のためにクレームを返します
// 新しいコードでは GetUser() を使用してください
func GetAccessClaims(c echo.Context) *AccessTokenClaims {
	user := GetUser(c)
	if user == nil {
		return nil
	}
	sessionID := GetSessionID(c)
	return &AccessTokenClaims{
		UserID:    user.ID,
		SessionID: sessionID,
	}
}
