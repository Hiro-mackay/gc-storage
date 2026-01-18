package email

import (
	"bytes"
	"fmt"
	"html/template"
)

// TemplateType はメールテンプレートの種類を定義します
type TemplateType string

const (
	TemplateWelcome         TemplateType = "welcome"
	TemplatePasswordReset   TemplateType = "password_reset"
	TemplateEmailVerify     TemplateType = "email_verify"
	TemplateGroupInvitation TemplateType = "group_invitation"
	TemplateShareNotify     TemplateType = "share_notify"
)

// TemplateData はテンプレートデータを定義します
type TemplateData struct {
	AppName     string
	AppURL      string
	UserName    string
	ActionURL   string
	ActionText  string
	ExpiresIn   string
	GroupName   string
	InviterName string
	FileName    string
	SharerName  string
}

// DefaultTemplateData はデフォルトのテンプレートデータを返します
func DefaultTemplateData() TemplateData {
	return TemplateData{
		AppName: "GC Storage",
		AppURL:  "http://localhost:3000",
	}
}

// テンプレート定義
var templates = map[TemplateType]*template.Template{
	TemplateWelcome:         template.Must(template.New("welcome").Parse(welcomeTemplate)),
	TemplatePasswordReset:   template.Must(template.New("password_reset").Parse(passwordResetTemplate)),
	TemplateEmailVerify:     template.Must(template.New("email_verify").Parse(emailVerifyTemplate)),
	TemplateGroupInvitation: template.Must(template.New("group_invitation").Parse(groupInvitationTemplate)),
	TemplateShareNotify:     template.Must(template.New("share_notify").Parse(shareNotifyTemplate)),
}

// RenderTemplate はテンプレートをレンダリングします
func RenderTemplate(templateType TemplateType, data TemplateData) (string, error) {
	tmpl, ok := templates[templateType]
	if !ok {
		return "", fmt.Errorf("template not found: %s", templateType)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}

// テンプレート本文
const welcomeTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.AppName}}へようこそ</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #2563eb;">{{.AppName}}へようこそ</h1>
        <p>{{.UserName}}さん、ご登録ありがとうございます。</p>
        <p>{{.AppName}}は、チームでファイルを安全に共有・管理できるクラウドストレージサービスです。</p>
        <p style="margin: 30px 0;">
            <a href="{{.AppURL}}" style="background-color: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px;">
                使い始める
            </a>
        </p>
        <p>ご不明な点がございましたら、お気軽にお問い合わせください。</p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        <p style="font-size: 12px; color: #666;">
            このメールは{{.AppName}}からの自動送信です。
        </p>
    </div>
</body>
</html>`

const passwordResetTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>パスワードリセット</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #2563eb;">パスワードリセット</h1>
        <p>{{.UserName}}さん、</p>
        <p>パスワードリセットのリクエストを受け付けました。</p>
        <p>以下のボタンをクリックして、新しいパスワードを設定してください。</p>
        <p style="margin: 30px 0;">
            <a href="{{.ActionURL}}" style="background-color: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px;">
                {{.ActionText}}
            </a>
        </p>
        <p style="color: #666; font-size: 14px;">
            このリンクは{{.ExpiresIn}}有効です。
        </p>
        <p>このリクエストに心当たりがない場合は、このメールを無視してください。</p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        <p style="font-size: 12px; color: #666;">
            このメールは{{.AppName}}からの自動送信です。
        </p>
    </div>
</body>
</html>`

const emailVerifyTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>メールアドレスの確認</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #2563eb;">メールアドレスの確認</h1>
        <p>{{.UserName}}さん、</p>
        <p>メールアドレスの確認をお願いします。</p>
        <p>以下のボタンをクリックして、メールアドレスを確認してください。</p>
        <p style="margin: 30px 0;">
            <a href="{{.ActionURL}}" style="background-color: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px;">
                {{.ActionText}}
            </a>
        </p>
        <p style="color: #666; font-size: 14px;">
            このリンクは{{.ExpiresIn}}有効です。
        </p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        <p style="font-size: 12px; color: #666;">
            このメールは{{.AppName}}からの自動送信です。
        </p>
    </div>
</body>
</html>`

const groupInvitationTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>グループへの招待</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #2563eb;">グループへの招待</h1>
        <p>{{.UserName}}さん、</p>
        <p>{{.InviterName}}さんから「{{.GroupName}}」グループへの招待が届いています。</p>
        <p style="margin: 30px 0;">
            <a href="{{.ActionURL}}" style="background-color: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px;">
                {{.ActionText}}
            </a>
        </p>
        <p style="color: #666; font-size: 14px;">
            この招待は{{.ExpiresIn}}有効です。
        </p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        <p style="font-size: 12px; color: #666;">
            このメールは{{.AppName}}からの自動送信です。
        </p>
    </div>
</body>
</html>`

const shareNotifyTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>ファイル共有のお知らせ</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #2563eb;">ファイル共有のお知らせ</h1>
        <p>{{.UserName}}さん、</p>
        <p>{{.SharerName}}さんから「{{.FileName}}」が共有されました。</p>
        <p style="margin: 30px 0;">
            <a href="{{.ActionURL}}" style="background-color: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px;">
                {{.ActionText}}
            </a>
        </p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        <p style="font-size: 12px; color: #666;">
            このメールは{{.AppName}}からの自動送信です。
        </p>
    </div>
</body>
</html>`
