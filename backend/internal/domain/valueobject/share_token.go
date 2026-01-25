package valueobject

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
)

const (
	ShareTokenLength = 32 // 32 bytes = 43 chars base64 URL-safe
)

var (
	ErrShareTokenEmpty   = errors.New("share token cannot be empty")
	ErrShareTokenInvalid = errors.New("share token is invalid")
)

// ShareToken は共有リンクのトークンを表す値オブジェクト
type ShareToken struct {
	value string
}

// NewShareToken は新しいShareTokenを生成します
func NewShareToken() (ShareToken, error) {
	bytes := make([]byte, ShareTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return ShareToken{}, err
	}

	token := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)
	return ShareToken{value: token}, nil
}

// ReconstructShareToken は既存のトークン文字列からShareTokenを復元します
func ReconstructShareToken(token string) (ShareToken, error) {
	if token == "" {
		return ShareToken{}, ErrShareTokenEmpty
	}

	// Validate base64 URL encoding
	_, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(token)
	if err != nil {
		return ShareToken{}, ErrShareTokenInvalid
	}

	return ShareToken{value: token}, nil
}

// Value は値を返します
func (t ShareToken) Value() string {
	return t.value
}

// String は文字列を返します
func (t ShareToken) String() string {
	return t.value
}

// IsEmpty は空かどうかを判定します
func (t ShareToken) IsEmpty() bool {
	return t.value == ""
}

// Equals は等価性を判定します
func (t ShareToken) Equals(other ShareToken) bool {
	return t.value == other.value
}
