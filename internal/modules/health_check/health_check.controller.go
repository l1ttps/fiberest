package healthcheck

import "github.com/gofiber/fiber/v3"

// Controller handles health check related requests
type Controller struct {
	app     *fiber.App
	service *Service
}

// NewController creates a new health check controller
func NewController(app *fiber.App, service *Service) *Controller {
	return &Controller{
		app:     app,
		service: service,
	}
}

// RegisterRoutes registers all health check routes
func (c *Controller) RegisterRoutes() {
	c.app.Get("/health-check", c.healthCheck)
}

// healthCheck handles GET /health-check request
func (c *Controller) healthCheck(ctx fiber.Ctx) error {
	result := c.service.CheckHealth()
	return ctx.SendString(result)
}
