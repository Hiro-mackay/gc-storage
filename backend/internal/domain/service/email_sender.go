package service

import "context"

// EmailSender はメール送信サービスのインターフェースを定義します
type EmailSender interface {
	// SendEmailVerification はメール確認用のメールを送信します
	SendEmailVerification(ctx context.Context, to, userName, verifyURL string) error

	// SendPasswordReset はパスワードリセット用のメールを送信します
	SendPasswordReset(ctx context.Context, to, userName, resetURL string) error
}
