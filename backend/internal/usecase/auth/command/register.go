package command

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// RegisterInput は登録の入力を定義します
type RegisterInput struct {
	Email     string
	Password  string
	Name      string
	UserAgent string
	IPAddress string
}

// RegisterOutput は登録の出力を定義します
type RegisterOutput struct {
	UserID    uuid.UUID
	SessionID string
	User      *entity.User
}

// RegisterCommand はユーザー登録コマンドです
type RegisterCommand struct {
	userRepo                   repository.UserRepository
	sessionRepo                repository.SessionRepository
	folderRepo                 repository.FolderRepository
	folderClosureRepo          repository.FolderClosureRepository
	relationshipRepo           authz.RelationshipRepository
	emailVerificationTokenRepo repository.EmailVerificationTokenRepository
	txManager                  repository.TransactionManager
	emailSender                service.EmailSender
	appURL                     string
}

// NewRegisterCommand は新しいRegisterCommandを作成します
func NewRegisterCommand(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	folderRepo repository.FolderRepository,
	folderClosureRepo repository.FolderClosureRepository,
	relationshipRepo authz.RelationshipRepository,
	emailVerificationTokenRepo repository.EmailVerificationTokenRepository,
	txManager repository.TransactionManager,
	emailSender service.EmailSender,
	appURL string,
) *RegisterCommand {
	return &RegisterCommand{
		userRepo:                   userRepo,
		sessionRepo:                sessionRepo,
		folderRepo:                 folderRepo,
		folderClosureRepo:          folderClosureRepo,
		relationshipRepo:           relationshipRepo,
		emailVerificationTokenRepo: emailVerificationTokenRepo,
		txManager:                  txManager,
		emailSender:                emailSender,
		appURL:                     appURL,
	}
}

// Execute はユーザー登録を実行します
func (c *RegisterCommand) Execute(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
	// 1. メールアドレスのバリデーション
	email, err := valueobject.NewEmail(input.Email)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 2. パスワードのバリデーション
	password, err := valueobject.NewPassword(input.Password, input.Email)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 3. メールアドレスの重複チェック
	exists, err := c.userRepo.Exists(ctx, email)
	if err != nil {
		return nil, apperror.NewInternalError(err)
	}
	if exists {
		return nil, apperror.NewConflictError("email already exists")
	}

	var user *entity.User
	var verificationToken *entity.EmailVerificationToken

	// 4. トランザクションでユーザー作成とPersonal Folder作成
	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// ユーザー作成
		user = entity.NewUser(email, input.Name, password.Hash())

		if err := c.userRepo.Create(ctx, user); err != nil {
			return err
		}

		// Personal Folder 作成
		folderNameStr := input.Name + "'s folder"
		folderName, err := valueobject.NewFolderName(folderNameStr)
		if err != nil {
			return fmt.Errorf("failed to create folder name: %w", err)
		}

		personalFolder, err := entity.NewFolder(folderName, nil, user.ID, 0)
		if err != nil {
			return fmt.Errorf("failed to create personal folder: %w", err)
		}

		if err := c.folderRepo.Create(ctx, personalFolder); err != nil {
			return fmt.Errorf("failed to save personal folder: %w", err)
		}

		// Closure Table 自己参照
		if err := c.folderClosureRepo.InsertSelfReference(ctx, personalFolder.ID); err != nil {
			return fmt.Errorf("failed to insert folder closure: %w", err)
		}

		// オーナーリレーションシップを作成 (user --owner--> folder)
		ownerRelation := authz.NewOwnerRelationship(user.ID, authz.ObjectTypeFolder, personalFolder.ID)
		if err := c.relationshipRepo.Create(ctx, ownerRelation); err != nil {
			return fmt.Errorf("failed to create owner relationship: %w", err)
		}

		// User に personal_folder_id を設定
		if err := c.userRepo.SetPersonalFolderID(ctx, user.ID, personalFolder.ID); err != nil {
			return fmt.Errorf("failed to update user with personal folder: %w", err)
		}
		user.SetPersonalFolder(personalFolder.ID)

		// 確認トークン作成（emailVerificationTokenRepoが設定されている場合のみ）
		if c.emailVerificationTokenRepo != nil {
			now := time.Now()
			verificationToken = &entity.EmailVerificationToken{
				ID:        uuid.New(),
				UserID:    user.ID,
				Token:     generateSecureToken(),
				ExpiresAt: now.Add(24 * time.Hour),
				CreatedAt: now,
			}

			if err := c.emailVerificationTokenRepo.Create(ctx, verificationToken); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, apperror.NewInternalError(err)
	}

	// 5. セッション作成（自動ログイン）
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

	// 6. 確認メール送信（トランザクション外で実行、失敗しても登録は成功扱い）
	if c.emailSender != nil && verificationToken != nil {
		verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", c.appURL, verificationToken.Token)
		if err := c.emailSender.SendEmailVerification(ctx, user.Email.String(), user.Name, verifyURL); err != nil {
			// メール送信失敗はログに記録するが、エラーは返さない
			slog.Error("failed to send verification email", "error", err, "user_id", user.ID)
		}
	}

	return &RegisterOutput{
		UserID:    user.ID,
		SessionID: sessionID,
		User:      user,
	}, nil
}

// generateSecureToken はセキュアなトークンを生成します
func generateSecureToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
