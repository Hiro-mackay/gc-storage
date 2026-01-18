package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	HeaderRequestID     = "X-Request-ID"
	ContextKeyRequestID = "request_id"
)

// RequestID はリクエストIDを生成・設定するミドルウェアを返します
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestID := c.Request().Header.Get(HeaderRequestID)
			if requestID == "" {
				requestID = uuid.New().String()
			}

			c.Set(ContextKeyRequestID, requestID)
			c.Response().Header().Set(HeaderRequestID, requestID)

			return next(c)
		}
	}
}

// GetRequestID はコンテキストからリクエストIDを取得します
func GetRequestID(c echo.Context) string {
	if id, ok := c.Get(ContextKeyRequestID).(string); ok {
		return id
	}
	return ""
}
