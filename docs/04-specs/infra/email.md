# Email インフラストラクチャ仕様書

## 概要

本ドキュメントでは、GC StorageにおけるSMTPメール送信、テンプレート管理、非同期送信の実装仕様を定義します。

**関連アーキテクチャ:**
- [BACKEND.md](../../02-architecture/BACKEND.md) - バックエンド設計
- [user.md](../../03-domains/user.md) - ユーザードメイン

---

## 1. SMTP接続管理

### 1.1 クライアント構成

```go
// backend/internal/infrastructure/external/email/client.go

package email

import (
    "crypto/tls"
    "fmt"
    "net/smtp"
    "time"
)

// Config はSMTP接続設定を定義します
type Config struct {
    Host           string        // SMTPホスト
    Port           int           // SMTPポート (25, 465, 587)
    Username       string        // SMTP認証ユーザー名
    Password       string        // SMTP認証パスワード
    FromAddress    string        // 送信元アドレス
    FromName       string        // 送信元名
    UseTLS         bool          // TLS使用有無
    TLSInsecure    bool          // TLS証明書検証スキップ（開発環境用）
    ConnectTimeout time.Duration // 接続タイムアウト
    SendTimeout    time.Duration // 送信タイムアウト
}

// DefaultConfig はデフォルト設定を返します
func DefaultConfig() Config {
    return Config{
        Host:           "localhost",
        Port:           1025, // MailHogデフォルトポート
        FromAddress:    "noreply@gc-storage.example.com",
        FromName:       "GC Storage",
        UseTLS:         false,
        TLSInsecure:    true,
        ConnectTimeout: 10 * time.Second,
        SendTimeout:    30 * time.Second,
    }
}

// SMTPClient はSMTP操作を提供します
type SMTPClient struct {
    config Config
}

// NewSMTPClient は新しいSMTPClientを作成します
func NewSMTPClient(cfg Config) *SMTPClient {
    return &SMTPClient{config: cfg}
}

// Send はメールを送信します
func (c *SMTPClient) Send(msg *Message) error {
    addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

    // 認証設定
    var auth smtp.Auth
    if c.config.Username != "" && c.config.Password != "" {
        auth = smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.Host)
    }

    // TLS設定
    tlsConfig := &tls.Config{
        InsecureSkipVerify: c.config.TLSInsecure,
        ServerName:         c.config.Host,
    }

    // メッセージを構築
    from := fmt.Sprintf("%s <%s>", c.config.FromName, c.config.FromAddress)
    headers := map[string]string{
        "From":         from,
        "To":           msg.To,
        "Subject":      msg.Subject,
        "MIME-Version": "1.0",
        "Content-Type": fmt.Sprintf(`%s; charset="UTF-8"`, msg.ContentType),
    }

    var rawMsg string
    for k, v := range headers {
        rawMsg += fmt.Sprintf("%s: %s\r\n", k, v)
    }
    rawMsg += "\r\n" + msg.Body

    // TLSポート (465) の場合
    if c.config.Port == 465 {
        return c.sendWithTLS(addr, auth, tlsConfig, msg.To, []byte(rawMsg))
    }

    // STARTTLS (587) またはプレーンテキスト (25, 1025)
    return c.sendWithSTARTTLS(addr, auth, tlsConfig, msg.To, []byte(rawMsg))
}

func (c *SMTPClient) sendWithTLS(addr string, auth smtp.Auth, tlsConfig *tls.Config, to string, msg []byte) error {
    conn, err := tls.Dial("tcp", addr, tlsConfig)
    if err != nil {
        return fmt.Errorf("failed to connect with TLS: %w", err)
    }
    defer conn.Close()

    client, err := smtp.NewClient(conn, c.config.Host)
    if err != nil {
        return fmt.Errorf("failed to create SMTP client: %w", err)
    }
    defer client.Close()

    if auth != nil {
        if err := client.Auth(auth); err != nil {
            return fmt.Errorf("failed to authenticate: %w", err)
        }
    }

    if err := client.Mail(c.config.FromAddress); err != nil {
        return fmt.Errorf("failed to set sender: %w", err)
    }

    if err := client.Rcpt(to); err != nil {
        return fmt.Errorf("failed to set recipient: %w", err)
    }

    w, err := client.Data()
    if err != nil {
        return fmt.Errorf("failed to open data writer: %w", err)
    }

    _, err = w.Write(msg)
    if err != nil {
        return fmt.Errorf("failed to write message: %w", err)
    }

    if err := w.Close(); err != nil {
        return fmt.Errorf("failed to close data writer: %w", err)
    }

    return client.Quit()
}

func (c *SMTPClient) sendWithSTARTTLS(addr string, auth smtp.Auth, tlsConfig *tls.Config, to string, msg []byte) error {
    client, err := smtp.Dial(addr)
    if err != nil {
        return fmt.Errorf("failed to connect: %w", err)
    }
    defer client.Close()

    // STARTTLS対応確認
    if ok, _ := client.Extension("STARTTLS"); ok && c.config.UseTLS {
        if err := client.StartTLS(tlsConfig); err != nil {
            return fmt.Errorf("failed to start TLS: %w", err)
        }
    }

    if auth != nil {
        if err := client.Auth(auth); err != nil {
            return fmt.Errorf("failed to authenticate: %w", err)
        }
    }

    if err := client.Mail(c.config.FromAddress); err != nil {
        return fmt.Errorf("failed to set sender: %w", err)
    }

    if err := client.Rcpt(to); err != nil {
        return fmt.Errorf("failed to set recipient: %w", err)
    }

    w, err := client.Data()
    if err != nil {
        return fmt.Errorf("failed to open data writer: %w", err)
    }

    _, err = w.Write(msg)
    if err != nil {
        return fmt.Errorf("failed to write message: %w", err)
    }

    if err := w.Close(); err != nil {
        return fmt.Errorf("failed to close data writer: %w", err)
    }

    return client.Quit()
}

// HealthCheck はSMTP接続を確認します
func (c *SMTPClient) HealthCheck() error {
    addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
    client, err := smtp.Dial(addr)
    if err != nil {
        return fmt.Errorf("failed to connect to SMTP server: %w", err)
    }
    defer client.Close()
    return client.Quit()
}
```

### 1.2 メッセージ構造

```go
// backend/internal/infrastructure/external/email/message.go

package email

// ContentType はメールのコンテンツタイプを表します
type ContentType string

const (
    ContentTypePlain ContentType = "text/plain"
    ContentTypeHTML  ContentType = "text/html"
)

// Message はメールメッセージを表します
type Message struct {
    To          string
    Subject     string
    Body        string
    ContentType ContentType
}

// NewTextMessage はテキストメッセージを作成します
func NewTextMessage(to, subject, body string) *Message {
    return &Message{
        To:          to,
        Subject:     subject,
        Body:        body,
        ContentType: ContentTypePlain,
    }
}

// NewHTMLMessage はHTMLメッセージを作成します
func NewHTMLMessage(to, subject, body string) *Message {
    return &Message{
        To:          to,
        Subject:     subject,
        Body:        body,
        ContentType: ContentTypeHTML,
    }
}
```

### 1.3 環境変数設定

```bash
# .env.local (MailHog)
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM_ADDRESS=noreply@gc-storage.local
SMTP_FROM_NAME=GC Storage
SMTP_USE_TLS=false

# 本番環境 (.env.sample)
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM_ADDRESS=noreply@gc-storage.example.com
SMTP_FROM_NAME=GC Storage
SMTP_USE_TLS=true
```

### 1.4 ディレクトリ構成

```
backend/internal/infrastructure/external/email/
├── client.go           # SMTPクライアント
├── message.go          # メッセージ構造
├── service.go          # メールサービス
├── templates.go        # テンプレートローダー
└── templates/          # メールテンプレート
    ├── verification.html
    ├── password_reset.html
    ├── invitation.html
    ├── share_notification.html
    └── base.html
```

---

## 2. テンプレート管理

### 2.1 テンプレートローダー

```go
// backend/internal/infrastructure/external/email/templates.go

package email

import (
    "bytes"
    "embed"
    "fmt"
    "html/template"
    "sync"
)

//go:embed templates/*.html
var templateFS embed.FS

// TemplateManager はメールテンプレートを管理します
type TemplateManager struct {
    templates map[string]*template.Template
    baseURL   string
    mu        sync.RWMutex
}

// NewTemplateManager は新しいTemplateManagerを作成します
func NewTemplateManager(baseURL string) (*TemplateManager, error) {
    tm := &TemplateManager{
        templates: make(map[string]*template.Template),
        baseURL:   baseURL,
    }

    // テンプレートをロード
    if err := tm.loadTemplates(); err != nil {
        return nil, err
    }

    return tm, nil
}

func (tm *TemplateManager) loadTemplates() error {
    // ベーステンプレートを読み込み
    baseContent, err := templateFS.ReadFile("templates/base.html")
    if err != nil {
        return fmt.Errorf("failed to read base template: %w", err)
    }

    templateNames := []string{
        "verification",
        "password_reset",
        "invitation",
        "share_notification",
    }

    for _, name := range templateNames {
        content, err := templateFS.ReadFile(fmt.Sprintf("templates/%s.html", name))
        if err != nil {
            return fmt.Errorf("failed to read template %s: %w", name, err)
        }

        // ベーステンプレートと結合
        t, err := template.New(name).Parse(string(baseContent))
        if err != nil {
            return fmt.Errorf("failed to parse base template for %s: %w", name, err)
        }

        t, err = t.Parse(string(content))
        if err != nil {
            return fmt.Errorf("failed to parse template %s: %w", name, err)
        }

        tm.templates[name] = t
    }

    return nil
}

// Render はテンプレートをレンダリングします
func (tm *TemplateManager) Render(name string, data interface{}) (string, error) {
    tm.mu.RLock()
    t, ok := tm.templates[name]
    tm.mu.RUnlock()

    if !ok {
        return "", fmt.Errorf("template not found: %s", name)
    }

    var buf bytes.Buffer
    if err := t.Execute(&buf, data); err != nil {
        return "", fmt.Errorf("failed to render template %s: %w", name, err)
    }

    return buf.String(), nil
}

// BaseURL はベースURLを返します
func (tm *TemplateManager) BaseURL() string {
    return tm.baseURL
}
```

### 2.2 テンプレートデータ構造

```go
// backend/internal/infrastructure/external/email/template_data.go

package email

import "time"

// VerificationData はメール認証用のデータ
type VerificationData struct {
    UserName        string
    VerificationURL string
    ExpiresAt       time.Time
    SupportEmail    string
}

// PasswordResetData はパスワードリセット用のデータ
type PasswordResetData struct {
    UserName     string
    ResetURL     string
    ExpiresAt    time.Time
    IPAddress    string
    SupportEmail string
}

// InvitationData はグループ招待用のデータ
type InvitationData struct {
    InviterName   string
    GroupName     string
    InvitationURL string
    ExpiresAt     time.Time
    Message       string
}

// ShareNotificationData は共有通知用のデータ
type ShareNotificationData struct {
    SharerName   string
    ResourceName string
    ResourceType string // "file" or "folder"
    ShareURL     string
    Permission   string // "view", "edit", "manage"
    Message      string
}
```

### 2.3 テンプレートファイル例

```html
<!-- internal/infrastructure/external/email/templates/base.html -->
<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{block "title" .}}GC Storage{{end}}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            text-align: center;
            padding: 20px 0;
            border-bottom: 1px solid #eee;
        }
        .content {
            padding: 30px 0;
        }
        .button {
            display: inline-block;
            padding: 12px 24px;
            background-color: #2563eb;
            color: white;
            text-decoration: none;
            border-radius: 6px;
            margin: 20px 0;
        }
        .footer {
            padding: 20px 0;
            border-top: 1px solid #eee;
            font-size: 12px;
            color: #666;
            text-align: center;
        }
        .warning {
            background-color: #fef3c7;
            padding: 10px;
            border-radius: 4px;
            margin: 10px 0;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>GC Storage</h1>
    </div>
    <div class="content">
        {{block "content" .}}{{end}}
    </div>
    <div class="footer">
        <p>このメールはGC Storageから自動送信されています。</p>
        <p>ご不明な点がございましたら、サポートまでお問い合わせください。</p>
    </div>
</body>
</html>
```

```html
<!-- internal/infrastructure/external/email/templates/verification.html -->
{{define "title"}}メールアドレスの確認 - GC Storage{{end}}

{{define "content"}}
<h2>メールアドレスの確認</h2>

<p>{{.UserName}} 様</p>

<p>GC Storageへのご登録ありがとうございます。</p>

<p>以下のボタンをクリックして、メールアドレスの確認を完了してください。</p>

<p style="text-align: center;">
    <a href="{{.VerificationURL}}" class="button">メールアドレスを確認する</a>
</p>

<div class="warning">
    <p><strong>注意:</strong> このリンクは {{.ExpiresAt.Format "2006年01月02日 15:04"}} まで有効です。</p>
</div>

<p>ボタンがクリックできない場合は、以下のURLをブラウザに直接貼り付けてください:</p>
<p style="word-break: break-all;">{{.VerificationURL}}</p>

<p>このメールに心当たりがない場合は、このメールを無視してください。</p>
{{end}}
```

```html
<!-- internal/infrastructure/external/email/templates/password_reset.html -->
{{define "title"}}パスワードリセット - GC Storage{{end}}

{{define "content"}}
<h2>パスワードリセット</h2>

<p>{{.UserName}} 様</p>

<p>パスワードリセットのリクエストを受け付けました。</p>

<p>以下のボタンをクリックして、新しいパスワードを設定してください。</p>

<p style="text-align: center;">
    <a href="{{.ResetURL}}" class="button">パスワードをリセットする</a>
</p>

<div class="warning">
    <p><strong>注意:</strong> このリンクは {{.ExpiresAt.Format "2006年01月02日 15:04"}} まで有効です。</p>
    <p>リクエスト元IPアドレス: {{.IPAddress}}</p>
</div>

<p>ボタンがクリックできない場合は、以下のURLをブラウザに直接貼り付けてください:</p>
<p style="word-break: break-all;">{{.ResetURL}}</p>

<p>このリクエストに心当たりがない場合は、アカウントのセキュリティを確認し、
サポート（{{.SupportEmail}}）までご連絡ください。</p>
{{end}}
```

```html
<!-- internal/infrastructure/external/email/templates/invitation.html -->
{{define "title"}}グループへの招待 - GC Storage{{end}}

{{define "content"}}
<h2>グループへの招待</h2>

<p>こんにちは、</p>

<p><strong>{{.InviterName}}</strong> さんから、<strong>{{.GroupName}}</strong> グループへの招待を受けています。</p>

{{if .Message}}
<blockquote style="border-left: 3px solid #2563eb; padding-left: 15px; margin: 20px 0; color: #555;">
    {{.Message}}
</blockquote>
{{end}}

<p style="text-align: center;">
    <a href="{{.InvitationURL}}" class="button">招待を受け入れる</a>
</p>

<div class="warning">
    <p><strong>注意:</strong> この招待は {{.ExpiresAt.Format "2006年01月02日 15:04"}} まで有効です。</p>
</div>

<p>ボタンがクリックできない場合は、以下のURLをブラウザに直接貼り付けてください:</p>
<p style="word-break: break-all;">{{.InvitationURL}}</p>

<p>この招待に心当たりがない場合は、このメールを無視してください。</p>
{{end}}
```

```html
<!-- internal/infrastructure/external/email/templates/share_notification.html -->
{{define "title"}}共有のお知らせ - GC Storage{{end}}

{{define "content"}}
<h2>共有のお知らせ</h2>

<p>こんにちは、</p>

<p><strong>{{.SharerName}}</strong> さんが{{if eq .ResourceType "file"}}ファイル{{else}}フォルダ{{end}}を共有しました。</p>

<table style="margin: 20px 0; width: 100%;">
    <tr>
        <td style="padding: 8px; color: #666;">{{if eq .ResourceType "file"}}ファイル名{{else}}フォルダ名{{end}}</td>
        <td style="padding: 8px;"><strong>{{.ResourceName}}</strong></td>
    </tr>
    <tr>
        <td style="padding: 8px; color: #666;">権限</td>
        <td style="padding: 8px;">
            {{if eq .Permission "view"}}閲覧のみ{{else if eq .Permission "edit"}}編集可能{{else}}管理者{{end}}
        </td>
    </tr>
</table>

{{if .Message}}
<blockquote style="border-left: 3px solid #2563eb; padding-left: 15px; margin: 20px 0; color: #555;">
    {{.Message}}
</blockquote>
{{end}}

<p style="text-align: center;">
    <a href="{{.ShareURL}}" class="button">共有を確認する</a>
</p>

<p>ボタンがクリックできない場合は、以下のURLをブラウザに直接貼り付けてください:</p>
<p style="word-break: break-all;">{{.ShareURL}}</p>
{{end}}
```

---

## 3. メールサービス

### 3.1 サービスインターフェース

```go
// backend/internal/domain/service/email.go

package service

import "context"

// EmailService はメール送信のインターフェースを定義します
type EmailService interface {
    // 認証関連
    SendVerificationEmail(ctx context.Context, email, userName, verificationURL string) error
    SendPasswordResetEmail(ctx context.Context, email, userName, resetURL, ipAddress string) error

    // グループ関連
    SendInvitationEmail(ctx context.Context, email, inviterName, groupName, invitationURL, message string) error

    // 共有関連
    SendShareNotificationEmail(ctx context.Context, params ShareNotificationParams) error
}

// ShareNotificationParams は共有通知のパラメータ
type ShareNotificationParams struct {
    Email        string
    SharerName   string
    ResourceName string
    ResourceType string
    ShareURL     string
    Permission   string
    Message      string
}
```

### 3.2 サービス実装

```go
// backend/internal/infrastructure/external/email/service.go

package email

import (
    "context"
    "fmt"
    "time"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
)

// EmailServiceImpl はEmailServiceの実装です
type EmailServiceImpl struct {
    client          *SMTPClient
    templateManager *TemplateManager
    supportEmail    string
}

// NewEmailService は新しいEmailServiceを作成します
func NewEmailService(client *SMTPClient, tm *TemplateManager, supportEmail string) *EmailServiceImpl {
    return &EmailServiceImpl{
        client:          client,
        templateManager: tm,
        supportEmail:    supportEmail,
    }
}

// SendVerificationEmail はメール認証メールを送信します
func (s *EmailServiceImpl) SendVerificationEmail(ctx context.Context, email, userName, verificationURL string) error {
    data := VerificationData{
        UserName:        userName,
        VerificationURL: verificationURL,
        ExpiresAt:       time.Now().Add(24 * time.Hour),
        SupportEmail:    s.supportEmail,
    }

    body, err := s.templateManager.Render("verification", data)
    if err != nil {
        return fmt.Errorf("failed to render verification template: %w", err)
    }

    msg := NewHTMLMessage(email, "【GC Storage】メールアドレスの確認", body)
    return s.client.Send(msg)
}

// SendPasswordResetEmail はパスワードリセットメールを送信します
func (s *EmailServiceImpl) SendPasswordResetEmail(ctx context.Context, email, userName, resetURL, ipAddress string) error {
    data := PasswordResetData{
        UserName:     userName,
        ResetURL:     resetURL,
        ExpiresAt:    time.Now().Add(1 * time.Hour),
        IPAddress:    ipAddress,
        SupportEmail: s.supportEmail,
    }

    body, err := s.templateManager.Render("password_reset", data)
    if err != nil {
        return fmt.Errorf("failed to render password_reset template: %w", err)
    }

    msg := NewHTMLMessage(email, "【GC Storage】パスワードリセット", body)
    return s.client.Send(msg)
}

// SendInvitationEmail はグループ招待メールを送信します
func (s *EmailServiceImpl) SendInvitationEmail(ctx context.Context, email, inviterName, groupName, invitationURL, message string) error {
    data := InvitationData{
        InviterName:   inviterName,
        GroupName:     groupName,
        InvitationURL: invitationURL,
        ExpiresAt:     time.Now().Add(7 * 24 * time.Hour),
        Message:       message,
    }

    body, err := s.templateManager.Render("invitation", data)
    if err != nil {
        return fmt.Errorf("failed to render invitation template: %w", err)
    }

    subject := fmt.Sprintf("【GC Storage】%sさんから%sへの招待", inviterName, groupName)
    msg := NewHTMLMessage(email, subject, body)
    return s.client.Send(msg)
}

// SendShareNotificationEmail は共有通知メールを送信します
func (s *EmailServiceImpl) SendShareNotificationEmail(ctx context.Context, params service.ShareNotificationParams) error {
    data := ShareNotificationData{
        SharerName:   params.SharerName,
        ResourceName: params.ResourceName,
        ResourceType: params.ResourceType,
        ShareURL:     params.ShareURL,
        Permission:   params.Permission,
        Message:      params.Message,
    }

    body, err := s.templateManager.Render("share_notification", data)
    if err != nil {
        return fmt.Errorf("failed to render share_notification template: %w", err)
    }

    resourceType := "ファイル"
    if params.ResourceType == "folder" {
        resourceType = "フォルダ"
    }

    subject := fmt.Sprintf("【GC Storage】%sさんが%sを共有しました", params.SharerName, resourceType)
    msg := NewHTMLMessage(params.Email, subject, body)
    return s.client.Send(msg)
}

// Verify interface compliance
var _ service.EmailService = (*EmailServiceImpl)(nil)
```

---

## 4. 非同期送信

### 4.1 メールキュー

```go
// backend/internal/infrastructure/external/email/queue.go

package email

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "sync"
    "time"

    "github.com/redis/go-redis/v9"
)

// QueuedEmail はキューに入れられたメールを表します
type QueuedEmail struct {
    ID        string          `json:"id"`
    Type      string          `json:"type"`
    Data      json.RawMessage `json:"data"`
    Attempts  int             `json:"attempts"`
    CreatedAt time.Time       `json:"created_at"`
    NextRetry time.Time       `json:"next_retry"`
}

// EmailQueue はメール送信キューを管理します
type EmailQueue struct {
    redis      *redis.Client
    queueKey   string
    maxRetries int
    retryDelay time.Duration
}

// NewEmailQueue は新しいEmailQueueを作成します
func NewEmailQueue(redisClient *redis.Client) *EmailQueue {
    return &EmailQueue{
        redis:      redisClient,
        queueKey:   "email:queue",
        maxRetries: 3,
        retryDelay: 5 * time.Minute,
    }
}

// Enqueue はメールをキューに追加します
func (q *EmailQueue) Enqueue(ctx context.Context, emailType string, data interface{}) error {
    dataBytes, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("failed to marshal email data: %w", err)
    }

    email := QueuedEmail{
        ID:        fmt.Sprintf("email:%d", time.Now().UnixNano()),
        Type:      emailType,
        Data:      dataBytes,
        Attempts:  0,
        CreatedAt: time.Now(),
        NextRetry: time.Now(),
    }

    emailBytes, err := json.Marshal(email)
    if err != nil {
        return fmt.Errorf("failed to marshal queued email: %w", err)
    }

    return q.redis.RPush(ctx, q.queueKey, emailBytes).Err()
}

// Dequeue はキューからメールを取得します
func (q *EmailQueue) Dequeue(ctx context.Context) (*QueuedEmail, error) {
    result, err := q.redis.LPop(ctx, q.queueKey).Bytes()
    if err != nil {
        if err == redis.Nil {
            return nil, nil
        }
        return nil, fmt.Errorf("failed to dequeue email: %w", err)
    }

    var email QueuedEmail
    if err := json.Unmarshal(result, &email); err != nil {
        return nil, fmt.Errorf("failed to unmarshal queued email: %w", err)
    }

    return &email, nil
}

// Requeue は失敗したメールを再キューに追加します
func (q *EmailQueue) Requeue(ctx context.Context, email *QueuedEmail) error {
    email.Attempts++
    email.NextRetry = time.Now().Add(q.retryDelay * time.Duration(email.Attempts))

    if email.Attempts >= q.maxRetries {
        // デッドレターキューに移動
        return q.moveToDeadLetter(ctx, email)
    }

    emailBytes, err := json.Marshal(email)
    if err != nil {
        return fmt.Errorf("failed to marshal queued email: %w", err)
    }

    return q.redis.RPush(ctx, q.queueKey, emailBytes).Err()
}

func (q *EmailQueue) moveToDeadLetter(ctx context.Context, email *QueuedEmail) error {
    emailBytes, err := json.Marshal(email)
    if err != nil {
        return fmt.Errorf("failed to marshal queued email: %w", err)
    }

    deadLetterKey := "email:dead_letter"
    return q.redis.RPush(ctx, deadLetterKey, emailBytes).Err()
}

// QueueLength はキューの長さを返します
func (q *EmailQueue) QueueLength(ctx context.Context) (int64, error) {
    return q.redis.LLen(ctx, q.queueKey).Result()
}
```

### 4.2 メールワーカー

```go
// backend/internal/infrastructure/external/email/worker.go

package email

import (
    "context"
    "encoding/json"
    "log/slog"
    "sync"
    "time"
)

// EmailWorker はメール送信ワーカーです
type EmailWorker struct {
    queue   *EmailQueue
    service *EmailServiceImpl
    logger  *slog.Logger

    stopCh chan struct{}
    wg     sync.WaitGroup
}

// NewEmailWorker は新しいEmailWorkerを作成します
func NewEmailWorker(queue *EmailQueue, service *EmailServiceImpl, logger *slog.Logger) *EmailWorker {
    return &EmailWorker{
        queue:   queue,
        service: service,
        logger:  logger,
        stopCh:  make(chan struct{}),
    }
}

// Start はワーカーを開始します
func (w *EmailWorker) Start(ctx context.Context, workers int) {
    for i := 0; i < workers; i++ {
        w.wg.Add(1)
        go w.worker(ctx, i)
    }
}

// Stop はワーカーを停止します
func (w *EmailWorker) Stop() {
    close(w.stopCh)
    w.wg.Wait()
}

func (w *EmailWorker) worker(ctx context.Context, id int) {
    defer w.wg.Done()

    w.logger.Info("email worker started", "worker_id", id)

    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-w.stopCh:
            w.logger.Info("email worker stopped", "worker_id", id)
            return
        case <-ctx.Done():
            w.logger.Info("email worker context cancelled", "worker_id", id)
            return
        case <-ticker.C:
            w.processQueue(ctx)
        }
    }
}

func (w *EmailWorker) processQueue(ctx context.Context) {
    email, err := w.queue.Dequeue(ctx)
    if err != nil {
        w.logger.Error("failed to dequeue email", "error", err)
        return
    }

    if email == nil {
        return // キューが空
    }

    // リトライ時刻までの待機
    if time.Now().Before(email.NextRetry) {
        // キューに戻す
        w.queue.Requeue(ctx, email)
        return
    }

    if err := w.sendEmail(ctx, email); err != nil {
        w.logger.Error("failed to send email",
            "email_id", email.ID,
            "type", email.Type,
            "attempts", email.Attempts,
            "error", err,
        )

        // リトライキューに追加
        if err := w.queue.Requeue(ctx, email); err != nil {
            w.logger.Error("failed to requeue email", "error", err)
        }
        return
    }

    w.logger.Info("email sent successfully",
        "email_id", email.ID,
        "type", email.Type,
    )
}

func (w *EmailWorker) sendEmail(ctx context.Context, email *QueuedEmail) error {
    switch email.Type {
    case "verification":
        var data struct {
            Email           string `json:"email"`
            UserName        string `json:"user_name"`
            VerificationURL string `json:"verification_url"`
        }
        if err := json.Unmarshal(email.Data, &data); err != nil {
            return err
        }
        return w.service.SendVerificationEmail(ctx, data.Email, data.UserName, data.VerificationURL)

    case "password_reset":
        var data struct {
            Email     string `json:"email"`
            UserName  string `json:"user_name"`
            ResetURL  string `json:"reset_url"`
            IPAddress string `json:"ip_address"`
        }
        if err := json.Unmarshal(email.Data, &data); err != nil {
            return err
        }
        return w.service.SendPasswordResetEmail(ctx, data.Email, data.UserName, data.ResetURL, data.IPAddress)

    case "invitation":
        var data struct {
            Email         string `json:"email"`
            InviterName   string `json:"inviter_name"`
            GroupName     string `json:"group_name"`
            InvitationURL string `json:"invitation_url"`
            Message       string `json:"message"`
        }
        if err := json.Unmarshal(email.Data, &data); err != nil {
            return err
        }
        return w.service.SendInvitationEmail(ctx, data.Email, data.InviterName, data.GroupName, data.InvitationURL, data.Message)

    default:
        return fmt.Errorf("unknown email type: %s", email.Type)
    }
}
```

### 4.3 非同期メールサービス

```go
// backend/internal/infrastructure/external/email/async_service.go

package email

import (
    "context"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
)

// AsyncEmailService は非同期メール送信サービスです
type AsyncEmailService struct {
    queue       *EmailQueue
    syncService *EmailServiceImpl
    useAsync    bool
}

// NewAsyncEmailService は新しいAsyncEmailServiceを作成します
func NewAsyncEmailService(queue *EmailQueue, syncService *EmailServiceImpl, useAsync bool) *AsyncEmailService {
    return &AsyncEmailService{
        queue:       queue,
        syncService: syncService,
        useAsync:    useAsync,
    }
}

// SendVerificationEmail はメール認証メールを送信します
func (s *AsyncEmailService) SendVerificationEmail(ctx context.Context, email, userName, verificationURL string) error {
    if !s.useAsync {
        return s.syncService.SendVerificationEmail(ctx, email, userName, verificationURL)
    }

    return s.queue.Enqueue(ctx, "verification", map[string]string{
        "email":            email,
        "user_name":        userName,
        "verification_url": verificationURL,
    })
}

// SendPasswordResetEmail はパスワードリセットメールを送信します
func (s *AsyncEmailService) SendPasswordResetEmail(ctx context.Context, email, userName, resetURL, ipAddress string) error {
    if !s.useAsync {
        return s.syncService.SendPasswordResetEmail(ctx, email, userName, resetURL, ipAddress)
    }

    return s.queue.Enqueue(ctx, "password_reset", map[string]string{
        "email":      email,
        "user_name":  userName,
        "reset_url":  resetURL,
        "ip_address": ipAddress,
    })
}

// SendInvitationEmail はグループ招待メールを送信します
func (s *AsyncEmailService) SendInvitationEmail(ctx context.Context, email, inviterName, groupName, invitationURL, message string) error {
    if !s.useAsync {
        return s.syncService.SendInvitationEmail(ctx, email, inviterName, groupName, invitationURL, message)
    }

    return s.queue.Enqueue(ctx, "invitation", map[string]string{
        "email":          email,
        "inviter_name":   inviterName,
        "group_name":     groupName,
        "invitation_url": invitationURL,
        "message":        message,
    })
}

// SendShareNotificationEmail は共有通知メールを送信します
func (s *AsyncEmailService) SendShareNotificationEmail(ctx context.Context, params service.ShareNotificationParams) error {
    if !s.useAsync {
        return s.syncService.SendShareNotificationEmail(ctx, params)
    }

    return s.queue.Enqueue(ctx, "share_notification", params)
}

// Verify interface compliance
var _ service.EmailService = (*AsyncEmailService)(nil)
```

---

## 5. 初期化とDI

### 5.1 依存関係の初期化

```go
// backend/internal/infrastructure/di/email.go

package di

import (
    "log/slog"

    "github.com/redis/go-redis/v9"

    "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/external/email"
)

// EmailComponents はメール関連の依存関係を保持します
type EmailComponents struct {
    Client       *email.SMTPClient
    Templates    *email.TemplateManager
    SyncService  *email.EmailServiceImpl
    AsyncService *email.AsyncEmailService
    Queue        *email.EmailQueue
    Worker       *email.EmailWorker
}

// NewEmailComponents はメール関連の依存関係を初期化します
func NewEmailComponents(
    cfg email.Config,
    baseURL string,
    supportEmail string,
    redisClient *redis.Client,
    useAsync bool,
    logger *slog.Logger,
) (*EmailComponents, error) {
    client := email.NewSMTPClient(cfg)

    templates, err := email.NewTemplateManager(baseURL)
    if err != nil {
        return nil, err
    }

    syncService := email.NewEmailService(client, templates, supportEmail)

    queue := email.NewEmailQueue(redisClient)
    asyncService := email.NewAsyncEmailService(queue, syncService, useAsync)

    worker := email.NewEmailWorker(queue, syncService, logger)

    return &EmailComponents{
        Client:       client,
        Templates:    templates,
        SyncService:  syncService,
        AsyncService: asyncService,
        Queue:        queue,
        Worker:       worker,
    }, nil
}

// StartWorker はメールワーカーを開始します
func (c *EmailComponents) StartWorker(ctx context.Context, workers int) {
    c.Worker.Start(ctx, workers)
}

// StopWorker はメールワーカーを停止します
func (c *EmailComponents) StopWorker() {
    c.Worker.Stop()
}
```

---

## 6. テストヘルパー

### 6.1 モックメールサービス

```go
// backend/internal/infrastructure/external/email/testhelper/mock.go

package testhelper

import (
    "context"
    "sync"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
)

// SentEmail は送信されたメールを表します
type SentEmail struct {
    Type string
    To   string
    Data interface{}
}

// MockEmailService はテスト用のモックメールサービスです
type MockEmailService struct {
    mu    sync.Mutex
    sent  []SentEmail
    Error error
}

// NewMockEmailService は新しいMockEmailServiceを作成します
func NewMockEmailService() *MockEmailService {
    return &MockEmailService{
        sent: make([]SentEmail, 0),
    }
}

// SendVerificationEmail はメール認証メールを記録します
func (m *MockEmailService) SendVerificationEmail(ctx context.Context, email, userName, verificationURL string) error {
    if m.Error != nil {
        return m.Error
    }

    m.mu.Lock()
    defer m.mu.Unlock()

    m.sent = append(m.sent, SentEmail{
        Type: "verification",
        To:   email,
        Data: map[string]string{
            "user_name":        userName,
            "verification_url": verificationURL,
        },
    })

    return nil
}

// SendPasswordResetEmail はパスワードリセットメールを記録します
func (m *MockEmailService) SendPasswordResetEmail(ctx context.Context, email, userName, resetURL, ipAddress string) error {
    if m.Error != nil {
        return m.Error
    }

    m.mu.Lock()
    defer m.mu.Unlock()

    m.sent = append(m.sent, SentEmail{
        Type: "password_reset",
        To:   email,
        Data: map[string]string{
            "user_name":  userName,
            "reset_url":  resetURL,
            "ip_address": ipAddress,
        },
    })

    return nil
}

// SendInvitationEmail はグループ招待メールを記録します
func (m *MockEmailService) SendInvitationEmail(ctx context.Context, email, inviterName, groupName, invitationURL, message string) error {
    if m.Error != nil {
        return m.Error
    }

    m.mu.Lock()
    defer m.mu.Unlock()

    m.sent = append(m.sent, SentEmail{
        Type: "invitation",
        To:   email,
        Data: map[string]string{
            "inviter_name":   inviterName,
            "group_name":     groupName,
            "invitation_url": invitationURL,
            "message":        message,
        },
    })

    return nil
}

// SendShareNotificationEmail は共有通知メールを記録します
func (m *MockEmailService) SendShareNotificationEmail(ctx context.Context, params service.ShareNotificationParams) error {
    if m.Error != nil {
        return m.Error
    }

    m.mu.Lock()
    defer m.mu.Unlock()

    m.sent = append(m.sent, SentEmail{
        Type: "share_notification",
        To:   params.Email,
        Data: params,
    })

    return nil
}

// GetSentEmails は送信されたメールを返します
func (m *MockEmailService) GetSentEmails() []SentEmail {
    m.mu.Lock()
    defer m.mu.Unlock()

    result := make([]SentEmail, len(m.sent))
    copy(result, m.sent)
    return result
}

// Clear は送信記録をクリアします
func (m *MockEmailService) Clear() {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.sent = make([]SentEmail, 0)
}

// Verify interface compliance
var _ service.EmailService = (*MockEmailService)(nil)
```

---

## 7. 受け入れ基準

### 7.1 機能要件

| 項目 | 基準 |
|------|------|
| SMTP接続 | MailHog/本番SMTPに接続できる |
| メール認証 | 認証メールが送信される |
| パスワードリセット | リセットメールが送信される |
| グループ招待 | 招待メールが送信される |
| 共有通知 | 共有通知メールが送信される |
| テンプレート | HTMLテンプレートが正しくレンダリングされる |
| 非同期送信 | キューイングと非同期送信が動作する |
| リトライ | 失敗時にリトライされる |

### 7.2 非機能要件

| 項目 | 基準 |
|------|------|
| 接続タイムアウト | 10秒 |
| 送信タイムアウト | 30秒 |
| リトライ回数 | 最大3回 |
| リトライ間隔 | 5分、10分、15分 |
| デッドレターキュー | 失敗メールの保存 |

### 7.3 チェックリスト

- [ ] SMTP接続が確立できる
- [ ] ヘルスチェックが動作する
- [ ] テンプレートが正しくレンダリングされる
- [ ] 認証メールが送信される
- [ ] パスワードリセットメールが送信される
- [ ] 招待メールが送信される
- [ ] 共有通知メールが送信される
- [ ] メールがキューに追加される
- [ ] ワーカーがキューを処理する
- [ ] 失敗時にリトライされる
- [ ] デッドレターキューに移動される
- [ ] MailHogでメールを確認できる

---

## 関連ドキュメント

- [redis.md](./redis.md) - Redis基盤仕様（キュー）
- [auth-registration.md](../features/auth-registration.md) - 登録仕様（メール認証）
- [auth-password.md](../features/auth-password.md) - パスワード仕様（リセットメール）
- [group-invitation.md](../features/group-invitation.md) - 招待仕様（招待メール）
- [sharing.md](../features/sharing.md) - 共有仕様（共有通知）
