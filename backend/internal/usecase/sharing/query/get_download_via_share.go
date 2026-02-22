package query

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// DownloadPresignedURLExpiry はダウンロード用PresignedURLの有効期限
const DownloadPresignedURLExpiry = 15 * time.Minute

// GetDownloadViaShareInput は共有リンク経由ダウンロードの入力を定義します
type GetDownloadViaShareInput struct {
	Token     string
	Password  string     // optional
	FileID    *uuid.UUID // required for folder shares, ignored for file shares
	UserID    *uuid.UUID // optional
	IPAddress string
	UserAgent string
}

// GetDownloadViaShareOutput は共有リンク経由ダウンロードの出力を定義します
type GetDownloadViaShareOutput struct {
	PresignedURL string
	FileName     string
	FileSize     int64
	MimeType     string
	ExpiresAt    time.Time
}

// GetDownloadViaShareQuery は共有リンク経由ダウンロードクエリです
type GetDownloadViaShareQuery struct {
	shareLinkRepo       repository.ShareLinkRepository
	shareLinkAccessRepo repository.ShareLinkAccessRepository
	fileRepo            repository.FileRepository
	fileVersionRepo     repository.FileVersionRepository
	folderClosureRepo   repository.FolderClosureRepository
	storageService      service.StorageService
}

// NewGetDownloadViaShareQuery は新しいGetDownloadViaShareQueryを作成します
func NewGetDownloadViaShareQuery(
	shareLinkRepo repository.ShareLinkRepository,
	shareLinkAccessRepo repository.ShareLinkAccessRepository,
	fileRepo repository.FileRepository,
	fileVersionRepo repository.FileVersionRepository,
	folderClosureRepo repository.FolderClosureRepository,
	storageService service.StorageService,
) *GetDownloadViaShareQuery {
	return &GetDownloadViaShareQuery{
		shareLinkRepo:       shareLinkRepo,
		shareLinkAccessRepo: shareLinkAccessRepo,
		fileRepo:            fileRepo,
		fileVersionRepo:     fileVersionRepo,
		folderClosureRepo:   folderClosureRepo,
		storageService:      storageService,
	}
}

// Execute は共有リンク経由ダウンロードを実行します
func (q *GetDownloadViaShareQuery) Execute(ctx context.Context, input GetDownloadViaShareInput) (*GetDownloadViaShareOutput, error) {
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
		if errors.Is(err, entity.ErrShareLinkExpired) || errors.Is(err, entity.ErrShareLinkRevoked) || errors.Is(err, entity.ErrShareLinkMaxAccessReached) {
			return nil, apperror.NewGoneError(err.Error())
		}
		return nil, apperror.NewForbiddenError(err.Error())
	}

	// 4. ダウンロード権限チェック
	if !shareLink.CanDownload() {
		return nil, apperror.NewForbiddenError("download is not allowed with this share link")
	}

	// 5. パスワード確認（必要な場合）
	if shareLink.RequiresPassword() {
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

	// 6. リソースタイプに応じてファイルを解決
	var targetFileID uuid.UUID
	if shareLink.ResourceType == authz.ResourceTypeFile {
		targetFileID = shareLink.ResourceID
	} else {
		// フォルダ共有の場合はFileIDが必須
		if input.FileID == nil {
			return nil, apperror.NewValidationError("file_id is required for folder share links", nil)
		}
		targetFileID = *input.FileID

		// ファイルがフォルダのサブツリーに属することを確認
		if err := q.verifyFileInFolderSubtree(ctx, shareLink.ResourceID, targetFileID); err != nil {
			return nil, err
		}
	}

	// 7. ファイルを取得
	file, err := q.fileRepo.FindByID(ctx, targetFileID)
	if err != nil {
		return nil, err
	}

	if !file.CanDownload() {
		return nil, apperror.NewForbiddenError("file is not available for download")
	}

	// 8. 最新バージョンを取得
	_, err = q.fileVersionRepo.FindLatestByFileID(ctx, file.ID)
	if err != nil {
		return nil, err
	}

	// 9. PresignedURLを生成
	presignedURL, err := q.storageService.GenerateGetURL(ctx, file.StorageKey.String(), DownloadPresignedURLExpiry)
	if err != nil {
		return nil, apperror.NewInternalError(err)
	}

	// 10. アクセスカウントを増やす
	shareLink.IncrementAccessCount()
	if err := q.shareLinkRepo.Update(ctx, shareLink); err != nil {
		return nil, err
	}

	// 11. アクセスログを記録（失敗は無視）
	access, err := entity.NewShareLinkAccess(
		shareLink.ID,
		input.IPAddress,
		input.UserAgent,
		input.UserID,
		entity.AccessActionDownload,
	)
	if err == nil {
		_ = q.shareLinkAccessRepo.Create(ctx, access)
	}

	return &GetDownloadViaShareOutput{
		PresignedURL: presignedURL.URL,
		FileName:     file.Name.String(),
		FileSize:     file.Size,
		MimeType:     file.MimeType.String(),
		ExpiresAt:    presignedURL.ExpiresAt,
	}, nil
}

// verifyFileInFolderSubtree はファイルが共有フォルダのサブツリーに属することを確認します
func (q *GetDownloadViaShareQuery) verifyFileInFolderSubtree(ctx context.Context, folderID, fileID uuid.UUID) error {
	// フォルダのすべての子孫IDを取得（自身を含む）
	descendantIDs, err := q.folderClosureRepo.FindDescendantIDs(ctx, folderID)
	if err != nil {
		return err
	}

	// ファイルを取得してフォルダIDを確認
	file, err := q.fileRepo.FindByID(ctx, fileID)
	if err != nil {
		return err
	}

	// ファイルのフォルダIDがサブツリーに含まれるか確認
	for _, id := range descendantIDs {
		if file.FolderID == id {
			return nil
		}
	}

	return apperror.NewNotFoundError("file not found in shared folder")
}
