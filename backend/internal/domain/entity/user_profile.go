package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// UserProfileSettings はユーザー設定を定義します
type UserProfileSettings struct {
	NotificationsEnabled bool `json:"notifications_enabled"`
	EmailNotifications   bool `json:"email_notifications"`
	Theme                string `json:"theme,omitempty"`
}

// UserProfile はユーザープロファイルエンティティを定義します
type UserProfile struct {
	UserID      uuid.UUID
	DisplayName string
	AvatarURL   string
	Bio         string
	Locale      string
	Timezone    string
	Settings    UserProfileSettings
	UpdatedAt   time.Time
}

// NewUserProfile は新しいUserProfileを作成します
func NewUserProfile(userID uuid.UUID) *UserProfile {
	return &UserProfile{
		UserID:   userID,
		Locale:   "ja",
		Timezone: "Asia/Tokyo",
		Settings: UserProfileSettings{
			NotificationsEnabled: true,
			EmailNotifications:   true,
			Theme:                "system",
		},
		UpdatedAt: time.Now(),
	}
}

// SettingsJSON はSettingsをJSON文字列として返します
func (p *UserProfile) SettingsJSON() ([]byte, error) {
	return json.Marshal(p.Settings)
}

// SetSettingsFromJSON はJSON文字列からSettingsを設定します
func (p *UserProfile) SetSettingsFromJSON(data []byte) error {
	return json.Unmarshal(data, &p.Settings)
}

// ValidateBio はbioの長さを検証します
func (p *UserProfile) ValidateBio() bool {
	return len([]rune(p.Bio)) <= 500
}
