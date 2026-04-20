package middlewares

import (
	"fiberest/pkg/http_error"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

// Middleware wraps Fiber's built-in limiter middleware.
// limit: Maximum number of requests allowed.
// duration: Time window in seconds.
func Limiter(limit int, duration int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        limit,
		Expiration: time.Duration(duration) * time.Second,
		LimitReached: func(c fiber.Ctx) error {
			return http_error.TooManyRequests(c, "Rate limit reached, please try again later")
		},
	})
}
