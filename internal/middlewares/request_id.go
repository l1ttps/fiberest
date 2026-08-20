package middlewares

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"
const RequestIDLocalKey = "request_id"

// RequestID generates or propagates X-Request-ID, stores in header and locals.
func RequestID() fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Get(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(RequestIDHeader, id)
		c.Locals(RequestIDLocalKey, id)
		// Also set response header explicitly for clients
		c.Set(RequestIDHeader, id)
		return c.Next()
	}
}

// GetRequestID retrieves request id from context locals.
func GetRequestID(c fiber.Ctx) string {
	if v, ok := c.Locals(RequestIDLocalKey).(string); ok {
		return v
	}
	return c.Get(RequestIDHeader)
}
