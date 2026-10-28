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
