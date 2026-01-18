package request

// UpdateProfileRequest はプロファイル更新リクエスト
type UpdateProfileRequest struct {
	DisplayName *string                  `json:"display_name" validate:"omitempty,max=100"`
	AvatarURL   *string                  `json:"avatar_url" validate:"omitempty,url"`
	Bio         *string                  `json:"bio" validate:"omitempty,max=500"`
	Locale      *string                  `json:"locale" validate:"omitempty,max=10"`
	Timezone    *string                  `json:"timezone" validate:"omitempty,max=50"`
	Settings    *ProfileSettingsRequest  `json:"settings" validate:"omitempty"`
}

// ProfileSettingsRequest はプロファイル設定リクエスト
type ProfileSettingsRequest struct {
	NotificationsEnabled *bool   `json:"notifications_enabled"`
	EmailNotifications   *bool   `json:"email_notifications"`
	Theme                *string `json:"theme" validate:"omitempty,oneof=system light dark"`
}
