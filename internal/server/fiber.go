package server

import (
	"context"
	"fmt"
	"time"

	"fiberest/docs" // Import swagger docs
	"fiberest/internal/common/constants"
	"fiberest/internal/configs"
	"fiberest/internal/middlewares"
	"fiberest/internal/modules/auth"
	"fiberest/pkg/http_error"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/yokeTH/gofiber-scalar/scalar/v3"
	"go.uber.org/fx"
)

func NewFiberApp(cfg *configs.Config, authService auth.AuthService) *fiber.App {
	app := fiber.New()

	app.Get("/", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).SendString(constants.ApplicationName)
	})

	app.Get("/health-check", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).SendString("OK")
	})

	// Configure CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: false,
	}))

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
func RegisterFiberLifecycle(lc fx.Lifecycle, app *fiber.App, cfg *configs.Config) {
	port := cfg.GetString("PORT")
	address := fmt.Sprintf(":%s", port)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Register 404 handler last, after all routes are registered
			Register404Handler(app)

			fmt.Printf("Starting server on %s...\n", address)

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
			fmt.Println("Shutting down server...")
			return app.Shutdown()
		},
	})
}
