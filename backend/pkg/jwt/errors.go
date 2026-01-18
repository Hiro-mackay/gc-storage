package jwt

import "errors"

var (
	ErrSecretKeyRequired = errors.New("jwt secret key is required")
	ErrSecretKeyTooShort = errors.New("jwt secret key must be at least 32 characters")
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenExpired      = errors.New("token has expired")
	ErrInvalidSigningMethod = errors.New("invalid signing method")
)
