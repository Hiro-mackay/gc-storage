package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Job は定期実行ジョブを定義します
type Job struct {
	Name     string
	Interval time.Duration
	Fn       func(ctx context.Context) error
}

// Manager はバックグラウンドワーカーを管理します
type Manager struct {
	jobs   []Job
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewManager は新しいWorker Managerを作成します
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Register は定期実行ジョブを登録します
func (m *Manager) Register(job Job) {
	m.jobs = append(m.jobs, job)
}

// Start は全ジョブのワーカーを開始します
func (m *Manager) Start() {
	for _, job := range m.jobs {
		m.wg.Add(1)
		go m.runJob(job)
	}
	slog.Info("worker manager started", "jobs", len(m.jobs))
}

// runJob は単一ジョブのワーカーループを実行します
func (m *Manager) runJob(job Job) {
	defer m.wg.Done()

	slog.Info("worker started", "job", job.Name, "interval", job.Interval)

	// 最初の実行を即座に行う
	if err := job.Fn(m.ctx); err != nil {
		slog.Error("worker job failed", "job", job.Name, "error", err)
	}

	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			slog.Info("worker stopping", "job", job.Name)
			return
		case <-ticker.C:
			if err := job.Fn(m.ctx); err != nil {
				slog.Error("worker job failed", "job", job.Name, "error", err)
			}
		}
	}
}

// Shutdown はすべてのワーカーを安全に停止します
func (m *Manager) Shutdown(timeout time.Duration) {
	slog.Info("shutting down worker manager...")
	m.cancel()

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("worker manager stopped gracefully")
	case <-time.After(timeout):
		slog.Warn("worker manager shutdown timed out")
	}
}
