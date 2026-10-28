package logger

import (
	"context"
	"log/slog"
	"testing"

	"fiberest/internal/configs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestConfig(logLevel, logFormat string) *configs.Config {
	cfg := &configs.Config{}
	cfg.Log.Level = logLevel
	cfg.Log.Format = logFormat
	cfg.App.Env = "development"
	return cfg
}

func TestNewLogger_LevelAndFormat(t *testing.T) {
	l := NewLogger(newTestConfig("debug", "json"))
	require.NotNil(t, l)
	assert.True(t, l.Enabled(context.TODO(), slog.LevelDebug))
	assert.True(t, l.Enabled(context.TODO(), slog.LevelInfo))

	l2 := NewLogger(newTestConfig("error", "text"))
	assert.False(t, l2.Enabled(context.TODO(), slog.LevelInfo))
	assert.True(t, l2.Enabled(context.TODO(), slog.LevelError))
}

func TestWithRequestID(t *testing.T) {
	wrapped := WithRequestID(NewLogger(newTestConfig("info", "text")), "abc-123")
	require.NotNil(t, wrapped)
	require.NotPanics(t, func() { wrapped.Info("test message") })
}
