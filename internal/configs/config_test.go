package configs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"PORT", "APP_ENV", "APP_NAME",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"DB_MAX_IDLE_CONNS", "DB_MAX_OPEN_CONNS", "DB_MAX_LIFETIME", "DB_LOG_LEVEL", "DB_AUTO_MIGRATE",
		"CORS_ALLOW_ORIGINS", "CORS_ALLOW_CREDENTIALS",
		"LOG_LEVEL", "LOG_FORMAT",
		"SESSION_DURATION", "BCRYPT_COST", "SESSION_COOKIE_SECURE",
	} {
		_ = os.Unsetenv(k)
	}
}

func TestNewConfig_defaults(t *testing.T) {
	clearEnv(t)
	cfg, err := NewConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, 3278, cfg.App.Port)
	assert.Equal(t, "development", cfg.App.Env)
	assert.Equal(t, "Fiberest", cfg.App.Name)
	assert.Equal(t, "localhost", cfg.DB.Host)
	assert.Equal(t, 5432, cfg.DB.Port)
	assert.Equal(t, "disable", cfg.DB.SSLMode)
	assert.Equal(t, 10, cfg.DB.MaxIdleConns)
	assert.Equal(t, 100, cfg.DB.MaxOpenConns)
	assert.Equal(t, "silent", cfg.DB.LogLevel)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "text", cfg.Log.Format, "non-production defaults to text")
	assert.Equal(t, []string{"*"}, cfg.CORS.AllowOrigins, "dev defaults CORS to *")
	assert.Equal(t, 168, int(cfg.Auth.SessionDuration.Hours()), "SESSION_DURATION defaults to 168h")
	assert.Equal(t, 10, cfg.Auth.BcryptCost)

	// Conditional defaults: development may auto-migrate, cookies stay plain http.
	assert.True(t, cfg.DB.AutoMigrate)
	assert.False(t, cfg.Auth.CookieSecure)
}

func TestNewConfig_productionDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "production")

	cfg, err := NewConfig()
	require.NoError(t, err)
	assert.Equal(t, "json", cfg.Log.Format)
	assert.Empty(t, cfg.CORS.AllowOrigins, "production must not default to *")
	assert.False(t, cfg.DB.AutoMigrate, "auto-migrate is off in production by default")
	assert.True(t, cfg.Auth.CookieSecure)
}

func TestNewConfig_envOverridesDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("PORT", "8080")
	t.Setenv("SESSION_DURATION", "24h")
	t.Setenv("CORS_ALLOW_ORIGINS", "https://a.example, https://b.example")
	t.Setenv("BCRYPT_COST", "12")
	t.Setenv("DB_AUTO_MIGRATE", "false")
	t.Setenv("SESSION_COOKIE_SECURE", "true")

	cfg, err := NewConfig()
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.App.Port)
	assert.Equal(t, 24, int(cfg.Auth.SessionDuration.Hours()))
	assert.Equal(t, []string{"https://a.example", "https://b.example"}, cfg.CORS.AllowOrigins)
	assert.Equal(t, 12, cfg.Auth.BcryptCost)
	assert.False(t, cfg.DB.AutoMigrate)
	assert.True(t, cfg.Auth.CookieSecure)
}

func TestNewConfig_invalidValues_returnError(t *testing.T) {
	values := map[string]string{
		"PORT":             "99999",
		"LOG_LEVEL":        "verbose",
		"LOG_FORMAT":       "yaml",
		"DB_PORT":          "99999",
		"SESSION_DURATION": "one week",
		"BCRYPT_COST":      "99",
	}

	for key, value := range values {
		t.Run(key+"="+value, func(t *testing.T) {
			clearEnv(t)
			t.Setenv(key, value)
			_, err := NewConfig()
			require.Error(t, err)
		})
	}
}
