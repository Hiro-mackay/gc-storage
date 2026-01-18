package jwt

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AccessTokenClaims はアクセストークンのクレームを定義します
type AccessTokenClaims struct {
	jwt.RegisteredClaims
	UserID    uuid.UUID `json:"uid"`
	Email     string    `json:"email"`
	SessionID string    `json:"sid"`
}

// RefreshTokenClaims はリフレッシュトークンのクレームを定義します
type RefreshTokenClaims struct {
	jwt.RegisteredClaims
	UserID    uuid.UUID `json:"uid"`
	SessionID string    `json:"sid"`
}
