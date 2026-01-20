package valueobject

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const (
	FolderNameMaxBytes = 255
)

var (
	ErrFolderNameEmpty          = errors.New("folder name cannot be empty")
	ErrFolderNameTooLong        = errors.New("folder name too long")
	ErrFolderNameForbiddenChars = errors.New("folder name contains forbidden characters")
	ErrFolderNameReserved       = errors.New("folder name is reserved")
)

// forbiddenFolderChars はフォルダ名に使用できない文字
var forbiddenFolderChars = []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}

// FolderName はフォルダ名を表す値オブジェクト
type FolderName struct {
	value string
}

// NewFolderName は文字列からFolderNameを生成します
func NewFolderName(name string) (FolderName, error) {
	// 前後の空白をトリム
	trimmed := strings.TrimSpace(name)

	// 空文字チェック
	if trimmed == "" {
		return FolderName{}, ErrFolderNameEmpty
	}

	// 予約名チェック
	if trimmed == "." || trimmed == ".." {
		return FolderName{}, ErrFolderNameReserved
	}

	// 長さチェック（UTF-8バイト数）
	if utf8.RuneCountInString(trimmed) > FolderNameMaxBytes {
		return FolderName{}, ErrFolderNameTooLong
	}

	// 禁止文字チェック
	for _, char := range forbiddenFolderChars {
		if strings.Contains(trimmed, char) {
			return FolderName{}, ErrFolderNameForbiddenChars
		}
	}

	return FolderName{value: trimmed}, nil
}

// Value は値を返します
func (fn FolderName) Value() string {
	return fn.value
}

// String は文字列を返します（Stringerインターフェース）
func (fn FolderName) String() string {
	return fn.value
}

// IsEmpty は空かどうかを判定します
func (fn FolderName) IsEmpty() bool {
	return fn.value == ""
}

// Equals は等価性を判定します
func (fn FolderName) Equals(other FolderName) bool {
	return fn.value == other.value
}
