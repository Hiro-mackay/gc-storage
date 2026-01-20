package valueobject

import (
	"errors"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	FileNameMaxBytes = 255
)

var (
	ErrFileNameEmpty          = errors.New("file name cannot be empty")
	ErrFileNameTooLong        = errors.New("file name too long")
	ErrFileNameForbiddenChars = errors.New("file name contains forbidden characters")
	ErrFileNameReserved       = errors.New("file name is reserved")
)

// forbiddenFileChars はファイル名に使用できない文字
var forbiddenFileChars = []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}

// FileName はファイル名を表す値オブジェクト
type FileName struct {
	value string
}

// NewFileName は文字列からFileNameを生成します
func NewFileName(name string) (FileName, error) {
	// 前後の空白をトリム
	trimmed := strings.TrimSpace(name)

	// 空文字チェック
	if trimmed == "" {
		return FileName{}, ErrFileNameEmpty
	}

	// 予約名チェック
	if trimmed == "." || trimmed == ".." {
		return FileName{}, ErrFileNameReserved
	}

	// 長さチェック（UTF-8バイト数）
	if utf8.RuneCountInString(trimmed) > FileNameMaxBytes {
		return FileName{}, ErrFileNameTooLong
	}

	// 禁止文字チェック
	for _, char := range forbiddenFileChars {
		if strings.Contains(trimmed, char) {
			return FileName{}, ErrFileNameForbiddenChars
		}
	}

	return FileName{value: trimmed}, nil
}

// Value は値を返します
func (fn FileName) Value() string {
	return fn.value
}

// String は文字列を返します（Stringerインターフェース）
func (fn FileName) String() string {
	return fn.value
}

// IsEmpty は空かどうかを判定します
func (fn FileName) IsEmpty() bool {
	return fn.value == ""
}

// Equals は等価性を判定します
func (fn FileName) Equals(other FileName) bool {
	return fn.value == other.value
}

// Extension は拡張子を返します（ドット付き）
func (fn FileName) Extension() string {
	return filepath.Ext(fn.value)
}

// BaseName は拡張子を除いたファイル名を返します
func (fn FileName) BaseName() string {
	ext := filepath.Ext(fn.value)
	return strings.TrimSuffix(fn.value, ext)
}
