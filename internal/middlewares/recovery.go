package middlewares

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"fiberest/pkg/http_error"

	"github.com/gofiber/fiber/v3"
)

// Recovery recovers from panics, logs them with the request_id and returns a
// generic 500. The panic value never reaches the client.
func Recovery(logger *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) (err error) {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			reqID := GetRequestID(c)
			logger.Error("panic recovered",
				slog.Any("panic", r),
				slog.String("stack", string(debug.Stack())),
				slog.String("method", c.Method()),
				slog.String("path", c.Path()),
				slog.String("request_id", reqID),
			)
			err = http_error.Internal(fmt.Errorf("panic: %v", r))
		}()
		return c.Next()
	}
}
