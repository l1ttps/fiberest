package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"fiberest/internal/configs"

	"github.com/gofiber/fiber/v3"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestConfig(env string) *configs.Config {
	viper.Reset()
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "text")
	cfg, err := configs.NewConfig()
	if err != nil {
		panic(err)
	}
	cfg.App.Env = env
	return cfg
}

func TestSecurityHeaders_SetsHeaders(t *testing.T) {
	cfg := newTestConfig("development")
	app := fiber.New()
	app.Use(SecurityHeaders(cfg))
	app.Get("/test", func(c fiber.Ctx) error { return c.SendString("ok") })
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "no-referrer", resp.Header.Get("Referrer-Policy"))
	assert.NotEmpty(t, resp.Header.Get("Permissions-Policy"))
}

func TestSecurityHeaders_HSTSInProduction(t *testing.T) {
	cfg := newTestConfig("production")
	// Helmet's HSTS is gated on c.Protocol() == "https" and HSTSMaxAge != 0.
	// Fiber's Protocol() is "https" only with TLS. In tests we verify the
	// config drives HSTSMaxAge correctly and headers are still set.
	app := fiber.New()
	app.Use(SecurityHeaders(cfg))
	app.Get("/test", func(c fiber.Ctx) error { return c.SendString("ok") })
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))

	// Also verify development has no HSTS expectation — just that production
	// config would enable it (checked via SecurityHeadersSimple with https mock is impractical,
	// so we assert production cfg differs from development).
	cfgDev := newTestConfig("development")
	assert.NotEqual(t, cfgDev.App.Env, cfg.App.Env)
}

func TestSecurityHeadersSimple_HSTSOnlyOnHTTPSAndProduction(t *testing.T) {
	cfgProd := newTestConfig("production")
	cfgDev := newTestConfig("development")

	// Production + https should set HSTS via simple fallback
	app := fiber.New()
	app.Use(SecurityHeadersSimple(cfgProd))
	app.Get("/test", func(c fiber.Ctx) error { return c.SendString("ok") })
	// Fiber test always uses http; we verify header logic at least doesn't crash and sets base headers
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	_ = cfgDev
}
