package valueobject

import "errors"

var (
	ErrInvalidShareLinkStatus = errors.New("invalid share link status")
)

// ShareLinkStatus は共有リンクの状態を表す値オブジェクト
type ShareLinkStatus string

const (
	ShareLinkStatusActive  ShareLinkStatus = "active"
	ShareLinkStatusRevoked ShareLinkStatus = "revoked"
	ShareLinkStatusExpired ShareLinkStatus = "expired"
)

// NewShareLinkStatus は文字列からShareLinkStatusを生成します
func NewShareLinkStatus(status string) (ShareLinkStatus, error) {
	s := ShareLinkStatus(status)
	if !s.IsValid() {
		return "", ErrInvalidShareLinkStatus
	}
	return s, nil
}

// IsValid は状態が有効かを判定します
func (s ShareLinkStatus) IsValid() bool {
	switch s {
	case ShareLinkStatusActive, ShareLinkStatusRevoked, ShareLinkStatusExpired:
		return true
	default:
		return false
	}
}

// String は文字列を返します
func (s ShareLinkStatus) String() string {
	return string(s)
}

// IsActive はアクティブ状態かを判定します
func (s ShareLinkStatus) IsActive() bool {
	return s == ShareLinkStatusActive
}

// IsRevoked は無効化済みかを判定します
func (s ShareLinkStatus) IsRevoked() bool {
	return s == ShareLinkStatusRevoked
}

// IsExpired は期限切れかを判定します
func (s ShareLinkStatus) IsExpired() bool {
	return s == ShareLinkStatusExpired
}

// CanAccess はアクセス可能かを判定します
func (s ShareLinkStatus) CanAccess() bool {
	return s == ShareLinkStatusActive
}
