package logger

import (
	"log/slog"
	"os"

	"fiberest/internal/configs"
)

// NewLogger creates a structured slog logger based on Config.
// Handler is JSON when LOG_FORMAT=json, otherwise Text.
// Level is derived from LOG_LEVEL (debug|info|warn|error).
func NewLogger(cfg *configs.Config) *slog.Logger {
	var level slog.Level
	switch cfg.Log.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if cfg.Log.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}

// WithRequestID returns a logger enriched with request_id field.
func WithRequestID(l *slog.Logger, requestID string) *slog.Logger {
	return l.With(slog.String("request_id", requestID))
}
