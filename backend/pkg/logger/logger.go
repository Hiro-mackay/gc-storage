package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// Config はロガー設定を定義します
type Config struct {
	Level      string // ログレベル (debug, info, warn, error)
	Format     string // フォーマット (json, text)
	Output     string // 出力先 (stdout, stderr, file path)
	AddSource  bool   // ソースコード位置を含めるか
	TimeFormat string // 時刻フォーマット (RFC3339, RFC3339Nano)
}

// DefaultConfig はデフォルト設定を返します
func DefaultConfig() Config {
	return Config{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		AddSource:  false,
		TimeFormat: "RFC3339",
	}
}

// コンテキストキー
type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	UserIDKey    contextKey = "user_id"
	SessionIDKey contextKey = "session_id"
)

// Setup はグローバルロガーをセットアップします
func Setup(cfg Config) error {
	var output io.Writer
	switch cfg.Output {
	case "stdout", "":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		file, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		output = file
	}

	level := parseLevel(cfg.Level)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	switch cfg.Format {
	case "json", "":
		handler = slog.NewJSONHandler(output, opts)
	case "text":
		handler = slog.NewTextHandler(output, opts)
	default:
		handler = slog.NewJSONHandler(output, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return nil
}

// parseLevel はログレベル文字列をslog.Levelに変換します
func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithContext はコンテキストから情報を抽出してロガーを返します
func WithContext(ctx context.Context) *slog.Logger {
	logger := slog.Default()

	if requestID, ok := ctx.Value(RequestIDKey).(string); ok && requestID != "" {
		logger = logger.With("request_id", requestID)
	}

	if userID, ok := ctx.Value(UserIDKey).(string); ok && userID != "" {
		logger = logger.With("user_id", userID)
	}

	if sessionID, ok := ctx.Value(SessionIDKey).(string); ok && sessionID != "" {
		logger = logger.With("session_id", sessionID)
	}

	return logger
}

// ContextWithRequestID はリクエストIDをコンテキストに追加します
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// ContextWithUserID はユーザーIDをコンテキストに追加します
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// ContextWithSessionID はセッションIDをコンテキストに追加します
func ContextWithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, SessionIDKey, sessionID)
}

// Info はInfoレベルでログを出力します
func Info(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Info(msg, args...)
}

// Debug はDebugレベルでログを出力します
func Debug(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Debug(msg, args...)
}

// Warn はWarnレベルでログを出力します
func Warn(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Warn(msg, args...)
}

// Error はErrorレベルでログを出力します
func Error(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Error(msg, args...)
}

// WithError はエラーを含むロガーを返します
func WithError(ctx context.Context, err error) *slog.Logger {
	return WithContext(ctx).With("error", err.Error())
}

// WithField はフィールドを追加したロガーを返します
func WithField(ctx context.Context, key string, value any) *slog.Logger {
	return WithContext(ctx).With(key, value)
}

// WithFields は複数フィールドを追加したロガーを返します
func WithFields(ctx context.Context, fields map[string]any) *slog.Logger {
	logger := WithContext(ctx)
	for k, v := range fields {
		logger = logger.With(k, v)
	}
	return logger
}
