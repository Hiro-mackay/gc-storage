package audit

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
)

// Service は監査ログの非同期書き込みサービスです
type Service struct {
	repo    repository.AuditLogRepository
	entries chan service.AuditEntry
	done    chan struct{}
}

// NewService は新しいAudit Serviceを作成します
func NewService(repo repository.AuditLogRepository, bufferSize int) *Service {
	if bufferSize <= 0 {
		bufferSize = 1000
	}
	s := &Service{
		repo:    repo,
		entries: make(chan service.AuditEntry, bufferSize),
		done:    make(chan struct{}),
	}
	go s.processLoop()
	return s
}

// Log は監査ログエントリをキューに追加します（非ブロッキング）
func (s *Service) Log(ctx context.Context, entry service.AuditEntry) {
	select {
	case s.entries <- entry:
	default:
		// バッファが満杯の場合はログ出力して破棄
		slog.Warn("audit log buffer full, dropping entry",
			"action", string(entry.Action),
			"resource_type", string(entry.ResourceType),
		)
	}
}

// processLoop はバッファからエントリを読み取り永続化します
func (s *Service) processLoop() {
	defer close(s.done)
	for entry := range s.entries {
		log := &entity.AuditLog{
			ID:           uuid.New(),
			UserID:       entry.UserID,
			Action:       entry.Action,
			ResourceType: entry.ResourceType,
			ResourceID:   entry.ResourceID,
			Details:      entry.Details,
			IPAddress:    entry.IPAddress,
			UserAgent:    entry.UserAgent,
			RequestID:    entry.RequestID,
			CreatedAt:    time.Now(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := s.repo.Create(ctx, log); err != nil {
			slog.Error("failed to write audit log",
				"error", err,
				"action", string(entry.Action),
				"resource_type", string(entry.ResourceType),
			)
		}
		cancel()
	}
}

// Shutdown はサービスを安全に停止します
func (s *Service) Shutdown() {
	close(s.entries)
	<-s.done
}

// インターフェースの実装を保証
var _ service.AuditService = (*Service)(nil)
