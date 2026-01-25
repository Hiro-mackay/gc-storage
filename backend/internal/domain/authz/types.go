package authz

import "errors"

var (
	ErrInvalidResourceType = errors.New("invalid resource type")
	ErrInvalidGranteeType  = errors.New("invalid grantee type")
)

// ResourceType はリソースの種類を表す型
type ResourceType string

const (
	ResourceTypeFile   ResourceType = "file"
	ResourceTypeFolder ResourceType = "folder"
)

// NewResourceType は文字列からResourceTypeを生成します
func NewResourceType(t string) (ResourceType, error) {
	rt := ResourceType(t)
	if !rt.IsValid() {
		return "", ErrInvalidResourceType
	}
	return rt, nil
}

// IsValid はリソースタイプが有効かを判定します
func (t ResourceType) IsValid() bool {
	switch t {
	case ResourceTypeFile, ResourceTypeFolder:
		return true
	default:
		return false
	}
}

// String は文字列を返します
func (t ResourceType) String() string {
	return string(t)
}

// GranteeType は権限付与対象の種類を表す型
type GranteeType string

const (
	GranteeTypeUser  GranteeType = "user"
	GranteeTypeGroup GranteeType = "group"
)

// NewGranteeType は文字列からGranteeTypeを生成します
func NewGranteeType(t string) (GranteeType, error) {
	gt := GranteeType(t)
	if !gt.IsValid() {
		return "", ErrInvalidGranteeType
	}
	return gt, nil
}

// IsValid は付与対象タイプが有効かを判定します
func (t GranteeType) IsValid() bool {
	switch t {
	case GranteeTypeUser, GranteeTypeGroup:
		return true
	default:
		return false
	}
}

// String は文字列を返します
func (t GranteeType) String() string {
	return string(t)
}

// IsUser はユーザータイプかを判定します
func (t GranteeType) IsUser() bool {
	return t == GranteeTypeUser
}

// IsGroup はグループタイプかを判定します
func (t GranteeType) IsGroup() bool {
	return t == GranteeTypeGroup
}
