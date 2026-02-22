package handler

import (
	"context"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
)

// HealthChecker はヘルスチェックを実行するインターフェースです
type HealthChecker interface {
	Health(ctx context.Context) error
}

// HealthHandler はヘルスチェック関連のHTTPハンドラーです
type HealthHandler struct {
	checkers map[string]HealthChecker
}

// NewHealthHandler は新しいHealthHandlerを作成します
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		checkers: make(map[string]HealthChecker),
	}
}

// RegisterChecker はヘルスチェッカーを登録します
func (h *HealthHandler) RegisterChecker(name string, checker HealthChecker) {
	h.checkers[name] = checker
}

// HealthResponse はヘルスチェックレスポンスを定義します
type HealthResponse struct {
	Status string `json:"status"`
}

// ReadyResponse はレディネスチェックレスポンスを定義します
type ReadyResponse struct {
	Status   string                   `json:"status"`
	Services map[string]ServiceStatus `json:"services,omitempty"`
}

// ServiceStatus はサービスのステータスを定義します
type ServiceStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// Check はライブネスチェックを実行します
// GET /health
func (h *HealthHandler) Check(c echo.Context) error {
	return c.JSON(http.StatusOK, HealthResponse{
		Status: "ok",
	})
}

// Ready はレディネスチェックを実行します
// GET /ready
func (h *HealthHandler) Ready(c echo.Context) error {
	ctx := c.Request().Context()
	services := make(map[string]ServiceStatus)
	allHealthy := true

	var mu sync.Mutex
	var wg sync.WaitGroup

	for name, checker := range h.checkers {
		wg.Add(1)
		go func(name string, checker HealthChecker) {
			defer wg.Done()

			err := checker.Health(ctx)
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				services[name] = ServiceStatus{
					Status:  "unhealthy",
					Message: err.Error(),
				}
				allHealthy = false
			} else {
				services[name] = ServiceStatus{
					Status: "healthy",
				}
			}
		}(name, checker)
	}

	wg.Wait()

	status := "ready"
	statusCode := http.StatusOK
	if !allHealthy {
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	}

	return c.JSON(statusCode, ReadyResponse{
		Status:   status,
		Services: services,
	})
}
