package http_error

import (
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// The handler must never leak an internal cause to the client.
func TestHandlerHidesInternalCause(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: NewHandler(nil)})
	app.Get("/boom", func(fiber.Ctx) error {
		return Internal(errors.New("pq: password authentication failed for user \"postgres\""))
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/boom", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Fatalf("want 500 got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), "postgres") {
		t.Fatalf("cause leaked to client: %s", body)
	}
	if !strings.Contains(string(body), `"code":"INTERNAL_ERROR"`) {
		t.Fatalf("missing code in body: %s", body)
	}
}

func TestHandlerRenders4xx(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: NewHandler(nil)})
	app.Get("/nope", func(fiber.Ctx) error {
		return NotFound("user not found")
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/nope", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Fatalf("want 404 got %d", resp.StatusCode)
	}
}
