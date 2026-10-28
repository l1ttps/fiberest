package middlewares

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is both the inbound and outbound correlation header.
	// pkg/http_error carries a copy so it need not import internal.
	RequestIDHeader   = "X-Request-ID"
	RequestIDLocalKey = "request_id"
)

// RequestID propagates or generates X-Request-ID and stores it in locals.
func RequestID() fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Get(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(RequestIDHeader, id)
		c.Locals(RequestIDLocalKey, id)
		return c.Next()
	}
}

// GetRequestID retrieves the request id from locals (falls back to the header).
func GetRequestID(c fiber.Ctx) string {
	if v, ok := c.Locals(RequestIDLocalKey).(string); ok {
		return v
	}
	return c.Get(RequestIDHeader)
}
