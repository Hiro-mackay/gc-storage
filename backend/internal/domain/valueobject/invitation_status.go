package valueobject

import "errors"

var (
	ErrInvalidInvitationStatus = errors.New("invalid invitation status")
)

// InvitationStatus は招待の状態を表す値オブジェクト
type InvitationStatus string

const (
	InvitationStatusPending   InvitationStatus = "pending"
	InvitationStatusAccepted  InvitationStatus = "accepted"
	InvitationStatusDeclined  InvitationStatus = "declined"
	InvitationStatusExpired   InvitationStatus = "expired"
	InvitationStatusCancelled InvitationStatus = "cancelled"
)

// NewInvitationStatus は文字列からInvitationStatusを生成します
func NewInvitationStatus(status string) (InvitationStatus, error) {
	s := InvitationStatus(status)
	if !s.IsValid() {
		return "", ErrInvalidInvitationStatus
	}
	return s, nil
}

// IsValid は状態が有効かを判定します
func (s InvitationStatus) IsValid() bool {
	switch s {
	case InvitationStatusPending, InvitationStatusAccepted, InvitationStatusDeclined, InvitationStatusExpired, InvitationStatusCancelled:
		return true
	default:
		return false
	}
}

// String は文字列を返します
func (s InvitationStatus) String() string {
	return string(s)
}

// IsPending は保留中かを判定します
func (s InvitationStatus) IsPending() bool {
	return s == InvitationStatusPending
}

// IsAccepted は承諾済みかを判定します
func (s InvitationStatus) IsAccepted() bool {
	return s == InvitationStatusAccepted
}

// IsDeclined は拒否済みかを判定します
func (s InvitationStatus) IsDeclined() bool {
	return s == InvitationStatusDeclined
}

// IsExpired は期限切れかを判定します
func (s InvitationStatus) IsExpired() bool {
	return s == InvitationStatusExpired
}

// IsCancelled はキャンセル済みかを判定します
func (s InvitationStatus) IsCancelled() bool {
	return s == InvitationStatusCancelled
}

// CanRespond は応答可能かを判定します（Pendingのみ）
func (s InvitationStatus) CanRespond() bool {
	return s == InvitationStatusPending
}

// IsFinal は最終状態かを判定します
func (s InvitationStatus) IsFinal() bool {
	return s != InvitationStatusPending
}
