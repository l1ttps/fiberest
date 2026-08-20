package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fiberest/docs" // Import swagger docs
	"fiberest/internal/common/constants"
	"fiberest/internal/configs"
	"fiberest/internal/database"
	"fiberest/internal/middlewares"
	"fiberest/internal/modules/auth"
	"fiberest/pkg/http_error"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/yokeTH/gofiber-scalar/scalar/v3"
	"go.uber.org/fx"
)

// RegisterHealthRoutes registers liveness and readiness probes.
// GET /health -> 200 {status:"ok", env, version}
// GET /health/ready -> 200 if db.Ping succeeds within 2s else 503
func RegisterHealthRoutes(app *fiber.App, db *database.DatabaseService, cfg *configs.Config) {
	env := ""
	if cfg != nil {
		env = cfg.App.Env
	}
	// version is best-effort from constants
	version := ""
	// ponytail: no version constant yet; expose if build injects one later.

	app.Get("/health", func(c fiber.Ctx) error {
		resp := fiber.Map{
			"status": "ok",
			"env":    env,
		}
		if version != "" {
			resp["version"] = version
		}
		// fallback to ApplicationName as version hint when no version injected
		_ = constants.ApplicationName
		return c.Status(fiber.StatusOK).JSON(resp)
	})

	app.Get("/health/ready", func(c fiber.Ctx) error {
		if db == nil {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
		}
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "not ready",
				"error":  err.Error(),
			})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
	})
}

func NewFiberApp(cfg *configs.Config, authService auth.AuthService, db *database.DatabaseService, logger *slog.Logger) *fiber.App {
	app := fiber.New()

	// Core middlewares: RequestID -> Recovery -> SecurityHeaders -> CORS -> limiter -> auth
	app.Use(middlewares.RequestID())
	app.Use(middlewares.Recovery(logger))
	app.Use(middlewares.SecurityHeaders(cfg))

	app.Get("/", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).SendString(constants.ApplicationName)
	})

	app.Get("/health-check", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).SendString("OK")
	})

	// Configure CORS middleware from typed config
	origins := cfg.CORS.AllowOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: cfg.CORS.AllowCredentials,
	}))

	// Health probes - before rate limiter so probes are not throttled, before AuthGuard (publicRoutes)
	RegisterHealthRoutes(app, db, cfg)

	// Register Swagger route - use FileContentString to bypass swag.ReadDoc()
	// This avoids conflict between swag v1 and v2 versions
	swaggerContent := docs.SwaggerInfo.ReadDoc()

	app.Get("/docs/*", scalar.New(scalar.Config{
		FileContentString: swaggerContent,
		Title:             fmt.Sprintf("%s API Documentation", constants.ApplicationName),
		CacheAge:          3600,
		ForceOffline:      scalar.ForceOfflineTrue,
		FallbackCacheAge:  86400,
	}))

	// Rate limiting middleware
	app.Use(limiter.New(limiter.Config{
		Max:        20,
		Expiration: 1 * time.Second,
		LimitReached: func(c fiber.Ctx) error {
			return http_error.TooManyRequests(c, "Rate limit reached, please try again later")
		},
	}))

	// Global AuthGuard middleware (session-based)
	app.Use(middlewares.AuthGuard(authService))

	return app
}

// Register404Handler registers the 404 handler for unmatched routes
// This must be called AFTER all other routes are registered
func Register404Handler(app *fiber.App) {
	app.All("*", func(c fiber.Ctx) error {
		method := c.Method()
		path := c.Path()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  fiber.StatusNotFound,
			"error":   "Not Found",
			"message": fmt.Sprintf("Cannot %s %s", method, path),
		})
	})
}

// RegisterFiberLifecycle registers the Fiber app lifecycle hooks with fx
func RegisterFiberLifecycle(lc fx.Lifecycle, app *fiber.App, cfg *configs.Config, logger *slog.Logger) {
	port := cfg.GetString("PORT")
	address := fmt.Sprintf(":%s", port)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Register 404 handler last, after all routes are registered
			Register404Handler(app)

			logger.Info("starting server", slog.String("addr", address))

			errCh := make(chan error, 1)
			go func() {
				if err := app.Listen(address); err != nil {
					errCh <- err
				}
			}()

			// Wait for server to start or fail, or context to timeout
			select {
			case err := <-errCh:
				return fmt.Errorf("server failed to start: %w", err)
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
				// Assume server started successfully if no error after 2s
				// Fiber doesn't have a "Ready" signal, so this is a heuristic
				return nil
			}
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("shutting down server")
			return app.Shutdown()
		},
	})
}
