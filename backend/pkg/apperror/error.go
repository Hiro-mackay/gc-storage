package apperror

import (
	"fmt"
	"net/http"
)

// ErrorCode はエラーコードを表します
type ErrorCode string

const (
	CodeValidationError    ErrorCode = "VALIDATION_ERROR"
	CodeInvalidRequest     ErrorCode = "INVALID_REQUEST"
	CodeUnauthorized       ErrorCode = "UNAUTHORIZED"
	CodeTokenExpired       ErrorCode = "TOKEN_EXPIRED"
	CodeForbidden          ErrorCode = "FORBIDDEN"
	CodeQuotaExceeded      ErrorCode = "QUOTA_EXCEEDED"
	CodeNotFound           ErrorCode = "NOT_FOUND"
	CodeConflict           ErrorCode = "CONFLICT"
	CodeRateLimitExceeded  ErrorCode = "RATE_LIMIT_EXCEEDED"
	CodeInternalError      ErrorCode = "INTERNAL_ERROR"
	CodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

// AppError はアプリケーションエラーを表します
type AppError struct {
	Code       ErrorCode    `json:"code"`
	Message    string       `json:"message"`
	Details    []FieldError `json:"details,omitempty"`
	HTTPStatus int          `json:"-"`
	Err        error        `json:"-"`
}

// FieldError はフィールドエラーを表します
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error はerrorインターフェースを実装します
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap は元のエラーを返します
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewValidationError はバリデーションエラーを作成します
func NewValidationError(message string, details []FieldError) *AppError {
	return &AppError{
		Code:       CodeValidationError,
		Message:    message,
		Details:    details,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewInvalidRequestError は不正リクエストエラーを作成します
func NewInvalidRequestError(message string) *AppError {
	return &AppError{
		Code:       CodeInvalidRequest,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewUnauthorizedError は認証エラーを作成します
func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Code:       CodeUnauthorized,
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
	}
}

// NewTokenExpiredError はトークン期限切れエラーを作成します
func NewTokenExpiredError() *AppError {
	return &AppError{
		Code:       CodeTokenExpired,
		Message:    "token has expired",
		HTTPStatus: http.StatusUnauthorized,
	}
}

// NewForbiddenError は権限エラーを作成します
func NewForbiddenError(message string) *AppError {
	return &AppError{
		Code:       CodeForbidden,
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}
}

// NewQuotaExceededError はクォータ超過エラーを作成します
func NewQuotaExceededError(message string) *AppError {
	return &AppError{
		Code:       CodeQuotaExceeded,
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}
}

// NewNotFoundError はリソース不在エラーを作成します
func NewNotFoundError(resource string) *AppError {
	return &AppError{
		Code:       CodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		HTTPStatus: http.StatusNotFound,
	}
}

// NewConflictError は競合エラーを作成します
func NewConflictError(message string) *AppError {
	return &AppError{
		Code:       CodeConflict,
		Message:    message,
		HTTPStatus: http.StatusConflict,
	}
}

// NewTooManyRequestsError はレート制限エラーを作成します
func NewTooManyRequestsError(message string) *AppError {
	return &AppError{
		Code:       CodeRateLimitExceeded,
		Message:    message,
		HTTPStatus: http.StatusTooManyRequests,
	}
}

// NewInternalError は内部エラーを作成します
func NewInternalError(err error) *AppError {
	return &AppError{
		Code:       CodeInternalError,
		Message:    "internal server error",
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

// NewServiceUnavailableError はサービス利用不可エラーを作成します
func NewServiceUnavailableError(message string) *AppError {
	return &AppError{
		Code:       CodeServiceUnavailable,
		Message:    message,
		HTTPStatus: http.StatusServiceUnavailable,
	}
}

// HasCode はエラーが特定のコードかどうかを判定します
func (e *AppError) HasCode(code ErrorCode) bool {
	return e.Code == code
}

// IsNotFound はリソース不在エラーかどうかを判定します
func IsNotFound(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == CodeNotFound
	}
	return false
}

// IsUnauthorized は認証エラーかどうかを判定します
func IsUnauthorized(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == CodeUnauthorized || appErr.Code == CodeTokenExpired
	}
	return false
}

// IsForbidden は権限エラーかどうかを判定します
func IsForbidden(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == CodeForbidden
	}
	return false
}
