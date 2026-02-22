package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ErrorResponse はエラーレスポンス構造を定義します
type ErrorResponse struct {
	Error ErrorBody   `json:"error"`
	Meta  interface{} `json:"meta"`
}

// ErrorBody はエラー本体を定義します
type ErrorBody struct {
	Code    string                `json:"code"`
	Message string                `json:"message"`
	Details []apperror.FieldError `json:"details,omitempty"`
}

// CustomHTTPErrorHandler はカスタムエラーハンドラーです
func CustomHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		// AppErrorの場合
		response := ErrorResponse{
			Error: ErrorBody{
				Code:    string(appErr.Code),
				Message: appErr.Message,
				Details: appErr.Details,
			},
		}

		// 内部エラーの場合はログ出力
		if appErr.HTTPStatus >= 500 {
			slog.Error("internal error",
				"request_id", GetRequestID(c),
				"error", appErr.Error(),
			)
		}

		_ = c.JSON(appErr.HTTPStatus, response)
		return
	}

	// Echo HTTPErrorの場合
	var he *echo.HTTPError
	if errors.As(err, &he) {
		response := ErrorResponse{
			Error: ErrorBody{
				Code:    http.StatusText(he.Code),
				Message: fmt.Sprintf("%v", he.Message),
			},
		}

		_ = c.JSON(he.Code, response)
		return
	}

	// 未知のエラー
	slog.Error("unknown error",
		"request_id", GetRequestID(c),
		"error", err.Error(),
	)

	response := ErrorResponse{
		Error: ErrorBody{
			Code:    "INTERNAL_ERROR",
			Message: "internal server error",
		},
	}

	_ = c.JSON(http.StatusInternalServerError, response)
}
