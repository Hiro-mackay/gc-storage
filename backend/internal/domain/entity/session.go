package entity

import (
	"time"

	"github.com/google/uuid"
)

// MaxActiveSessionsPerUser はユーザーあたりの最大アクティブセッション数 (R-SS002)
const MaxActiveSessionsPerUser = 10

// Session はセッションエンティティを定義します
type Session struct {
	ID           string
	UserID       uuid.UUID
	RefreshToken string
	UserAgent    string
	IPAddress    string
	ExpiresAt    time.Time
	CreatedAt    time.Time
	LastUsedAt   time.Time
}

// IsExpired はセッションが期限切れかを判定します
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsValid はセッションが有効かを判定します
func (s *Session) IsValid() bool {
	return !s.IsExpired()
}

// UpdateLastUsed は最終使用日時を更新します
func (s *Session) UpdateLastUsed() {
	s.LastUsedAt = time.Now()
}
