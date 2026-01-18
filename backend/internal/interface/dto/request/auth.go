package request

// RegisterRequest はユーザー登録リクエスト
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,password"`
	Name     string `json:"name" validate:"required,min=1,max=100"`
}

// LoginRequest はログインリクエスト
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// ResendEmailVerificationRequest は確認メール再送リクエスト
type ResendEmailVerificationRequest struct {
	Email string `json:"email" validate:"required,email"`
}
