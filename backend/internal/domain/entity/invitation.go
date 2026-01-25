package entity

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

const (
	InvitationTokenLength = 32  // 32 bytes = 43 chars base64 URL-safe
	InvitationExpiryHours = 168 // 7 days
)

var (
	ErrInvitationExpired    = errors.New("invitation has expired")
	ErrInvitationNotPending = errors.New("invitation is not pending")
	ErrInvitationCancelled  = errors.New("invitation has been cancelled")
)

// Invitation はグループ招待エンティティ
type Invitation struct {
	ID        uuid.UUID
	GroupID   uuid.UUID
	Email     valueobject.Email
	Token     string
	Role      valueobject.GroupRole
	InvitedBy uuid.UUID
	ExpiresAt time.Time
	Status    valueobject.InvitationStatus
	CreatedAt time.Time
}

// NewInvitation は新しい招待を作成します
func NewInvitation(
	groupID uuid.UUID,
	email valueobject.Email,
	role valueobject.GroupRole,
	invitedBy uuid.UUID,
) (*Invitation, error) {
	token, err := generateInvitationToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &Invitation{
		ID:        uuid.New(),
		GroupID:   groupID,
		Email:     email,
		Token:     token,
		Role:      role,
		InvitedBy: invitedBy,
		ExpiresAt: now.Add(time.Hour * InvitationExpiryHours),
		Status:    valueobject.InvitationStatusPending,
		CreatedAt: now,
	}, nil
}

// ReconstructInvitation はDBから招待を復元します
func ReconstructInvitation(
	id uuid.UUID,
	groupID uuid.UUID,
	email valueobject.Email,
	token string,
	role valueobject.GroupRole,
	invitedBy uuid.UUID,
	expiresAt time.Time,
	status valueobject.InvitationStatus,
	createdAt time.Time,
) *Invitation {
	return &Invitation{
		ID:        id,
		GroupID:   groupID,
		Email:     email,
		Token:     token,
		Role:      role,
		InvitedBy: invitedBy,
		ExpiresAt: expiresAt,
		Status:    status,
		CreatedAt: createdAt,
	}
}

// generateInvitationToken はランダムなトークンを生成します
func generateInvitationToken() (string, error) {
	bytes := make([]byte, InvitationTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes), nil
}

// IsPending は保留中かを判定します
func (i *Invitation) IsPending() bool {
	return i.Status.IsPending()
}

// IsExpired は期限切れかを判定します
func (i *Invitation) IsExpired() bool {
	return time.Now().After(i.ExpiresAt) || i.Status.IsExpired()
}

// CanRespond は応答可能かを判定します
func (i *Invitation) CanRespond() error {
	if !i.Status.IsPending() {
		return ErrInvitationNotPending
	}
	if i.IsExpired() {
		return ErrInvitationExpired
	}

	return nil
}

// Accept は招待を承諾します
func (i *Invitation) Accept() error {
	if err := i.CanRespond(); err != nil {
		return err
	}
	i.Status = valueobject.InvitationStatusAccepted
	return nil
}

// Decline は招待を拒否します
func (i *Invitation) Decline() error {
	if err := i.CanRespond(); err != nil {
		return err
	}
	i.Status = valueobject.InvitationStatusDeclined
	return nil
}

// Cancel は招待をキャンセルします
func (i *Invitation) Cancel() error {
	if !i.Status.IsPending() {
		return ErrInvitationNotPending
	}
	i.Status = valueobject.InvitationStatusCancelled
	return nil
}

// MarkExpired は期限切れにします
func (i *Invitation) MarkExpired() {
	if i.Status.IsPending() {
		i.Status = valueobject.InvitationStatusExpired
	}
}

// IsForEmail は指定メールアドレス向けの招待かを判定します
func (i *Invitation) IsForEmail(email valueobject.Email) bool {
	return i.Email.Equals(email)
}

// IsForGroup は指定グループの招待かを判定します
func (i *Invitation) IsForGroup(groupID uuid.UUID) bool {
	return i.GroupID == groupID
}

// WasInvitedBy は指定ユーザーが招待者かを判定します
func (i *Invitation) WasInvitedBy(userID uuid.UUID) bool {
	return i.InvitedBy == userID
}
