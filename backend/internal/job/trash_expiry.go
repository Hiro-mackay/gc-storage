package job

import (
	"context"
	"log/slog"
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
)

const trashExpiryChunkSize = 100

// TrashExpiryJob is a background job that permanently deletes expired archived files.
type TrashExpiryJob struct {
	archivedFileRepo        repository.ArchivedFileRepository
	archivedFileVersionRepo repository.ArchivedFileVersionRepository
	storageService          service.StorageService
	txManager               repository.TransactionManager
	interval                time.Duration
}

// NewTrashExpiryJob creates a new TrashExpiryJob.
func NewTrashExpiryJob(
	archivedFileRepo repository.ArchivedFileRepository,
	archivedFileVersionRepo repository.ArchivedFileVersionRepository,
	storageService service.StorageService,
	txManager repository.TransactionManager,
) *TrashExpiryJob {
	return &TrashExpiryJob{
		archivedFileRepo:        archivedFileRepo,
		archivedFileVersionRepo: archivedFileVersionRepo,
		storageService:          storageService,
		txManager:               txManager,
		interval:                24 * time.Hour,
	}
}

// Start runs the expiry job on a daily ticker loop until context is cancelled.
func (j *TrashExpiryJob) Start(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.run(ctx)
		}
	}
}

func (j *TrashExpiryJob) run(ctx context.Context) {
	expired, err := j.archivedFileRepo.FindExpired(ctx)
	if err != nil {
		slog.Error("trash expiry job: find expired failed", "error", err)
		return
	}
	if len(expired) == 0 {
		return
	}

	storageKeysToDelete := make([]string, 0, len(expired))

	for i := 0; i < len(expired); i += trashExpiryChunkSize {
		end := i + trashExpiryChunkSize
		if end > len(expired) {
			end = len(expired)
		}
		chunk := expired[i:end]

		err = j.txManager.WithTransaction(ctx, func(ctx context.Context) error {
			for _, af := range chunk {
				if err := j.archivedFileVersionRepo.DeleteByArchivedFileID(ctx, af.ID); err != nil {
					return err
				}
				if err := j.archivedFileRepo.Delete(ctx, af.ID); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			slog.Error("trash expiry job: delete chunk failed", "error", err)
			continue
		}

		for _, af := range chunk {
			storageKeysToDelete = append(storageKeysToDelete, af.StorageKey.String())
		}
	}

	if len(storageKeysToDelete) > 0 {
		if err := j.storageService.DeleteObjects(ctx, storageKeysToDelete); err != nil {
			slog.Error("trash expiry job: storage delete failed", "error", err, "count", len(storageKeysToDelete))
		} else {
			slog.Info("trash expiry job: deleted expired files", "count", len(storageKeysToDelete))
		}
	}
}
