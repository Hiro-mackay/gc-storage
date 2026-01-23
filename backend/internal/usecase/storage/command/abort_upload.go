package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// AbortUploadInput はアップロード中断の入力を定義します
type AbortUploadInput struct {
	SessionID uuid.UUID
	UserID    uuid.UUID
}

// AbortUploadOutput はアップロード中断の出力を定義します
type AbortUploadOutput struct {
	SessionID uuid.UUID
	Aborted   bool
}

// AbortUploadCommand はアップロード中断コマンドです
type AbortUploadCommand struct {
	uploadSessionRepo repository.UploadSessionRepository
	fileRepo          repository.FileRepository
	storageService    service.StorageService
	txManager         repository.TransactionManager
}

// NewAbortUploadCommand は新しいAbortUploadCommandを作成します
func NewAbortUploadCommand(
	uploadSessionRepo repository.UploadSessionRepository,
	fileRepo repository.FileRepository,
	storageService service.StorageService,
	txManager repository.TransactionManager,
) *AbortUploadCommand {
	return &AbortUploadCommand{
		uploadSessionRepo: uploadSessionRepo,
		fileRepo:          fileRepo,
		storageService:    storageService,
		txManager:         txManager,
	}
}

// Execute はアップロード中断を実行します
func (c *AbortUploadCommand) Execute(ctx context.Context, input AbortUploadInput) (*AbortUploadOutput, error) {
	// 1. セッション取得
	session, err := c.uploadSessionRepo.FindByID(ctx, input.SessionID)
	if err != nil {
		return nil, err
	}

	// 2. 所有者チェック
	if !session.IsOwnedBy(input.UserID) {
		return nil, apperror.NewForbiddenError("not authorized to abort this upload")
	}

	// 3. 中断可能か確認
	if session.IsCompleted() {
		return nil, apperror.NewValidationError("upload session already completed", nil)
	}
	if session.IsAborted() {
		return nil, apperror.NewValidationError("upload session already aborted", nil)
	}

	// 4. トランザクションで中断処理
	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// セッションを中断
		if err := session.Abort(); err != nil {
			return err
		}
		if err := c.uploadSessionRepo.Update(ctx, session); err != nil {
			return err
		}

		// ファイルをupload_failedに
		file, err := c.fileRepo.FindByID(ctx, session.FileID)
		if err != nil {
			return err
		}
		if err := file.MarkUploadFailed(); err != nil {
			return err
		}
		return c.fileRepo.Update(ctx, file)
	})

	if err != nil {
		return nil, err
	}

	// 5. MinIOのマルチパートアップロードを中断（トランザクション外）
	if session.MinioUploadID != nil {
		_ = c.storageService.AbortMultipartUpload(ctx, session.StorageKey.String(), *session.MinioUploadID)
	}

	return &AbortUploadOutput{
		SessionID: session.ID,
		Aborted:   true,
	}, nil
}
