package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"time"
)

// Config はSMTP設定を定義します
type Config struct {
	Host     string // SMTPホスト
	Port     int    // SMTPポート
	Username string // 認証ユーザー名
	Password string // 認証パスワード
	From     string // 送信元アドレス
	FromName string // 送信元名
	UseTLS   bool   // TLS使用有無
	Timeout  time.Duration
}

// DefaultConfig はデフォルト設定を返します
func DefaultConfig() Config {
	return Config{
		Host:     "localhost",
		Port:     1025, // MailHog default
		From:     "noreply@gc-storage.local",
		FromName: "GC Storage",
		UseTLS:   false,
		Timeout:  10 * time.Second,
	}
}

// SMTPClient はSMTPクライアントを提供します
type SMTPClient struct {
	config Config
}

// NewSMTPClient は新しいSMTPClientを作成します
func NewSMTPClient(cfg Config) *SMTPClient {
	return &SMTPClient{
		config: cfg,
	}
}

// Config は設定を返します
func (c *SMTPClient) Config() Config {
	return c.config
}

// Send はメールを送信します
func (c *SMTPClient) Send(to []string, subject, body string) error {
	msg := c.buildMessage(to, subject, body, "text/plain")
	return c.send(to, msg)
}

// SendHTML はHTMLメールを送信します
func (c *SMTPClient) SendHTML(to []string, subject, htmlBody string) error {
	msg := c.buildMessage(to, subject, htmlBody, "text/html")
	return c.send(to, msg)
}

// buildMessage はメールメッセージを構築します
func (c *SMTPClient) buildMessage(to []string, subject, body, contentType string) []byte {
	from := c.config.From
	if c.config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", c.config.FromName, c.config.From)
	}

	headers := map[string]string{
		"From":         from,
		"To":           to[0],
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": fmt.Sprintf("%s; charset=\"utf-8\"", contentType),
	}

	msg := ""
	for k, v := range headers {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	msg += "\r\n" + body

	return []byte(msg)
}

// send は実際にメールを送信します
func (c *SMTPClient) send(to []string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	var auth smtp.Auth
	if c.config.Username != "" && c.config.Password != "" {
		auth = smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.Host)
	}

	if c.config.UseTLS {
		return c.sendWithTLS(addr, auth, to, msg)
	}

	return smtp.SendMail(addr, auth, c.config.From, to, msg)
}

// sendWithTLS はTLSを使用してメールを送信します
func (c *SMTPClient) sendWithTLS(addr string, auth smtp.Auth, to []string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: c.config.Host,
	}

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
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}
	}

	if err = client.Mail(c.config.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", addr, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data writer: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return client.Quit()
}
