package entity

import (
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// UserStatus はユーザーの状態を定義します
type UserStatus string

const (
	UserStatusPending     UserStatus = "pending"
	UserStatusActive      UserStatus = "active"
	UserStatusSuspended   UserStatus = "suspended"
	UserStatusDeactivated UserStatus = "deactivated"
)

// IsValid は状態が有効かを判定します
func (s UserStatus) IsValid() bool {
	switch s {
	case UserStatusPending, UserStatusActive, UserStatusSuspended, UserStatusDeactivated:
		return true
	default:
		return false
	}
}

// User はユーザーエンティティを定義します
type User struct {
	ID            uuid.UUID
	Email         valueobject.Email
	Name          string
	PasswordHash  string
	Status        UserStatus
	EmailVerified bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// IsActive はユーザーがアクティブかを判定します
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// IsPending はユーザーが確認待ちかを判定します
func (u *User) IsPending() bool {
	return u.Status == UserStatusPending
}

// CanLogin はユーザーがログイン可能かを判定します
func (u *User) CanLogin() bool {
	return u.Status == UserStatusActive && u.EmailVerified
}

// HasPassword はユーザーがパスワードを持っているかを判定します
func (u *User) HasPassword() bool {
	return u.PasswordHash != ""
}
