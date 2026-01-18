package server

import (
	"context"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
)

// Config はサーバー設定を定義します
type Config struct {
	Host            string        // ホスト (default: "")
	Port            int           // ポート (default: 8080)
	ReadTimeout     time.Duration // 読み取りタイムアウト (default: 30s)
	WriteTimeout    time.Duration // 書き込みタイムアウト (default: 30s)
	ShutdownTimeout time.Duration // シャットダウンタイムアウト (default: 10s)
	BodyLimit       string        // リクエストボディ制限 (default: "10MB")
	Debug           bool          // デバッグモード
}

// DefaultConfig はデフォルト設定を返します
func DefaultConfig() Config {
	return Config{
		Host:            "",
		Port:            8080,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		ShutdownTimeout: 10 * time.Second,
		BodyLimit:       "10MB",
		Debug:           false,
	}
}

// Server はHTTPサーバーを提供します
type Server struct {
	echo   *echo.Echo
	config Config
}

// NewServer は新しいServerを作成します
func NewServer(cfg Config) *Server {
	e := echo.New()

	// 基本設定
	e.Debug = cfg.Debug
	e.HideBanner = true
	e.HidePort = true

	// サーバーのタイムアウト設定
	e.Server.ReadTimeout = cfg.ReadTimeout
	e.Server.WriteTimeout = cfg.WriteTimeout

	return &Server{
		echo:   e,
		config: cfg,
	}
}

// Echo は内部のecho.Echoを返します
func (s *Server) Echo() *echo.Echo {
	return s.echo
}

// Config は設定を返します
func (s *Server) Config() Config {
	return s.config
}

// Start はサーバーを開始します
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	return s.echo.Start(addr)
}

// Shutdown はサーバーを停止します
func (s *Server) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
	defer cancel()
	return s.echo.Shutdown(shutdownCtx)
}

// Address はサーバーのアドレスを返します
func (s *Server) Address() string {
	return fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
}
