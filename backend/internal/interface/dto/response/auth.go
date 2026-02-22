package response

import (
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// LoginResponse はログイン・登録共通レスポンス
type LoginResponse struct {
	User *UserResponse `json:"user"`
}

// UserResponse はユーザー情報レスポンス
// Note: avatar_urlはProfileResponseから取得してください
type UserResponse struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
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

// ForgotPasswordResponse はパスワードリセットリクエストレスポンス
type ForgotPasswordResponse struct {
	Message string `json:"message"`
}

// ResetPasswordResponse はパスワードリセット実行レスポンス
type ResetPasswordResponse struct {
	Message string `json:"message"`
}

// ChangePasswordResponse はパスワード変更レスポンス
type ChangePasswordResponse struct {
	Message string `json:"message"`
}

// OAuthLoginResponse はOAuthログインレスポンス
type OAuthLoginResponse struct {
	User      *UserResponse `json:"user"`
	IsNewUser bool          `json:"is_new_user"`
}

// SetPasswordResponse はパスワード設定レスポンス
type SetPasswordResponse struct {
	Message string `json:"message"`
}

// LogoutResponse はログアウトレスポンス
type LogoutResponse struct {
	Message string `json:"message"`
}
