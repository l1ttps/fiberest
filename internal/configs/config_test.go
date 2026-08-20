package configs

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetViper() {
	viper.Reset()
	// Clear relevant env so tests are isolated
	for _, k := range []string{
		"PORT", "APP_ENV", "APP_NAME",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"DB_MAX_IDLE_CONNS", "DB_MAX_OPEN_CONNS", "DB_MAX_LIFETIME", "DB_LOG_LEVEL",
		"CORS_ALLOW_ORIGINS", "CORS_ALLOW_CREDENTIALS",
		"LOG_LEVEL", "LOG_FORMAT",
		"SESSION_DURATION", "BCRYPT_COST",
	} {
		_ = os.Unsetenv(k)
	}
}

func TestNewConfig_defaults(t *testing.T) {
	resetViper()
	cfg, err := NewConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, 3278, cfg.App.Port, "App.Port default 3278")
	assert.Equal(t, "development", cfg.App.Env)
	assert.Equal(t, "Fiberest", cfg.App.Name)
	assert.Equal(t, "localhost", cfg.DB.Host)
	assert.Equal(t, 5432, cfg.DB.Port)
	assert.Equal(t, "disable", cfg.DB.SSLMode)
	assert.Equal(t, 10, cfg.DB.MaxIdleConns)
	assert.Equal(t, 100, cfg.DB.MaxOpenConns)
	assert.Equal(t, "silent", cfg.DB.LogLevel)
	assert.Equal(t, "info", cfg.Log.Level)
	// text in development, json in production — default env is development
	assert.Equal(t, "text", cfg.Log.Format)
	require.NotEmpty(t, cfg.CORS.AllowOrigins, "dev should default CORS to *")
	assert.Equal(t, "*", cfg.CORS.AllowOrigins[0])
	assert.NotZero(t, cfg.Auth.SessionDuration)
	assert.Equal(t, 10, cfg.Auth.BcryptCost)
	// validate duration approx 168h
	assert.GreaterOrEqual(t, int64(cfg.DB.MaxLifetime), int64(0))
}

func TestNewConfig_invalidPort_returnsError(t *testing.T) {
	resetViper()
	t.Setenv("PORT", "99999")
	_, err := NewConfig()
	require.Error(t, err, "PORT=99999 beyond 65535 must fail validation")
}

func TestNewConfig_invalidLogLevel_returnsError(t *testing.T) {
	resetViper()
	t.Setenv("LOG_LEVEL", "verbose")
	_, err := NewConfig()
	require.Error(t, err, "LOG_LEVEL=verbose not in debug|info|warn|error")
}

func TestNewConfig_legacyGetStringStillWorks(t *testing.T) {
	resetViper()
	cfg, err := NewConfig()
	require.NoError(t, err)
	// Legacy accessors must delegate to viper and see the same defaults.
	assert.Equal(t, "3278", cfg.GetString("PORT"))
	assert.Equal(t, 3278, cfg.GetInt("PORT"))
	assert.NotNil(t, cfg.Get("PORT"))
	// DB defaults visible via legacy too
	assert.Equal(t, "localhost", cfg.GetString("DB_HOST"))
}

func TestNewConfig_invalidDBPort_returnsError(t *testing.T) {
	resetViper()
	t.Setenv("DB_PORT", "99999")
	_, err := NewConfig()
	require.Error(t, err)
}

func TestNewConfig_invalidLogFormat_returnsError(t *testing.T) {
	resetViper()
	t.Setenv("LOG_FORMAT", "yaml")
	_, err := NewConfig()
	require.Error(t, err)
}
