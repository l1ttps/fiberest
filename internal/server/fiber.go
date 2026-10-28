package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"slices"
	"time"

	"fiberest/docs" // Import swagger docs
	"fiberest/internal/configs"
	"fiberest/internal/database"
	"fiberest/internal/middlewares"
	"fiberest/pkg/http_error"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/yokeTH/gofiber-scalar/scalar/v3"
	"go.uber.org/fx"
)

const (
	bodyLimit    = 4 << 20 // 4 MiB
	readTimeout  = 10 * time.Second
	writeTimeout = 20 * time.Second
	idleTimeout  = 60 * time.Second
)

// RegisterHealthRoutes registers liveness and readiness probes.
// GET /health -> 200 {status:"ok", env}
// GET /health/ready -> 200 if db.Ping succeeds within 2s else 503
func RegisterHealthRoutes(app *fiber.App, db *database.DatabaseService, cfg *configs.Config) {
	app.Get("/health", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "ok",
			"env":    cfg.App.Env,
		})
	})

	app.Get("/health/ready", func(c fiber.Ctx) error {
		if db == nil {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
		}
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			// Generic body on purpose: probe errors may name internal hosts.
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "not ready"})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
	})
}

func NewFiberApp(cfg *configs.Config, db *database.DatabaseService, logger *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		BodyLimit:    bodyLimit,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
		ErrorHandler: http_error.NewHandler(logger),
	})

	// Core middlewares: RequestID -> Recovery -> SecurityHeaders -> CORS -> limiter.
	app.Use(middlewares.RequestID())
	app.Use(middlewares.Recovery(logger))
	app.Use(middlewares.SecurityHeaders(cfg))

	app.Get("/", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).SendString(cfg.App.Name)
	})

	app.Get("/health-check", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).SendString("OK")
	})

	origins := cfg.CORS.AllowOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}
	allowCredentials := cfg.CORS.AllowCredentials
	if allowCredentials && slices.Contains(origins, "*") {
		// Credentialed wildcard is invalid per the CORS spec; browsers reject it.
		allowCredentials = false
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		AllowCredentials: allowCredentials,
	}))

	// Health probes are registered before the rate limiter so they are never throttled.
	RegisterHealthRoutes(app, db, cfg)

	// Scalar API docs. FileContentString bypasses swag.ReadDoc() (swag v1/v2 conflict).
	app.Get("/docs/*", scalar.New(scalar.Config{
		FileContentString: docs.SwaggerInfo.ReadDoc(),
		Title:             fmt.Sprintf("%s API Documentation", cfg.App.Name),
		CacheAge:          3600,
		ForceOffline:      scalar.ForceOfflineTrue,
		FallbackCacheAge:  86400,
	}))

	app.Use(limiter.New(limiter.Config{
		Max:        20,
		Expiration: 1 * time.Second,
		LimitReached: func(c fiber.Ctx) error {
			return http_error.TooManyRequests("Rate limit reached, please try again later")
		},
	}))

	// Authentication is applied per route group (see the auth/users modules); there
	// is no global guard, so unknown paths fall through to the 404 handler below.
	return app
}

// Register404Handler registers the catch-all for unmatched routes.
// This must be called AFTER all other routes are registered.
func Register404Handler(app *fiber.App) {
	app.All("*", func(c fiber.Ctx) error {
		return http_error.NotFound(fmt.Sprintf("Cannot %s %s", c.Method(), c.Path()))
	})
}

// RegisterFiberLifecycle binds the listener synchronously (so a bad port fails
// startup loudly) and shuts down gracefully on stop.
func RegisterFiberLifecycle(lc fx.Lifecycle, app *fiber.App, cfg *configs.Config, logger *slog.Logger) {
	address := fmt.Sprintf(":%d", cfg.App.Port)

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			Register404Handler(app)

			ln, err := net.Listen("tcp", address)
			if err != nil {
				return fmt.Errorf("listen %s: %w", address, err)
			}

			logger.Info("starting server", slog.String("addr", address), slog.String("env", cfg.App.Env))

			go func() {
				err := app.Listener(ln, fiber.ListenConfig{
					DisableStartupMessage: cfg.App.Env != "development",
				})
				if err != nil && !errors.Is(err, net.ErrClosed) {
					logger.Error("server stopped", slog.String("error", err.Error()))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("shutting down server")
			return app.ShutdownWithContext(ctx)
		},
	})
}
