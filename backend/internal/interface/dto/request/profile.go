package request

// UpdateProfileRequest はプロファイル更新リクエスト
type UpdateProfileRequest struct {
	AvatarURL               *string                         `json:"avatar_url" validate:"omitempty,url"`
	Bio                     *string                         `json:"bio" validate:"omitempty,max=500"`
	Locale                  *string                         `json:"locale" validate:"omitempty,max=10"`
	Timezone                *string                         `json:"timezone" validate:"omitempty,max=50"`
	Theme                   *string                         `json:"theme" validate:"omitempty,oneof=system light dark"`
	NotificationPreferences *NotificationPreferencesRequest `json:"notification_preferences" validate:"omitempty"`
}

// NotificationPreferencesRequest は通知設定リクエスト
type NotificationPreferencesRequest struct {
	EmailEnabled *bool `json:"email_enabled"`
	PushEnabled  *bool `json:"push_enabled"`
}
