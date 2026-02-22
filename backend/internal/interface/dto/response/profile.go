package response

import (
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// ProfileDataResponse はプロファイルデータのみのレスポンス
type ProfileDataResponse struct {
	ID                      string                     `json:"id"`
	UserID                  string                     `json:"user_id"`
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

// GetProfileResponse はプロファイル取得レスポンス（profile + user の入れ子構造）
type GetProfileResponse struct {
	Profile *ProfileDataResponse `json:"profile"`
	User    *UserResponse        `json:"user"`
}

// UpdateProfileResponse はプロファイル更新レスポンス（profile のみ）
type UpdateProfileResponse struct {
	Profile *ProfileDataResponse `json:"profile"`
}

// ToProfileDataResponse はエンティティをProfileDataResponseに変換します
func ToProfileDataResponse(profile *entity.UserProfile) *ProfileDataResponse {
	if profile == nil {
		return nil
	}
	return &ProfileDataResponse{
		ID:        profile.ID.String(),
		UserID:    profile.UserID.String(),
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

// ToGetProfileResponse はエンティティをGetProfileResponseに変換します
func ToGetProfileResponse(user *entity.User, profile *entity.UserProfile) *GetProfileResponse {
	if user == nil || profile == nil {
		return nil
	}
	return &GetProfileResponse{
		Profile: ToProfileDataResponse(profile),
		User:    ToUserResponse(user),
	}
}

// ToUpdateProfileResponse はエンティティをUpdateProfileResponseに変換します
func ToUpdateProfileResponse(profile *entity.UserProfile) *UpdateProfileResponse {
	if profile == nil {
		return nil
	}
	return &UpdateProfileResponse{
		Profile: ToProfileDataResponse(profile),
	}
}
