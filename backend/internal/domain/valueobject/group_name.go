package valueobject

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const (
	GroupNameMinLength = 1
	GroupNameMaxLength = 100
)

var (
	ErrGroupNameEmpty   = errors.New("group name cannot be empty")
	ErrGroupNameTooLong = errors.New("group name must be at most 100 characters")
)

// GroupName はグループ名を表す値オブジェクト
type GroupName struct {
	value string
}

// NewGroupName は文字列からGroupNameを生成します
func NewGroupName(name string) (GroupName, error) {
	trimmed := strings.TrimSpace(name)

	if trimmed == "" {
		return GroupName{}, ErrGroupNameEmpty
	}

	if utf8.RuneCountInString(trimmed) > GroupNameMaxLength {
		return GroupName{}, ErrGroupNameTooLong
	}

	return GroupName{value: trimmed}, nil
}

// Value は値を返します
func (gn GroupName) Value() string {
	return gn.value
}

// String は文字列を返します
func (gn GroupName) String() string {
	return gn.value
}

// IsEmpty は空かどうかを判定します
func (gn GroupName) IsEmpty() bool {
	return gn.value == ""
}

// Equals は等価性を判定します
func (gn GroupName) Equals(other GroupName) bool {
	return gn.value == other.value
}
