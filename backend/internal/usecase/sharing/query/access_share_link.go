package query

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// AccessShareLinkInput は共有リンクアクセスの入力を定義します
type AccessShareLinkInput struct {
	Token     string
	Password  string     // optional, required if password protected
	UserID    *uuid.UUID // optional, for logged-in users
	IPAddress string
	UserAgent string
	Action    string // view, download, upload
}

// AccessShareLinkOutput は共有リンクアクセスの出力を定義します
type AccessShareLinkOutput struct {
	ShareLink    *entity.ShareLink
	ResourceType string
	ResourceID   uuid.UUID
}

// AccessShareLinkQuery は共有リンクアクセスクエリです
type AccessShareLinkQuery struct {
	shareLinkRepo       repository.ShareLinkRepository
	shareLinkAccessRepo repository.ShareLinkAccessRepository
}

// NewAccessShareLinkQuery は新しいAccessShareLinkQueryを作成します
func NewAccessShareLinkQuery(
	shareLinkRepo repository.ShareLinkRepository,
	shareLinkAccessRepo repository.ShareLinkAccessRepository,
) *AccessShareLinkQuery {
	return &AccessShareLinkQuery{
		shareLinkRepo:       shareLinkRepo,
		shareLinkAccessRepo: shareLinkAccessRepo,
	}
}

// Execute は共有リンクアクセスを実行します
func (q *AccessShareLinkQuery) Execute(ctx context.Context, input AccessShareLinkInput) (*AccessShareLinkOutput, error) {
	// 1. トークンのバリデーション
	token, err := valueobject.ReconstructShareToken(input.Token)
	if err != nil {
		return nil, apperror.NewValidationError("invalid share link token", nil)
	}

	// 2. 共有リンクを取得
	shareLink, err := q.shareLinkRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// 3. アクセス可能か確認
	if err := shareLink.CanAccess(); err != nil {
		return nil, apperror.NewForbiddenError(err.Error())
	}

	// 4. アクションに応じた権限チェック
	action := entity.AccessAction(input.Action)
	if !action.IsValid() {
		return nil, apperror.NewValidationError("invalid action", nil)
	}

	// 5. パスワード確認（view以外のアクションで必要な場合）
	// view（情報取得）はパスワード不要で、hasPasswordフラグを返す
	if action != entity.AccessActionView && shareLink.RequiresPassword() {
		if input.Password == "" {
			return nil, apperror.NewUnauthorizedError("password is required")
		}
		comparePassword := func(hash, password string) error {
			return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		}
		if err := shareLink.ValidatePassword(input.Password, comparePassword); err != nil {
			return nil, apperror.NewUnauthorizedError("invalid password")
		}
	}

	// viewアクションはここで終了（アクセスカウントを増やさない）
	if action == entity.AccessActionView {
		return &AccessShareLinkOutput{
			ShareLink:    shareLink,
			ResourceType: shareLink.ResourceType.String(),
			ResourceID:   shareLink.ResourceID,
		}, nil
	}

	// 6. ダウンロード/アップロードの権限チェック
	if action == entity.AccessActionDownload && !shareLink.CanDownload() {
		return nil, apperror.NewForbiddenError("download is not allowed with this share link")
	}
	if action == entity.AccessActionUpload && !shareLink.CanUpload() {
		return nil, apperror.NewForbiddenError("upload is not allowed with this share link")
	}

	// 7. アクセスカウントを増やす
	shareLink.IncrementAccessCount()
	if err := q.shareLinkRepo.Update(ctx, shareLink); err != nil {
		return nil, err
	}

	// 8. アクセスログを記録
	access, err := entity.NewShareLinkAccess(
		shareLink.ID,
		input.IPAddress,
		input.UserAgent,
		input.UserID,
		action,
	)
	if err != nil {
		return nil, apperror.NewInternalError(err)
	}
	// Access log creation failure is intentionally ignored
	_ = q.shareLinkAccessRepo.Create(ctx, access)

	return &AccessShareLinkOutput{
		ShareLink:    shareLink,
		ResourceType: shareLink.ResourceType.String(),
		ResourceID:   shareLink.ResourceID,
	}, nil
}
