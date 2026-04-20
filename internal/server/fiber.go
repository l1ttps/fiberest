package server

import (
	"context"
	"fmt"
	"time"

	_ "fiberest/cmd/swag/docs" // Import swagger docs
	"fiberest/internal/configs"
	"fiberest/internal/middlewares"
	"fiberest/pkg/http_error"

	"github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"go.uber.org/fx"
)

// @title Fiberest API
// @version 1.0
// @description This is a sample server for a Fiber application.
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email fiber@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8080
// @BasePath /
// NewFiberApp creates a new Fiber app instance without starting it
func NewFiberApp(cfg *configs.Config) *fiber.App {
	app := fiber.New()

	// Configure CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: false,
	}))

	// Register Swagger route
	app.Get("/swagger/*", swaggo.New())

	// Rate limiting middleware
	app.Use(limiter.New(limiter.Config{
		Max:        20,
		Expiration: 1 * time.Second,
		LimitReached: func(c fiber.Ctx) error {
			return http_error.TooManyRequests(c, "Rate limit reached, please try again later")
		},
	}))

	// Global AuthGuard middleware
	app.Use(middlewares.AuthGuard(cfg))

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
