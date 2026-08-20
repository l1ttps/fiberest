package middlewares

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	app := fiber.New()
	app.Use(RequestID())
	app.Get("/test", func(c fiber.Ctx) error {
		id := GetRequestID(c)
		require.NotEmpty(t, id)
		// UUID v4 format check (36 chars, hyphens)
		assert.Len(t, id, 36)
		return c.SendString(id)
	})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Len(t, string(body), 36)
	assert.NotEmpty(t, resp.Header.Get(RequestIDHeader))
	assert.Equal(t, string(body), resp.Header.Get(RequestIDHeader))
	// Locals also set: verify via handler returning it matches header
}

func TestRequestID_PropagatesExisting(t *testing.T) {
	app := fiber.New()
	app.Use(RequestID())
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString(GetRequestID(c))
	})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(RequestIDHeader, "my-custom-id-123")
	resp, err := app.Test(req)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "my-custom-id-123", string(body))
	assert.Equal(t, "my-custom-id-123", resp.Header.Get(RequestIDHeader))
}

func TestGetRequestID_EmptyWhenNoMiddleware(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		id := GetRequestID(c)
		assert.Empty(t, id)
		return c.SendStatus(200)
	})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
