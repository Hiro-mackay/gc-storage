package query

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
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

// FolderContent はフォルダ内コンテンツを表します
type FolderContent struct {
	ID       uuid.UUID
	Name     string
	Type     string  // "file" or "folder"
	MimeType *string // file only
	Size     *int64  // file only
}

// AccessShareLinkOutput は共有リンクアクセスの出力を定義します
type AccessShareLinkOutput struct {
	ShareLink    *entity.ShareLink
	ResourceType string
	ResourceID   uuid.UUID
	ResourceName string
	PresignedURL *string         // file shares only
	Contents     []FolderContent // folder shares only
}

// AccessShareLinkQuery は共有リンクアクセスクエリです
type AccessShareLinkQuery struct {
	shareLinkRepo       repository.ShareLinkRepository
	shareLinkAccessRepo repository.ShareLinkAccessRepository
	fileRepo            repository.FileRepository
	fileVersionRepo     repository.FileVersionRepository
	folderRepo          repository.FolderRepository
	storageService      service.StorageService
}

// NewAccessShareLinkQuery は新しいAccessShareLinkQueryを作成します
func NewAccessShareLinkQuery(
	shareLinkRepo repository.ShareLinkRepository,
	shareLinkAccessRepo repository.ShareLinkAccessRepository,
	fileRepo repository.FileRepository,
	fileVersionRepo repository.FileVersionRepository,
	folderRepo repository.FolderRepository,
	storageService service.StorageService,
) *AccessShareLinkQuery {
	return &AccessShareLinkQuery{
		shareLinkRepo:       shareLinkRepo,
		shareLinkAccessRepo: shareLinkAccessRepo,
		fileRepo:            fileRepo,
		fileVersionRepo:     fileVersionRepo,
		folderRepo:          folderRepo,
		storageService:      storageService,
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
		if errors.Is(err, entity.ErrShareLinkExpired) || errors.Is(err, entity.ErrShareLinkRevoked) || errors.Is(err, entity.ErrShareLinkMaxAccessReached) {
			return nil, apperror.NewGoneError(err.Error())
		}
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

	// 6. viewアクションはリソース名のみ返す（PresignedURL生成しない、アクセスカウントを増やさない）
	if action == entity.AccessActionView {
		resourceName, err := q.fetchResourceName(ctx, shareLink)
		if err != nil {
			return nil, err
		}
		return &AccessShareLinkOutput{
			ShareLink:    shareLink,
			ResourceType: shareLink.ResourceType.String(),
			ResourceID:   shareLink.ResourceID,
			ResourceName: resourceName,
		}, nil
	}

	// 7. リソース情報を取得（ファイルの場合はPresignedURLも生成）
	resourceName, presignedURL, contents, err := q.fetchResourceInfo(ctx, shareLink)
	if err != nil {
		return nil, err
	}

	// 8. ダウンロード/アップロードの権限チェック
	if action == entity.AccessActionDownload && !shareLink.CanDownload() {
		return nil, apperror.NewForbiddenError("download is not allowed with this share link")
	}
	if action == entity.AccessActionUpload && !shareLink.CanUpload() {
		return nil, apperror.NewForbiddenError("upload is not allowed with this share link")
	}

	// 9. アクセスカウントを増やす
	shareLink.IncrementAccessCount()
	if err := q.shareLinkRepo.Update(ctx, shareLink); err != nil {
		return nil, err
	}

	// 10. アクセスログを記録
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
		ResourceName: resourceName,
		PresignedURL: presignedURL,
		Contents:     contents,
	}, nil
}

// fetchResourceName はリソース名のみを取得します（view用）
func (q *AccessShareLinkQuery) fetchResourceName(ctx context.Context, shareLink *entity.ShareLink) (string, error) {
	if shareLink.ResourceType == authz.ResourceTypeFile {
		file, err := q.fileRepo.FindByID(ctx, shareLink.ResourceID)
		if err != nil {
			return "", err
		}
		return file.Name.String(), nil
	}

	folder, err := q.folderRepo.FindByID(ctx, shareLink.ResourceID)
	if err != nil {
		return "", err
	}
	return folder.Name.String(), nil
}

// fetchResourceInfo はリソース情報を取得します
// ファイルの場合はPresignedURLも生成して返します
func (q *AccessShareLinkQuery) fetchResourceInfo(ctx context.Context, shareLink *entity.ShareLink) (string, *string, []FolderContent, error) {
	if shareLink.ResourceType == authz.ResourceTypeFile {
		file, err := q.fileRepo.FindByID(ctx, shareLink.ResourceID)
		if err != nil {
			return "", nil, nil, err
		}

		// 最新バージョンを取得してPresignedURLを生成
		_, err = q.fileVersionRepo.FindLatestByFileID(ctx, file.ID)
		if err != nil {
			return "", nil, nil, err
		}

		presigned, err := q.storageService.GenerateGetURL(ctx, file.StorageKey.String(), DownloadPresignedURLExpiry)
		if err != nil {
			return "", nil, nil, apperror.NewInternalError(err)
		}

		url := presigned.URL
		return file.Name.String(), &url, nil, nil
	}

	folder, err := q.folderRepo.FindByID(ctx, shareLink.ResourceID)
	if err != nil {
		return "", nil, nil, err
	}

	files, err := q.fileRepo.FindByFolderID(ctx, folder.ID)
	if err != nil {
		return "", nil, nil, err
	}

	contents := make([]FolderContent, 0, len(files))
	for _, f := range files {
		mime := f.MimeType.String()
		size := f.Size
		contents = append(contents, FolderContent{
			ID:       f.ID,
			Name:     f.Name.String(),
			Type:     "file",
			MimeType: &mime,
			Size:     &size,
		})
	}

	return folder.Name.String(), nil, contents, nil
}
