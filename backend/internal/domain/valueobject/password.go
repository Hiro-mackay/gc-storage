package valueobject

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLength = 8
	maxPasswordLength = 256
	bcryptCost        = 12
)

// Password はパスワードを表す値オブジェクトです
type Password struct {
	hash string
}

// NewPassword は新しいPasswordを作成します（平文から）
func NewPassword(plaintext string, email string) (Password, error) {
	if len(plaintext) < minPasswordLength {
		return Password{}, fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}

	if len(plaintext) > maxPasswordLength {
		return Password{}, fmt.Errorf("password must be at most %d characters", maxPasswordLength)
	}

	// パスワード強度チェック
	if err := validatePasswordStrength(plaintext); err != nil {
		return Password{}, err
	}

	// メールアドレスとの類似チェック
	emailLower := strings.ToLower(email)
	passwordLower := strings.ToLower(plaintext)
	if strings.Contains(passwordLower, strings.Split(emailLower, "@")[0]) {
		return Password{}, fmt.Errorf("password cannot contain email username")
	}

	// bcryptでハッシュ化
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcryptCost)
	if err != nil {
		return Password{}, fmt.Errorf("failed to hash password: %w", err)
	}

	return Password{hash: string(hash)}, nil
}

// PasswordFromHash はハッシュからPasswordを作成します（DBからの復元用）
func PasswordFromHash(hash string) Password {
	return Password{hash: hash}
}

// Hash はパスワードハッシュを返します
func (p Password) Hash() string {
	return p.hash
}

// Verify は平文パスワードがハッシュと一致するか検証します
func (p Password) Verify(plaintext string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(p.hash), []byte(plaintext))
	return err == nil
}

// validatePasswordStrength はパスワード強度を検証します
// 英大文字、英小文字、数字のうち2種以上を含む必要があります
func validatePasswordStrength(password string) error {
	var hasUpper, hasLower, hasDigit bool

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		}
	}

	count := 0
	if hasUpper {
		count++
	}
	if hasLower {
		count++
	}
	if hasDigit {
		count++
	}

	if count < 2 {
		return fmt.Errorf("password must contain at least 2 of: uppercase, lowercase, digit")
	}

	return nil
}
