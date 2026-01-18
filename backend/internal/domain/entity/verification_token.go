package entity

import (
	"time"

	"github.com/google/uuid"
)

// VerificationToken はメール確認トークンエンティティを定義します
type VerificationToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// IsExpired はトークンが期限切れかを判定します
func (v *VerificationToken) IsExpired() bool {
	return time.Now().After(v.ExpiresAt)
}

// IsValid はトークンが有効かを判定します
func (v *VerificationToken) IsValid() bool {
	return !v.IsExpired()
}

// PasswordResetToken はパスワードリセットトークンエンティティを定義します
type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
	UsedAt    *time.Time
}

// IsExpired はトークンが期限切れかを判定します
func (p *PasswordResetToken) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}

// IsUsed はトークンが使用済みかを判定します
func (p *PasswordResetToken) IsUsed() bool {
	return p.UsedAt != nil
}

// IsValid はトークンが有効かを判定します
func (p *PasswordResetToken) IsValid() bool {
	return !p.IsExpired() && !p.IsUsed()
}
