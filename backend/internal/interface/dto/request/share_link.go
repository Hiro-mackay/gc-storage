package request

// CreateShareLinkRequest は共有リンク作成リクエストです
type CreateShareLinkRequest struct {
	Permission     string  `json:"permission" validate:"required,oneof=read write"`
	Password       *string `json:"password"`
	ExpiresAt      *string `json:"expiresAt"` // RFC3339 format
	MaxAccessCount *int    `json:"maxAccessCount" validate:"omitempty,min=1"`
}

// UpdateShareLinkRequest は共有リンク更新リクエストです
type UpdateShareLinkRequest struct {
	Password       *string `json:"password"`
	ExpiresAt      *string `json:"expiresAt"` // RFC3339 format
	MaxAccessCount *int    `json:"maxAccessCount" validate:"omitempty,min=1"`
}

// AccessShareLinkRequest は共有リンクアクセスリクエストです
type AccessShareLinkRequest struct {
	Password string `json:"password"`
}
