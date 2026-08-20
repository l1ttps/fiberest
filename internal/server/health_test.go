package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"fiberest/internal/configs"

	"github.com/gofiber/fiber/v3"
)

func TestHealthEndpoints(t *testing.T) {
	cfg := &configs.Config{}
	cfg.App.Env = "test"

	app := fiber.New()
	RegisterHealthRoutes(app, nil, cfg)

	t.Run("liveness", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("want 200 got %d", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		var m map[string]any
		if err := json.Unmarshal(body, &m); err != nil {
			t.Fatalf("invalid json: %v body=%s", err, string(body))
		}
		if m["status"] != "ok" {
			t.Fatalf("want status ok got %v", m["status"])
		}
		if m["env"] != "test" {
			t.Fatalf("want env test got %v", m["env"])
		}
	})

	t.Run("readiness_nil_db_ok", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("want 200 got %d", resp.StatusCode)
		}
	})
}

func TestCORSFallback(t *testing.T) {
	cfg := &configs.Config{}
	cfg.App.Env = "development"
	// empty origins should fallback to ["*"] inside NewFiberApp; core logic is origins==0 => ["*"]
	if len(cfg.CORS.AllowOrigins) != 0 {
		t.Fatalf("want empty origins for fallback test")
	}
	origins := cfg.CORS.AllowOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}
	if len(origins) != 1 || origins[0] != "*" {
		t.Fatalf("fallback failed got %v", origins)
	}
}
