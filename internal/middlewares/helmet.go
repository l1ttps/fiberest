package middlewares

import (
	"fiberest/internal/configs"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
)

// SecurityHeaders returns helmet middleware with required headers.
// HSTS only when Env==production (via helmet's protocol check + MaxAge).
func SecurityHeaders(cfg *configs.Config) fiber.Handler {
	hstsAge := 0
	if cfg.App.Env == "production" {
		hstsAge = 31536000
	}
	return helmet.New(helmet.Config{
		XSSProtection:             "0",
		ContentTypeNosniff:        "nosniff",
		XFrameOptions:             "DENY",
		ReferrerPolicy:            "no-referrer",
		PermissionPolicy:          "geolocation=(), microphone=(), camera=()",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
		OriginAgentCluster:        "?1",
		XDNSPrefetchControl:       "off",
		XDownloadOptions:          "noopen",
		XPermittedCrossDomain:     "none",
		HSTSMaxAge:                hstsAge,
	})
}

// SecurityHeadersSimple is a stdlib-only fallback without fiber helmet.
// Kept for reference; not used when helmet dependency is available.
func SecurityHeadersSimple(cfg *configs.Config) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "0")
		c.Set("Referrer-Policy", "no-referrer")
		c.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		if cfg.App.Env == "production" && c.Protocol() == "https" {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		return c.Next()
	}
}

// Ensure SecurityHeadersSimple is referenced (avoid unused warning if not exported via other path).
var _ = SecurityHeadersSimple
