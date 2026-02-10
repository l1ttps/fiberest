package server

import (
	"context"
	"fmt"

	"fiberest/internal/configs"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
)

// NewFiberApp creates a new Fiber app instance without starting it
func NewFiberApp() *fiber.App {
	app := fiber.New()
	return app
}

// RegisterFiberLifecycle registers the Fiber app lifecycle hooks with fx
func RegisterFiberLifecycle(lc fx.Lifecycle, app *fiber.App, cfg *configs.Config) {
	port := cfg.GetString("PORT")
	address := fmt.Sprintf(":%s", port)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			fmt.Printf("Starting server on %s...\n", address)
			// Start server in a goroutine to not block
			go func() {
				if err := app.Listen(address); err != nil {
					fmt.Printf("Server error: %v\n", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			fmt.Println("Shutting down server...")
			return app.Shutdown()
		},
	})
}
