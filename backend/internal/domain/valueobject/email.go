package valueobject

import (
	"fmt"
	"net/mail"
	"strings"
)

// Email はメールアドレスを表す値オブジェクトです
type Email struct {
	value string
}

// NewEmail は新しいEmailを作成します
func NewEmail(value string) (Email, error) {
	value = strings.TrimSpace(strings.ToLower(value))

	if value == "" {
		return Email{}, fmt.Errorf("email cannot be empty")
	}

	if len(value) > 255 {
		return Email{}, fmt.Errorf("email must be at most 255 characters")
	}

	// RFC 5322に準拠したメールアドレスの検証
	_, err := mail.ParseAddress(value)
	if err != nil {
		return Email{}, fmt.Errorf("invalid email format")
	}

	return Email{value: value}, nil
}

// String はメールアドレスを文字列で返します
func (e Email) String() string {
	return e.value
}

// Equals は2つのEmailが等しいかを判定します
func (e Email) Equals(other Email) bool {
	return e.value == other.value
}

// Domain はメールアドレスのドメイン部分を返します
func (e Email) Domain() string {
	parts := strings.Split(e.value, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

// LocalPart はメールアドレスのローカル部分（@より前）を返します
func (e Email) LocalPart() string {
	parts := strings.Split(e.value, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}
