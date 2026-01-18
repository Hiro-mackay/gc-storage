package response

import (
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// ProfileResponse はプロファイル情報レスポンス
type ProfileResponse struct {
	UserID      string                    `json:"user_id"`
	Email       string                    `json:"email"`
	Name        string                    `json:"name"`
	DisplayName string                    `json:"display_name,omitempty"`
	AvatarURL   string                    `json:"avatar_url,omitempty"`
	Bio         string                    `json:"bio,omitempty"`
	Locale      string                    `json:"locale"`
	Timezone    string                    `json:"timezone"`
	Settings    *ProfileSettingsResponse  `json:"settings"`
	UpdatedAt   time.Time                 `json:"updated_at"`
}

// ProfileSettingsResponse はプロファイル設定レスポンス
type ProfileSettingsResponse struct {
	NotificationsEnabled bool   `json:"notifications_enabled"`
	EmailNotifications   bool   `json:"email_notifications"`
	Theme                string `json:"theme"`
}

// ToProfileResponse はエンティティをレスポンスに変換します
func ToProfileResponse(user *entity.User, profile *entity.UserProfile) *ProfileResponse {
	if user == nil || profile == nil {
		return nil
	}
	return &ProfileResponse{
		UserID:      user.ID.String(),
		Email:       user.Email.String(),
		Name:        user.Name,
		DisplayName: profile.DisplayName,
		AvatarURL:   profile.AvatarURL,
		Bio:         profile.Bio,
		Locale:      profile.Locale,
		Timezone:    profile.Timezone,
		Settings: &ProfileSettingsResponse{
			NotificationsEnabled: profile.Settings.NotificationsEnabled,
			EmailNotifications:   profile.Settings.EmailNotifications,
			Theme:                profile.Settings.Theme,
		},
		UpdatedAt: profile.UpdatedAt,
	}
}

// UpdateProfileResponse はプロファイル更新レスポンス
type UpdateProfileResponse struct {
	Profile *ProfileResponse `json:"profile"`
	Message string           `json:"message"`
}
