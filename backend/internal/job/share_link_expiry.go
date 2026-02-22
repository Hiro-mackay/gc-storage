package job

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// ShareLinkExpiryJob is a background job that marks expired share links.
type ShareLinkExpiryJob struct {
	shareLinkRepo repository.ShareLinkRepository
	interval      time.Duration
}

// NewShareLinkExpiryJob creates a new ShareLinkExpiryJob.
func NewShareLinkExpiryJob(shareLinkRepo repository.ShareLinkRepository) *ShareLinkExpiryJob {
	return &ShareLinkExpiryJob{
		shareLinkRepo: shareLinkRepo,
		interval:      time.Hour,
	}
}

// Start runs the expiry job on a ticker loop until context is cancelled.
func (j *ShareLinkExpiryJob) Start(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			expired, err := j.shareLinkRepo.FindExpired(ctx)
			if err != nil {
				slog.Error("share link expiry job: find expired failed", "error", err)
				continue
			}
			if len(expired) == 0 {
				continue
			}

			ids := make([]uuid.UUID, 0, len(expired))
			for _, link := range expired {
				ids = append(ids, link.ID)
			}

			count, err := j.shareLinkRepo.UpdateStatusBatch(ctx, ids, valueobject.ShareLinkStatusExpired)
			if err != nil {
				slog.Error("share link expiry job: update status failed", "error", err)
				continue
			}
			if count > 0 {
				slog.Info("expired share links", "count", count)
			}
		}
	}
}
