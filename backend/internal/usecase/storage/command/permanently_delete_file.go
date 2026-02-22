package command

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// PermanentlyDeleteFileInput はファイル完全削除の入力を定義します
type PermanentlyDeleteFileInput struct {
	ArchivedFileID uuid.UUID
	UserID         uuid.UUID
}

// PermanentlyDeleteFileCommand はファイル完全削除コマンドです
type PermanentlyDeleteFileCommand struct {
	archivedFileRepo        repository.ArchivedFileRepository
	archivedFileVersionRepo repository.ArchivedFileVersionRepository
	storageService          service.StorageService
	txManager               repository.TransactionManager
}

// NewPermanentlyDeleteFileCommand は新しいPermanentlyDeleteFileCommandを作成します
func NewPermanentlyDeleteFileCommand(
	archivedFileRepo repository.ArchivedFileRepository,
	archivedFileVersionRepo repository.ArchivedFileVersionRepository,
	storageService service.StorageService,
	txManager repository.TransactionManager,
) *PermanentlyDeleteFileCommand {
	return &PermanentlyDeleteFileCommand{
		archivedFileRepo:        archivedFileRepo,
		archivedFileVersionRepo: archivedFileVersionRepo,
		storageService:          storageService,
		txManager:               txManager,
	}
}

// Execute はファイル完全削除を実行します
func (c *PermanentlyDeleteFileCommand) Execute(ctx context.Context, input PermanentlyDeleteFileInput) error {
	// 1. アーカイブファイル取得
	archivedFile, err := c.archivedFileRepo.FindByID(ctx, input.ArchivedFileID)
	if err != nil {
		return err
	}

	// 2. 所有者チェック
	if !archivedFile.IsOwnedBy(input.UserID) {
		return apperror.NewForbiddenError("not authorized to delete this file")
	}

	// ストレージキーを事前に保存（トランザクション後に使用）
	storageKey := archivedFile.StorageKey.String()

	// 3. トランザクションでDB削除
	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// バージョン削除
		if err := c.archivedFileVersionRepo.DeleteByArchivedFileID(ctx, archivedFile.ID); err != nil {
			return err
		}
		// ファイル削除
		return c.archivedFileRepo.Delete(ctx, archivedFile.ID)
	})

	if err != nil {
		return err
	}

	// 4. MinIOからオブジェクト削除（トランザクション外）
	if err := c.storageService.DeleteObject(ctx, storageKey); err != nil {
		slog.Error("failed to delete storage object",
			"storage_key", storageKey,
			"error", err,
		)
	}

	return nil
}
