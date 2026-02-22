package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// CompleteUploadInput はアップロード完了の入力を定義します（Webhook用）
type CompleteUploadInput struct {
	StorageKey     string // MinIOオブジェクトキー（ファイルID）
	MinioVersionID string // MinIOバージョンID
	Size           int64
	ETag           string // チェックサム
}

// CompleteUploadOutput はアップロード完了の出力を定義します
type CompleteUploadOutput struct {
	FileID    uuid.UUID
	SessionID uuid.UUID
	Completed bool // 全パーツ完了したかどうか
}

// CompleteUploadCommand はアップロード完了コマンドです（MinIO Webhook用）
type CompleteUploadCommand struct {
	fileRepo          repository.FileRepository
	fileVersionRepo   repository.FileVersionRepository
	uploadSessionRepo repository.UploadSessionRepository
	uploadPartRepo    repository.UploadPartRepository
	txManager         repository.TransactionManager
}

// NewCompleteUploadCommand は新しいCompleteUploadCommandを作成します
func NewCompleteUploadCommand(
	fileRepo repository.FileRepository,
	fileVersionRepo repository.FileVersionRepository,
	uploadSessionRepo repository.UploadSessionRepository,
	uploadPartRepo repository.UploadPartRepository,
	txManager repository.TransactionManager,
) *CompleteUploadCommand {
	return &CompleteUploadCommand{
		fileRepo:          fileRepo,
		fileVersionRepo:   fileVersionRepo,
		uploadSessionRepo: uploadSessionRepo,
		uploadPartRepo:    uploadPartRepo,
		txManager:         txManager,
	}
}

// Execute はアップロード完了処理を実行します
func (c *CompleteUploadCommand) Execute(ctx context.Context, input CompleteUploadInput) (*CompleteUploadOutput, error) {
	// 1. StorageKeyからUploadSessionを検索
	storageKey, err := parseStorageKey(input.StorageKey)
	if err != nil {
		return nil, apperror.NewValidationError("invalid storage key", nil)
	}

	session, err := c.uploadSessionRepo.FindByStorageKey(ctx, storageKey)
	if err != nil {
		return nil, err
	}

	// 2. セッションの状態チェック
	if !session.CanAcceptUpload() {
		if session.IsCompleted() {
			// 既に完了している場合は冪等性のため成功を返す
			return &CompleteUploadOutput{
				FileID:    session.FileID,
				SessionID: session.ID,
				Completed: true,
			}, nil
		}
		return nil, apperror.NewValidationError("upload session cannot accept uploads", nil)
	}

	// 3. ファイル取得
	file, err := c.fileRepo.FindByID(ctx, session.FileID)
	if err != nil {
		return nil, err
	}

	// 4. マルチパートの場合はパーツを記録
	if session.IsMultipart {
		session.IncrementUploadedParts()

		// パーツ情報を記録
		part := entity.NewUploadPart(
			session.ID,
			session.UploadedParts, // 現在のパーツ番号
			input.Size,
			input.ETag,
		)

		err = c.uploadPartRepo.Create(ctx, part)
		if err != nil {
			return nil, err
		}

		// 全パーツ完了でない場合
		if !session.AllPartsUploaded() {
			if err := c.uploadSessionRepo.Update(ctx, session); err != nil {
				return nil, err
			}

			return &CompleteUploadOutput{
				FileID:    session.FileID,
				SessionID: session.ID,
				Completed: false,
			}, nil
		}
	}

	// 5. アップロード完了処理（トランザクション）
	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// ファイルバージョン作成
		version := entity.NewFileVersion(
			file.ID,
			file.CurrentVersion,
			input.MinioVersionID,
			input.Size,
			input.ETag,
			session.OwnerID,
		)

		if err := c.fileVersionRepo.Create(ctx, version); err != nil {
			return err
		}

		// ファイルを有効化
		if err := file.Activate(); err != nil {
			return err
		}
		file.UpdateSize(input.Size)

		if err := c.fileRepo.Update(ctx, file); err != nil {
			return err
		}

		// ステータスを永続化（UpdateFileクエリにstatusが含まれないため個別に更新）
		if err := c.fileRepo.UpdateStatus(ctx, file.ID, file.Status); err != nil {
			return err
		}

		// セッションを完了
		if err := session.Complete(); err != nil {
			return err
		}

		return c.uploadSessionRepo.Update(ctx, session)
	})

	if err != nil {
		return nil, err
	}

	return &CompleteUploadOutput{
		FileID:    session.FileID,
		SessionID: session.ID,
		Completed: true,
	}, nil
}

// parseStorageKey はストレージキー文字列をvalue objectに変換します
func parseStorageKey(key string) (valueobject.StorageKey, error) {
	return valueobject.NewStorageKeyFromString(key)
}
