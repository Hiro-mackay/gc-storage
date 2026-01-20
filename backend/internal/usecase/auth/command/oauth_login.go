package command

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/jwt"
)

// OAuthLoginInput はOAuthログインの入力を定義します
type OAuthLoginInput struct {
	Provider  string
	Code      string
	UserAgent string
	IPAddress string
}

// OAuthLoginOutput はOAuthログインの出力を定義します
type OAuthLoginOutput struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int // seconds
	User         *entity.User
	IsNewUser    bool
}

// OAuthLoginCommand はOAuthログインコマンドです
type OAuthLoginCommand struct {
	userRepo         repository.UserRepository
	profileRepo      repository.UserProfileRepository
	oauthAccountRepo repository.OAuthAccountRepository
	oauthFactory     service.OAuthClientFactory
	txManager        *database.TxManager
	sessionRepo      repository.SessionRepository
	jwtService       *jwt.JWTService
}

// NewOAuthLoginCommand は新しいOAuthLoginCommandを作成します
func NewOAuthLoginCommand(
	userRepo repository.UserRepository,
	profileRepo repository.UserProfileRepository,
	oauthAccountRepo repository.OAuthAccountRepository,
	oauthFactory service.OAuthClientFactory,
	txManager *database.TxManager,
	sessionRepo repository.SessionRepository,
	jwtService *jwt.JWTService,
) *OAuthLoginCommand {
	return &OAuthLoginCommand{
		userRepo:         userRepo,
		profileRepo:      profileRepo,
		oauthAccountRepo: oauthAccountRepo,
		oauthFactory:     oauthFactory,
		txManager:        txManager,
		sessionRepo:      sessionRepo,
		jwtService:       jwtService,
	}
}

// Execute はOAuthログインを実行します
func (c *OAuthLoginCommand) Execute(ctx context.Context, input OAuthLoginInput) (*OAuthLoginOutput, error) {
	// 1. プロバイダーの検証
	provider := valueobject.OAuthProvider(input.Provider)
	if !provider.IsValid() {
		return nil, apperror.NewValidationError("unsupported oauth provider", nil)
	}

	// 2. OAuthクライアントの取得
	oauthClient, err := c.oauthFactory.GetClient(provider)
	if err != nil {
		return nil, apperror.NewValidationError("unsupported oauth provider", nil)
	}

	// 3. 認可コードをトークンに交換
	tokens, err := oauthClient.ExchangeCode(ctx, input.Code)
	if err != nil {
		return nil, apperror.NewValidationError("invalid authorization code", nil)
	}

	// 4. ユーザー情報の取得
	userInfo, err := oauthClient.GetUserInfo(ctx, tokens.AccessToken)
	if err != nil {
		return nil, apperror.NewInternalError(err)
	}

	// 5. トランザクション内でユーザー処理
	var user *entity.User
	var oauthAccount *entity.OAuthAccount
	var isNewUser bool

	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		var txErr error

		// 5a. OAuthアカウントで検索
		oauthAccount, txErr = c.oauthAccountRepo.FindByProviderAndUserID(ctx, provider, userInfo.ProviderUserID)
		if txErr == nil {
			// 既存のOAuthアカウントがある場合、ユーザーを取得
			user, txErr = c.userRepo.FindByID(ctx, oauthAccount.UserID)
			if txErr != nil {
				return txErr
			}

			// トークンを更新
			oauthAccount.AccessToken = tokens.AccessToken
			oauthAccount.RefreshToken = tokens.RefreshToken
			if tokens.ExpiresIn > 0 {
				oauthAccount.TokenExpiresAt = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
			}
			oauthAccount.UpdatedAt = time.Now()
			if txErr = c.oauthAccountRepo.Update(ctx, oauthAccount); txErr != nil {
				return txErr
			}

			isNewUser = false
			return nil
		}

		// 5b. メールアドレスでユーザーを検索
		email, txErr := valueobject.NewEmail(userInfo.Email)
		if txErr != nil {
			return apperror.NewValidationError("invalid email from oauth provider", nil)
		}

		user, txErr = c.userRepo.FindByEmail(ctx, email)
		if txErr == nil {
			// 既存ユーザーがいる場合、OAuthアカウントを紐付け
			oauthAccount = &entity.OAuthAccount{
				ID:             uuid.New(),
				UserID:         user.ID,
				Provider:       provider,
				ProviderUserID: userInfo.ProviderUserID,
				Email:          userInfo.Email,
				AccessToken:    tokens.AccessToken,
				RefreshToken:   tokens.RefreshToken,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			if tokens.ExpiresIn > 0 {
				oauthAccount.TokenExpiresAt = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
			}

			if txErr = c.oauthAccountRepo.Create(ctx, oauthAccount); txErr != nil {
				return txErr
			}

			// ユーザーがpending状態の場合、activeに変更（OAuthはメール確認済みとみなす）
			if user.Status == entity.UserStatusPending {
				user.Status = entity.UserStatusActive
				user.EmailVerified = true
				user.UpdatedAt = time.Now()
				if txErr = c.userRepo.Update(ctx, user); txErr != nil {
					return txErr
				}
			}

			isNewUser = false
			return nil
		}

		// 5c. 新規ユーザーを作成
		now := time.Now()
		user = &entity.User{
			ID:            uuid.New(),
			Email:         email,
			Name:          userInfo.Name,
			PasswordHash:  "", // OAuthユーザーはパスワードなし
			Status:        entity.UserStatusActive,
			EmailVerified: true, // OAuthはメール確認済みとみなす
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if txErr = c.userRepo.Create(ctx, user); txErr != nil {
			return txErr
		}

		// UserProfileを作成（AvatarURLを含む）
		profile := entity.NewUserProfile(user.ID)
		profile.AvatarURL = userInfo.AvatarURL
		if txErr = c.profileRepo.Upsert(ctx, profile); txErr != nil {
			return txErr
		}

		// OAuthアカウントを作成
		oauthAccount = &entity.OAuthAccount{
			ID:             uuid.New(),
			UserID:         user.ID,
			Provider:       provider,
			ProviderUserID: userInfo.ProviderUserID,
			Email:          userInfo.Email,
			AccessToken:    tokens.AccessToken,
			RefreshToken:   tokens.RefreshToken,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if tokens.ExpiresIn > 0 {
			oauthAccount.TokenExpiresAt = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
		}

		if txErr = c.oauthAccountRepo.Create(ctx, oauthAccount); txErr != nil {
			return txErr
		}

		isNewUser = true
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 6. ユーザー状態チェック
	if user.Status != entity.UserStatusActive {
		switch user.Status {
		case entity.UserStatusSuspended:
			return nil, apperror.NewUnauthorizedError("account suspended")
		case entity.UserStatusDeactivated:
			return nil, apperror.NewUnauthorizedError("account deactivated")
		default:
			return nil, apperror.NewUnauthorizedError("account is not active")
		}
	}

	// 7. セッション作成
	sessionID := uuid.New().String()
	now := time.Now()
	expiresAt := now.Add(c.jwtService.GetRefreshTokenExpiry())

	accessToken, refreshToken, err := c.jwtService.GenerateTokenPair(user.ID, user.Email.String(), sessionID)
	if err != nil {
		return nil, apperror.NewInternalError(err)
	}

	session := &entity.Session{
		ID:           sessionID,
		UserID:       user.ID,
		RefreshToken: refreshToken,
		UserAgent:    input.UserAgent,
		IPAddress:    input.IPAddress,
		ExpiresAt:    expiresAt,
		CreatedAt:    now,
		LastUsedAt:   now,
	}

	if err := c.sessionRepo.Save(ctx, session); err != nil {
		return nil, apperror.NewInternalError(err)
	}

	return &OAuthLoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(c.jwtService.GetAccessTokenExpiry().Seconds()),
		User:         user,
		IsNewUser:    isNewUser,
	}, nil
}
