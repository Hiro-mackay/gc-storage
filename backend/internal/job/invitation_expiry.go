package job

import (
	"context"
	"log/slog"
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
)

// InvitationExpiryJob is a background job that expires old invitations.
type InvitationExpiryJob struct {
	invitationRepo repository.InvitationRepository
	interval       time.Duration
}

// NewInvitationExpiryJob creates a new InvitationExpiryJob.
func NewInvitationExpiryJob(invitationRepo repository.InvitationRepository) *InvitationExpiryJob {
	return &InvitationExpiryJob{
		invitationRepo: invitationRepo,
		interval:       time.Hour,
	}
}

// Start runs the expiry job on a ticker loop until context is cancelled.
func (j *InvitationExpiryJob) Start(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := j.invitationRepo.ExpireOld(ctx)
			if err != nil {
				slog.Error("invitation expiry job failed", "error", err)
				continue
			}
			if count > 0 {
				slog.Info("expired invitations", "count", count)
			}
		}
	}
}
