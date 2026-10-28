package middlewares

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fiberest/pkg/http_error"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecovery_CatchesPanic(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	app := fiber.New(fiber.Config{ErrorHandler: http_error.NewHandler(logger)})
	app.Use(RequestID())
	app.Use(Recovery(logger))
	app.Get("/panic", func(c fiber.Ctx) error {
		panic("boom")
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/panic", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(body, &parsed))
	assert.Equal(t, float64(500), parsed["status"])
	assert.NotContains(t, string(body), "boom", "panic value must not reach the client")
	assert.Contains(t, buf.String(), "panic recovered")
}

func TestRecovery_PassthroughWhenNoPanic(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	app := fiber.New()
	app.Use(Recovery(logger))
	app.Get("/ok", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/ok", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRecovery_LogsRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	app := fiber.New(fiber.Config{ErrorHandler: http_error.NewHandler(logger)})
	app.Use(RequestID())
	app.Use(Recovery(logger))
	app.Get("/panic2", func(c fiber.Ctx) error {
		panic("oops")
	})
	req := httptest.NewRequest(http.MethodGet, "/panic2", nil)
	req.Header.Set(RequestIDHeader, "test-req-id")
	_, err := app.Test(req)
	require.NoError(t, err)
	assert.True(t, strings.Contains(buf.String(), "test-req-id"))
}
