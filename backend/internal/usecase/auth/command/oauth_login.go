package command

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
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
	SessionID string
	User      *entity.User
	IsNewUser bool
}

// OAuthLoginCommand はOAuthログインコマンドです
type OAuthLoginCommand struct {
	userRepo          repository.UserRepository
	profileRepo       repository.UserProfileRepository
	oauthAccountRepo  repository.OAuthAccountRepository
	folderRepo        repository.FolderRepository
	folderClosureRepo repository.FolderClosureRepository
	relationshipRepo  authz.RelationshipRepository
	oauthFactory      service.OAuthClientFactory
	txManager         *database.TxManager
	sessionRepo       repository.SessionRepository
}

// NewOAuthLoginCommand は新しいOAuthLoginCommandを作成します
func NewOAuthLoginCommand(
	userRepo repository.UserRepository,
	profileRepo repository.UserProfileRepository,
	oauthAccountRepo repository.OAuthAccountRepository,
	folderRepo repository.FolderRepository,
	folderClosureRepo repository.FolderClosureRepository,
	relationshipRepo authz.RelationshipRepository,
	oauthFactory service.OAuthClientFactory,
	txManager *database.TxManager,
	sessionRepo repository.SessionRepository,
) *OAuthLoginCommand {
	return &OAuthLoginCommand{
		userRepo:          userRepo,
		profileRepo:       profileRepo,
		oauthAccountRepo:  oauthAccountRepo,
		folderRepo:        folderRepo,
		folderClosureRepo: folderClosureRepo,
		relationshipRepo:  relationshipRepo,
		oauthFactory:      oauthFactory,
		txManager:         txManager,
		sessionRepo:       sessionRepo,
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

		// Personal Folder を作成
		folderNameStr := userInfo.Name + "'s folder"
		folderName, txErr := valueobject.NewFolderName(folderNameStr)
		if txErr != nil {
			return txErr
		}

		personalFolder, txErr := entity.NewFolder(folderName, nil, user.ID, 0)
		if txErr != nil {
			return txErr
		}

		if txErr = c.folderRepo.Create(ctx, personalFolder); txErr != nil {
			return txErr
		}

		// Closure Table 自己参照
		if txErr = c.folderClosureRepo.InsertSelfReference(ctx, personalFolder.ID); txErr != nil {
			return txErr
		}

		// オーナーリレーションシップを作成 (user --owner--> folder)
		ownerRelation := authz.NewOwnerRelationship(user.ID, authz.ObjectTypeFolder, personalFolder.ID)
		if txErr = c.relationshipRepo.Create(ctx, ownerRelation); txErr != nil {
			return txErr
		}

		// User に personal_folder_id を設定
		if txErr = c.userRepo.SetPersonalFolderID(ctx, user.ID, personalFolder.ID); txErr != nil {
			return txErr
		}
		user.SetPersonalFolder(personalFolder.ID)

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

	// 7. セッション制限チェック (R-SS002)
	sessionCount, err := c.sessionRepo.CountByUserID(ctx, user.ID)
	if err != nil {
		return nil, apperror.NewInternalError(err)
	}

	// 最大セッション数に達している場合は最古のセッションを削除
	if sessionCount >= int64(entity.MaxActiveSessionsPerUser) {
		if err := c.sessionRepo.DeleteOldestByUserID(ctx, user.ID); err != nil {
			return nil, apperror.NewInternalError(err)
		}
	}

	// 8. セッション作成
	sessionID := uuid.New().String()
	now := time.Now()

	session := &entity.Session{
		ID:         sessionID,
		UserID:     user.ID,
		UserAgent:  input.UserAgent,
		IPAddress:  input.IPAddress,
		ExpiresAt:  now.Add(entity.SessionTTL),
		CreatedAt:  now,
		LastUsedAt: now,
	}

	if err := c.sessionRepo.Save(ctx, session); err != nil {
		return nil, apperror.NewInternalError(err)
	}

	return &OAuthLoginOutput{
		SessionID: sessionID,
		User:      user,
		IsNewUser: isNewUser,
	}, nil
}
