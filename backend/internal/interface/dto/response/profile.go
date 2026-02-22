package response

import (
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// ProfileResponse はプロファイル情報レスポンス
type ProfileResponse struct {
	UserID                  string                     `json:"user_id"`
	Email                   string                     `json:"email"`
	Name                    string                     `json:"name"`
	AvatarURL               string                     `json:"avatar_url,omitempty"`
	Bio                     string                     `json:"bio,omitempty"`
	Locale                  string                     `json:"locale"`
	Timezone                string                     `json:"timezone"`
	Theme                   string                     `json:"theme"`
	NotificationPreferences *NotificationPrefsResponse `json:"notification_preferences"`
	UpdatedAt               time.Time                  `json:"updated_at"`
}

// NotificationPrefsResponse は通知設定レスポンス
type NotificationPrefsResponse struct {
	EmailEnabled bool `json:"email_enabled"`
	PushEnabled  bool `json:"push_enabled"`
}

// ToProfileResponse はエンティティをレスポンスに変換します
func ToProfileResponse(user *entity.User, profile *entity.UserProfile) *ProfileResponse {
	if user == nil || profile == nil {
		return nil
	}
	return &ProfileResponse{
		UserID:    user.ID.String(),
		Email:     user.Email.String(),
		Name:      user.Name,
		AvatarURL: profile.AvatarURL,
		Bio:       profile.Bio,
		Locale:    profile.Locale,
		Timezone:  profile.Timezone,
		Theme:     profile.Theme,
		NotificationPreferences: &NotificationPrefsResponse{
			EmailEnabled: profile.NotificationPreferences.EmailEnabled,
			PushEnabled:  profile.NotificationPreferences.PushEnabled,
		},
		UpdatedAt: profile.UpdatedAt,
	}
}

// UpdateProfileResponse はプロファイル更新レスポンス
type UpdateProfileResponse struct {
	Profile *ProfileResponse `json:"profile"`
	Message string           `json:"message"`
}
