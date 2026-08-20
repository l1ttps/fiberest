package middlewares

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecovery_CatchesPanic(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	app := fiber.New()
	app.Use(RequestID())
	app.Use(Recovery(logger))
	app.Get("/panic", func(c fiber.Ctx) error {
		panic("boom")
	})
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, float64(500), body["status"])
}

func TestRecovery_PassthroughWhenNoPanic(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	app := fiber.New()
	app.Use(Recovery(logger))
	app.Get("/ok", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRecovery_LogsRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	app := fiber.New()
	app.Use(RequestID())
	app.Use(Recovery(logger))
	app.Get("/panic2", func(c fiber.Ctx) error {
		panic("oops")
	})
	req := httptest.NewRequest(http.MethodGet, "/panic2", nil)
	req.Header.Set(RequestIDHeader, "test-req-id")
	_, err := app.Test(req)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "test-req-id")
}
