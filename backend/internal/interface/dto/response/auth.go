package response

import (
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// RegisterResponse はユーザー登録レスポンス
type RegisterResponse struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

// LoginResponse はログインレスポンス
type LoginResponse struct {
	AccessToken string        `json:"access_token"`
	ExpiresIn   int           `json:"expires_in"`
	User        *UserResponse `json:"user"`
}

// RefreshResponse はトークンリフレッシュレスポンス
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// UserResponse はユーザー情報レスポンス
type UserResponse struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	AvatarURL     string    `json:"avatar_url,omitempty"`
	Status        string    `json:"status"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ToUserResponse はエンティティをレスポンスに変換します
func ToUserResponse(user *entity.User) *UserResponse {
	if user == nil {
		return nil
	}
	return &UserResponse{
		ID:            user.ID.String(),
		Email:         user.Email.String(),
		Name:          user.Name,
		AvatarURL:     user.AvatarURL,
		Status:        string(user.Status),
		EmailVerified: user.EmailVerified,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
	}
}

// VerifyEmailResponse はメール確認レスポンス
type VerifyEmailResponse struct {
	Message string `json:"message"`
}

// ResendEmailVerificationResponse は確認メール再送レスポンス
type ResendEmailVerificationResponse struct {
	Message string `json:"message"`
}
