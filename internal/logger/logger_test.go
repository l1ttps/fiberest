package logger

import (
	"context"
	"log/slog"
	"testing"

	"fiberest/internal/configs"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestConfig(logLevel, logFormat, env string) *configs.Config {
	viper.Reset()
	viper.SetDefault("LOG_LEVEL", logLevel)
	viper.SetDefault("LOG_FORMAT", logFormat)
	viper.SetDefault("APP_ENV", env)
	cfg, err := configs.NewConfig()
	if err != nil {
		panic(err)
	}
	// Override directly for test isolation
	cfg.Log.Level = logLevel
	cfg.Log.Format = logFormat
	cfg.App.Env = env
	return cfg
}

func TestNewLogger_LevelAndFormat(t *testing.T) {
	cfg := newTestConfig("debug", "json", "development")
	l := NewLogger(cfg)
	require.NotNil(t, l)
	assert.True(t, l.Enabled(context.TODO(), slog.LevelDebug))
	assert.True(t, l.Enabled(context.TODO(), slog.LevelInfo))

	cfg2 := newTestConfig("error", "text", "development")
	l2 := NewLogger(cfg2)
	assert.False(t, l2.Enabled(context.TODO(), slog.LevelInfo))
	assert.True(t, l2.Enabled(context.TODO(), slog.LevelError))
}

func TestWithRequestID(t *testing.T) {
	cfg := newTestConfig("info", "text", "development")
	l := NewLogger(cfg)
	wrapped := WithRequestID(l, "abc-123")
	require.NotNil(t, wrapped)
	// Should not panic when logging
	wrapped.Info("test message")
}
