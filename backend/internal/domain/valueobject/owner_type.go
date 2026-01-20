package valueobject

import (
	"errors"
	"strings"
)

// OwnerType は所有者タイプを表す値オブジェクト
type OwnerType string

const (
	OwnerTypeUser  OwnerType = "user"
	OwnerTypeGroup OwnerType = "group"
)

var (
	ErrInvalidOwnerType = errors.New("invalid owner type")
)

// NewOwnerType は文字列からOwnerTypeを生成します
func NewOwnerType(s string) (OwnerType, error) {
	ot := OwnerType(strings.ToLower(s))
	if !ot.IsValid() {
		return "", ErrInvalidOwnerType
	}
	return ot, nil
}

// IsValid はOwnerTypeが有効かどうかを判定します
func (ot OwnerType) IsValid() bool {
	return ot == OwnerTypeUser || ot == OwnerTypeGroup
}

// IsUser はユーザー所有かどうかを判定します
func (ot OwnerType) IsUser() bool {
	return ot == OwnerTypeUser
}

// IsGroup はグループ所有かどうかを判定します
func (ot OwnerType) IsGroup() bool {
	return ot == OwnerTypeGroup
}

// String は文字列を返します
func (ot OwnerType) String() string {
	return string(ot)
}
