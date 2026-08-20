package middlewares

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"fiberest/pkg/http_error"

	"github.com/gofiber/fiber/v3"
)

// Recovery recovers from panics, logs with request_id, returns 500 JSON.
func Recovery(logger *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				reqID := GetRequestID(c)
				l := logger
				if reqID != "" {
					l = logger.With(slog.String("request_id", reqID))
				}
				l.Error("panic recovered",
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())),
					slog.String("method", c.Method()),
					slog.String("path", c.Path()),
				)
				_ = http_error.InternalServerError(c, fmt.Sprintf("%v", r))
			}
		}()
		return c.Next()
	}
}
