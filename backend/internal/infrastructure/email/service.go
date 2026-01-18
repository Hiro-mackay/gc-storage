package email

import (
	"context"
	"fmt"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
)

// EmailService はメール送信サービスを提供します
type EmailService struct {
	client *SMTPClient
}

// NewEmailService は新しいEmailServiceを作成します
func NewEmailService(client *SMTPClient) *EmailService {
	return &EmailService{
		client: client,
	}
}

// SendWelcome はウェルカムメールを送信します
func (s *EmailService) SendWelcome(ctx context.Context, to, userName string) error {
	data := DefaultTemplateData()
	data.UserName = userName

	body, err := RenderTemplate(TemplateWelcome, data)
	if err != nil {
		return fmt.Errorf("failed to render welcome template: %w", err)
	}

	return s.client.SendHTML([]string{to}, fmt.Sprintf("%sへようこそ", data.AppName), body)
}

// SendPasswordReset はパスワードリセットメールを送信します
func (s *EmailService) SendPasswordReset(ctx context.Context, to, userName, resetURL string) error {
	data := DefaultTemplateData()
	data.UserName = userName
	data.ActionURL = resetURL
	data.ActionText = "パスワードをリセット"
	data.ExpiresIn = "1時間"

	body, err := RenderTemplate(TemplatePasswordReset, data)
	if err != nil {
		return fmt.Errorf("failed to render password reset template: %w", err)
	}

	return s.client.SendHTML([]string{to}, "パスワードリセット", body)
}

// SendEmailVerification はメール確認メールを送信します
func (s *EmailService) SendEmailVerification(ctx context.Context, to, userName, verifyURL string) error {
	data := DefaultTemplateData()
	data.UserName = userName
	data.ActionURL = verifyURL
	data.ActionText = "メールアドレスを確認"
	data.ExpiresIn = "24時間"

	body, err := RenderTemplate(TemplateEmailVerify, data)
	if err != nil {
		return fmt.Errorf("failed to render email verify template: %w", err)
	}

	return s.client.SendHTML([]string{to}, "メールアドレスの確認", body)
}

// SendGroupInvitation はグループ招待メールを送信します
func (s *EmailService) SendGroupInvitation(ctx context.Context, to, userName, inviterName, groupName, inviteURL string) error {
	data := DefaultTemplateData()
	data.UserName = userName
	data.InviterName = inviterName
	data.GroupName = groupName
	data.ActionURL = inviteURL
	data.ActionText = "招待を確認"
	data.ExpiresIn = "7日間"

	body, err := RenderTemplate(TemplateGroupInvitation, data)
	if err != nil {
		return fmt.Errorf("failed to render group invitation template: %w", err)
	}

	return s.client.SendHTML([]string{to}, fmt.Sprintf("「%s」グループへの招待", groupName), body)
}

// SendShareNotification は共有通知メールを送信します
func (s *EmailService) SendShareNotification(ctx context.Context, to, userName, sharerName, fileName, shareURL string) error {
	data := DefaultTemplateData()
	data.UserName = userName
	data.SharerName = sharerName
	data.FileName = fileName
	data.ActionURL = shareURL
	data.ActionText = "ファイルを見る"

	body, err := RenderTemplate(TemplateShareNotify, data)
	if err != nil {
		return fmt.Errorf("failed to render share notify template: %w", err)
	}

	return s.client.SendHTML([]string{to}, fmt.Sprintf("%sさんからファイルが共有されました", sharerName), body)
}

// インターフェースの実装を保証
var _ service.EmailSender = (*EmailService)(nil)
