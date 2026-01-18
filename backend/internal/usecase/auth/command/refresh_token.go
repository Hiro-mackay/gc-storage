package command

import (
	"context"
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/cache"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/jwt"
)

// RefreshTokenInput はトークンリフレッシュの入力を定義します
type RefreshTokenInput struct {
	RefreshToken string
}

// RefreshTokenOutput はトークンリフレッシュの出力を定義します
type RefreshTokenOutput struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// RefreshTokenCommand はトークンリフレッシュコマンドです
type RefreshTokenCommand struct {
	userRepo     repository.UserRepository
	sessionRepo  repository.SessionRepository
	jwtService   *jwt.JWTService
	jwtBlacklist *cache.JWTBlacklist
}

// NewRefreshTokenCommand は新しいRefreshTokenCommandを作成します
func NewRefreshTokenCommand(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	jwtService *jwt.JWTService,
	jwtBlacklist *cache.JWTBlacklist,
) *RefreshTokenCommand {
	return &RefreshTokenCommand{
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		jwtService:   jwtService,
		jwtBlacklist: jwtBlacklist,
	}
}

// Execute はトークンリフレッシュを実行します
func (c *RefreshTokenCommand) Execute(ctx context.Context, input RefreshTokenInput) (*RefreshTokenOutput, error) {
	// 1. リフレッシュトークンを検証
	claims, err := c.jwtService.ValidateRefreshToken(input.RefreshToken)
	if err != nil {
		return nil, apperror.NewUnauthorizedError("invalid refresh token")
	}

	// 2. セッションを検索
	session, err := c.sessionRepo.FindByID(ctx, claims.SessionID)
	if err != nil {
		return nil, apperror.NewUnauthorizedError("session not found")
	}

	// 3. セッション有効性チェック
	if session.ExpiresAt.Before(time.Now()) {
		return nil, apperror.NewUnauthorizedError("session expired")
	}

	if session.RefreshToken != input.RefreshToken {
		// トークンが一致しない = トークン再利用攻撃の可能性
		// セッションを無効化
		c.sessionRepo.DeleteByUserID(ctx, session.UserID)
		return nil, apperror.NewUnauthorizedError("token reuse detected")
	}

	// 4. ユーザー取得・状態チェック
	user, err := c.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, apperror.NewUnauthorizedError("user not found")
	}

	if user.Status != entity.UserStatusActive {
		return nil, apperror.NewUnauthorizedError("account is not active")
	}

	// 5. 新しいトークンペアを生成（トークンローテーション）
	newAccessToken, newRefreshToken, err := c.jwtService.GenerateTokenPair(
		user.ID,
		user.Email.String(),
		session.ID,
	)
	if err != nil {
		return nil, apperror.NewInternalError(err)
	}

	// 6. セッションを更新
	now := time.Now()
	session.RefreshToken = newRefreshToken
	session.LastUsedAt = now
	session.ExpiresAt = now.Add(c.jwtService.GetRefreshTokenExpiry())

	if err := c.sessionRepo.Save(ctx, session); err != nil {
		return nil, apperror.NewInternalError(err)
	}

	// 7. 古いリフレッシュトークンのJTIをブラックリストに追加（オプション）
	if c.jwtBlacklist != nil && claims.ExpiresAt != nil {
		c.jwtBlacklist.Add(ctx, claims.ID, claims.ExpiresAt.Time)
	}

	return &RefreshTokenOutput{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(c.jwtService.GetAccessTokenExpiry().Seconds()),
	}, nil
}
