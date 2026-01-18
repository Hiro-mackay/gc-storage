package middleware

import (
	"fmt"
	"log/slog"
	"runtime"

	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// Recover はパニックをリカバーするミドルウェアを返します
func Recover() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					// スタックトレースを取得
					buf := make([]byte, 4096)
					n := runtime.Stack(buf, false)
					stackTrace := string(buf[:n])

					slog.Error("panic recovered",
						"request_id", GetRequestID(c),
						"error", fmt.Sprintf("%v", r),
						"stack", stackTrace,
					)

					// 500エラーを返す
					c.Error(apperror.NewInternalError(fmt.Errorf("internal server error")))
				}
			}()

			return next(c)
		}
	}
}
