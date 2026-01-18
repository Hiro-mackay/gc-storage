package middleware

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

// Logger はリクエストロギングミドルウェアを返します
func Logger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			latency := time.Since(start)

			// 構造化ログ出力
			slog.Info("request",
				"request_id", GetRequestID(c),
				"method", c.Request().Method,
				"uri", c.Request().RequestURI,
				"status", c.Response().Status,
				"latency_ms", latency.Milliseconds(),
				"ip", c.RealIP(),
				"user_agent", c.Request().UserAgent(),
				"bytes_in", c.Request().ContentLength,
				"bytes_out", c.Response().Size,
			)

			return err
		}
	}
}
