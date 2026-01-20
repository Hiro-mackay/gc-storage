package entity

import (
	"time"

	"github.com/google/uuid"
)

// NotificationPreferences はユーザーの通知設定を定義します
type NotificationPreferences struct {
	EmailEnabled bool `json:"email_enabled"`
	PushEnabled  bool `json:"push_enabled"`
}

// UserProfile はユーザープロファイルエンティティを定義します
// Note: display_name は users テーブルにあるため、ここには含まれない
type UserProfile struct {
	ID                      uuid.UUID
	UserID                  uuid.UUID
	AvatarURL               string
	Bio                     string
	Timezone                string
	Locale                  string
	Theme                   string // "system", "light", "dark"
	NotificationPreferences NotificationPreferences
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// NewUserProfile は新しいUserProfileを作成します
func NewUserProfile(userID uuid.UUID) *UserProfile {
	now := time.Now()
	return &UserProfile{
		ID:       uuid.New(),
		UserID:   userID,
		Locale:   "ja",
		Timezone: "Asia/Tokyo",
		Theme:    "system",
		NotificationPreferences: NotificationPreferences{
			EmailEnabled: true,
			PushEnabled:  true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ValidateBio はbioの長さを検証します
func (p *UserProfile) ValidateBio() bool {
	return len([]rune(p.Bio)) <= 500
}

// SetTheme はテーマを設定します
func (p *UserProfile) SetTheme(theme string) {
	p.Theme = theme
	p.UpdatedAt = time.Now()
}

// SetNotificationPreferences は通知設定を設定します
func (p *UserProfile) SetNotificationPreferences(prefs NotificationPreferences) {
	p.NotificationPreferences = prefs
	p.UpdatedAt = time.Now()
}
