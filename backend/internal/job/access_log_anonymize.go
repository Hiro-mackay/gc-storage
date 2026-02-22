package job

import (
	"context"
	"log/slog"
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
)

// AccessLogAnonymizeJob is a background job that anonymizes old share link access logs.
type AccessLogAnonymizeJob struct {
	shareLinkAccessRepo repository.ShareLinkAccessRepository
	interval            time.Duration
}

// NewAccessLogAnonymizeJob creates a new AccessLogAnonymizeJob.
func NewAccessLogAnonymizeJob(shareLinkAccessRepo repository.ShareLinkAccessRepository) *AccessLogAnonymizeJob {
	return &AccessLogAnonymizeJob{
		shareLinkAccessRepo: shareLinkAccessRepo,
		interval:            24 * time.Hour,
	}
}

// Start runs the anonymize job on a ticker loop until context is cancelled.
func (j *AccessLogAnonymizeJob) Start(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := j.shareLinkAccessRepo.AnonymizeOldAccesses(ctx)
			if err != nil {
				slog.Error("access log anonymize job failed", "error", err)
				continue
			}
			if count > 0 {
				slog.Info("anonymized old access logs", "count", count)
			}
		}
	}
}
