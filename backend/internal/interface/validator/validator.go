package validator

import (
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// CustomValidator はEcho用のカスタムバリデーターです
type CustomValidator struct {
	validator *validator.Validate
}

// NewCustomValidator は新しいCustomValidatorを作成します
func NewCustomValidator() *CustomValidator {
	v := validator.New()

	// カスタムバリデーション登録
	v.RegisterValidation("filename", validateFileName)
	v.RegisterValidation("foldername", validateFolderName)
	v.RegisterValidation("password", validatePassword)

	return &CustomValidator{validator: v}
}

// Validate はリクエストを検証します
func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return cv.formatValidationErrors(err)
	}
	return nil
}

// formatValidationErrors はバリデーションエラーをフォーマットします
func (cv *CustomValidator) formatValidationErrors(err error) error {
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return apperror.NewValidationError(err.Error(), nil)
	}

	details := make([]apperror.FieldError, 0, len(validationErrors))
	for _, e := range validationErrors {
		details = append(details, apperror.FieldError{
			Field:   toSnakeCase(e.Field()),
			Message: getValidationMessage(e),
		})
	}

	return apperror.NewValidationError("validation failed", details)
}

// validateFileName はファイル名のバリデーション
func validateFileName(fl validator.FieldLevel) bool {
	name := fl.Field().String()
	if name == "" {
		return false
	}

	// 禁止文字チェック: / \ : * ? " < > |
	invalidChars := regexp.MustCompile(`[/\\:*?"<>|]`)
	if invalidChars.MatchString(name) {
		return false
	}

	// 隠しファイルや特殊ファイル名のチェック
	if strings.HasPrefix(name, ".") || name == "." || name == ".." {
		return false
	}

	return len(name) <= 255
}

// validateFolderName はフォルダ名のバリデーション
func validateFolderName(fl validator.FieldLevel) bool {
	name := fl.Field().String()
	if name == "" {
		return false
	}

	// 禁止文字チェック
	invalidChars := regexp.MustCompile(`[/\\:*?"<>|]`)
	return !invalidChars.MatchString(name) && len(name) <= 255
}

// validatePassword はパスワードのバリデーション
func validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	if len(password) < 8 || len(password) > 256 {
		return false
	}

	// 英大文字、英小文字、数字のうち2種以上
	var hasUpper, hasLower, hasDigit bool
	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasDigit = true
		}
	}

	count := 0
	if hasUpper {
		count++
	}
	if hasLower {
		count++
	}
	if hasDigit {
		count++
	}

	return count >= 2
}

// getValidationMessage はバリデーションエラーメッセージを返します
func getValidationMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "this field is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "must be at least " + e.Param() + " characters"
	case "max":
		return "must be at most " + e.Param() + " characters"
	case "uuid":
		return "must be a valid UUID"
	case "filename":
		return "must be a valid file name (no special characters)"
	case "foldername":
		return "must be a valid folder name (no special characters)"
	case "password":
		return "must be 8-256 characters with at least 2 of: uppercase, lowercase, digit"
	case "oneof":
		return "must be one of: " + e.Param()
	case "url":
		return "must be a valid URL"
	case "gte":
		return "must be greater than or equal to " + e.Param()
	case "lte":
		return "must be less than or equal to " + e.Param()
	default:
		return "validation failed"
	}
}

// toSnakeCase はPascalCase/camelCaseをsnake_caseに変換します
func toSnakeCase(str string) string {
	var result []rune
	for i, r := range str {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}
