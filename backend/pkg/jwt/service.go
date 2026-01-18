package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTService はJWT操作を提供します
type JWTService struct {
	config Config
}

// NewJWTService は新しいJWTServiceを作成します
func NewJWTService(cfg Config) *JWTService {
	return &JWTService{config: cfg}
}

// GenerateTokenPair はアクセストークンとリフレッシュトークンのペアを生成します
func (s *JWTService) GenerateTokenPair(userID uuid.UUID, email, sessionID string) (accessToken, refreshToken string, err error) {
	now := time.Now()

	// Access Token
	accessClaims := AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   userID.String(),
			Audience:  s.config.Audience,
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.AccessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		UserID:    userID,
		Email:     email,
		SessionID: sessionID,
	}

	accessToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.config.SecretKey))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	// Refresh Token
	refreshClaims := RefreshTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   userID.String(),
			Audience:  s.config.Audience,
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.RefreshTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		UserID:    userID,
		SessionID: sessionID,
	}

	refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(s.config.SecretKey))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// GenerateAccessToken はアクセストークンのみを生成します
func (s *JWTService) GenerateAccessToken(userID uuid.UUID, email, sessionID string) (string, error) {
	now := time.Now()

	claims := AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   userID.String(),
			Audience:  s.config.Audience,
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.AccessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		UserID:    userID,
		Email:     email,
		SessionID: sessionID,
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.config.SecretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return token, nil
}

// ValidateAccessToken はアクセストークンを検証します
func (s *JWTService) ValidateAccessToken(tokenString string) (*AccessTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %v", ErrInvalidSigningMethod, token.Header["alg"])
		}
		return []byte(s.config.SecretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse access token: %w", err)
	}

	claims, ok := token.Claims.(*AccessTokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateRefreshToken はリフレッシュトークンを検証します
func (s *JWTService) ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %v", ErrInvalidSigningMethod, token.Header["alg"])
		}
		return []byte(s.config.SecretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse refresh token: %w", err)
	}

	claims, ok := token.Claims.(*RefreshTokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetAccessTokenExpiry はアクセストークンの有効期限を返します
func (s *JWTService) GetAccessTokenExpiry() time.Duration {
	return s.config.AccessTokenExpiry
}

// GetRefreshTokenExpiry はリフレッシュトークンの有効期限を返します
func (s *JWTService) GetRefreshTokenExpiry() time.Duration {
	return s.config.RefreshTokenExpiry
}
