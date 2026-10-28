package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"fiberest/internal/configs"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestConfig(env string) *configs.Config {
	cfg := &configs.Config{}
	cfg.App.Env = env
	return cfg
}

func TestSecurityHeaders_SetsHeaders(t *testing.T) {
	app := fiber.New()
	app.Use(SecurityHeaders(newTestConfig("development")))
	app.Get("/test", func(c fiber.Ctx) error { return c.SendString("ok") })

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/test", nil))
	require.NoError(t, err)
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "no-referrer", resp.Header.Get("Referrer-Policy"))
	assert.NotEmpty(t, resp.Header.Get("Permissions-Policy"))
}

func TestSecurityHeaders_HSTSOnlyWithHTTPS(t *testing.T) {
	app := fiber.New()
	app.Use(SecurityHeaders(newTestConfig("production")))
	app.Get("/test", func(c fiber.Ctx) error { return c.SendString("ok") })

	// Fiber's test transport is plain http, so helmet must not emit HSTS even
	// though production sets HSTSMaxAge.
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/test", nil))
	require.NoError(t, err)
	assert.Empty(t, resp.Header.Get("Strict-Transport-Security"))
}
