package middlewares

import (
	"time"

	"fiberest/pkg/http_error"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

// Limiter wraps Fiber's rate limiter.
//
// ponytail: default in-memory storage is per-process, so the limit multiplies
// with replicas. Swap in a shared storage (Redis) when running more than one.
func Limiter(limit int, duration int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        limit,
		Expiration: time.Duration(duration) * time.Second,
		LimitReached: func(c fiber.Ctx) error {
			return http_error.TooManyRequests("Rate limit reached, please try again later")
		},
	})
}

// LoginLimiter is a strict per-route limiter for /auth/login: 5 req/min.
func LoginLimiter() fiber.Handler { return Limiter(5, 60) }
