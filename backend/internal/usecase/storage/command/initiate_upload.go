package command

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// InitiateUploadInput はアップロード開始の入力を定義します
type InitiateUploadInput struct {
	FolderID uuid.UUID // 必須 - アップロード先フォルダID
	FileName string
	MimeType string
	Size     int64
	OwnerID  uuid.UUID // 作成者のユーザーID
}

// UploadURL はアップロードURL情報を表します
type UploadURL struct {
	PartNumber int
	URL        string
	ExpiresAt  time.Time
}

// InitiateUploadOutput はアップロード開始の出力を定義します
type InitiateUploadOutput struct {
	SessionID   uuid.UUID
	FileID      uuid.UUID
	IsMultipart bool
	UploadURLs  []UploadURL
	ExpiresAt   time.Time
}

// InitiateUploadCommand はアップロード開始コマンドです
type InitiateUploadCommand struct {
	fileRepo          repository.FileRepository
	folderRepo        repository.FolderRepository
	uploadSessionRepo repository.UploadSessionRepository
	storageService    service.StorageService
	txManager         repository.TransactionManager
}

// NewInitiateUploadCommand は新しいInitiateUploadCommandを作成します
func NewInitiateUploadCommand(
	fileRepo repository.FileRepository,
	folderRepo repository.FolderRepository,
	uploadSessionRepo repository.UploadSessionRepository,
	storageService service.StorageService,
	txManager repository.TransactionManager,
) *InitiateUploadCommand {
	return &InitiateUploadCommand{
		fileRepo:          fileRepo,
		folderRepo:        folderRepo,
		uploadSessionRepo: uploadSessionRepo,
		storageService:    storageService,
		txManager:         txManager,
	}
}

// Execute はアップロード開始を実行します
func (c *InitiateUploadCommand) Execute(ctx context.Context, input InitiateUploadInput) (*InitiateUploadOutput, error) {
	// 1. ファイル名のバリデーション
	fileName, err := valueobject.NewFileName(input.FileName)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 2. MIMEタイプのバリデーション
	mimeType, err := valueobject.NewMimeType(input.MimeType)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 3. フォルダの存在と権限チェック（FolderIDは必須）
	folder, err := c.folderRepo.FindByID(ctx, input.FolderID)
	if err != nil {
		return nil, err
	}

	// 所有者チェック
	if !folder.IsOwnedBy(input.OwnerID) {
		return nil, apperror.NewForbiddenError("not authorized to upload to this folder")
	}

	// 4. 同名ファイルの存在チェック
	exists, err := c.fileRepo.ExistsByNameAndFolder(ctx, fileName, input.FolderID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.NewConflictError("file with same name already exists")
	}

	// 5. ファイルIDを生成（File と UploadSession で共有）
	fileID := uuid.New()

	// 6. マルチパートアップロードの判定
	isMultipart := input.Size >= entity.MultipartThreshold
	var minioUploadID *string

	// 7. マルチパートの場合はMinIOでアップロード開始
	if isMultipart {
		storageKey := valueobject.NewStorageKey(fileID).String()
		uploadID, err := c.storageService.CreateMultipartUpload(ctx, storageKey)
		if err != nil {
			return nil, apperror.NewInternalError(err)
		}
		minioUploadID = &uploadID
	}

	// 8. File と UploadSession を作成
	// 新規作成時は owner_id = created_by = 作成者（input.OwnerID）となる
	file := entity.NewFileWithID(
		fileID,
		input.FolderID,
		input.OwnerID, // createdBy - owner_id = created_by = 作成者
		fileName,
		mimeType,
		input.Size,
	)

	session := entity.NewUploadSession(
		fileID,
		input.OwnerID, // createdBy - owner_id = created_by = 作成者
		input.FolderID,
		fileName,
		mimeType,
		input.Size,
		minioUploadID,
	)

	// 9. トランザクションで保存
	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		if err := c.fileRepo.Create(ctx, file); err != nil {
			return err
		}
		return c.uploadSessionRepo.Create(ctx, session)
	})

	if err != nil {
		// MinIOのマルチパートアップロードをキャンセル
		if minioUploadID != nil {
			_ = c.storageService.AbortMultipartUpload(ctx, file.StorageKey.String(), *minioUploadID)
		}
		return nil, err
	}

	// 10. Presigned URL を生成
	uploadURLs := make([]UploadURL, 0, session.TotalParts)
	expiry := time.Until(session.ExpiresAt)

	if isMultipart {
		// マルチパートの場合は各パートのURLを生成
		for i := 1; i <= session.TotalParts; i++ {
			partURL, err := c.storageService.GeneratePartUploadURL(ctx, file.StorageKey.String(), *minioUploadID, i)
			if err != nil {
				return nil, apperror.NewInternalError(err)
			}
			uploadURLs = append(uploadURLs, UploadURL{
				PartNumber: i,
				URL:        partURL.URL,
				ExpiresAt:  partURL.ExpiresAt,
			})
		}
	} else {
		// シングルパートの場合
		putURL, err := c.storageService.GeneratePutURL(ctx, file.StorageKey.String(), expiry)
		if err != nil {
			return nil, apperror.NewInternalError(err)
		}
		uploadURLs = append(uploadURLs, UploadURL{
			PartNumber: 1,
			URL:        putURL.URL,
			ExpiresAt:  putURL.ExpiresAt,
		})
	}

	return &InitiateUploadOutput{
		SessionID:   session.ID,
		FileID:      file.ID,
		IsMultipart: isMultipart,
		UploadURLs:  uploadURLs,
		ExpiresAt:   session.ExpiresAt,
	}, nil
}
