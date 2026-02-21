package worker

import (
	"context"
	"log/slog"
	"time"
)

// TrashCleanupJobConfig はゴミ箱クリーンアップジョブの設定です
type TrashCleanupJobConfig struct {
	// RetentionDays はゴミ箱アイテムの保持日数です
	RetentionDays int
}

// NewTrashCleanupJob はゴミ箱クリーンアップジョブを作成します
// cleanupFn は実際のクリーンアップロジックを実行する関数です
func NewTrashCleanupJob(cleanupFn func(ctx context.Context, olderThan time.Time) (int, error), cfg TrashCleanupJobConfig) Job {
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = 30
	}

	return Job{
		Name:     "trash_cleanup",
		Interval: 1 * time.Hour,
		Fn: func(ctx context.Context) error {
			olderThan := time.Now().AddDate(0, 0, -cfg.RetentionDays)
			count, err := cleanupFn(ctx, olderThan)
			if err != nil {
				return err
			}
			if count > 0 {
				slog.Info("trash cleanup completed", "deleted", count)
			}
			return nil
		},
	}
}

// NewSessionCleanupJob はセッションクリーンアップジョブを作成します
func NewSessionCleanupJob(cleanupFn func(ctx context.Context) error) Job {
	return Job{
		Name:     "session_cleanup",
		Interval: 6 * time.Hour,
		Fn: func(ctx context.Context) error {
			if err := cleanupFn(ctx); err != nil {
				return err
			}
			slog.Debug("session cleanup completed")
			return nil
		},
	}
}

// NewHealthCheckJob はヘルスチェックジョブを作成します（データベース接続確認など）
func NewHealthCheckJob(checkFn func(ctx context.Context) error) Job {
	return Job{
		Name:     "health_check",
		Interval: 5 * time.Minute,
		Fn: func(ctx context.Context) error {
			if err := checkFn(ctx); err != nil {
				slog.Warn("health check failed", "error", err)
				return err
			}
			return nil
		},
	}
}
