package healthcheck

import "github.com/gofiber/fiber/v3"

// Controller handles health check related requests
type Controller struct {
	service Service
}

// NewController creates a new health check controller
func NewController(service Service) *Controller {
	return &Controller{
		service: service,
	}
}

// RegisterRoutes registers all health check routes
func RegisterRoutes(app *fiber.App, controller *Controller) {
	app.Get("/health-check", controller.healthCheck)
}

// healthCheck handles GET /health-check request
// @Summary Check health
// @Description Check if the server is running and return "OK"
// @Tags Health Check
// @Accept plain
// @Produce plain
// @Success 200 {string} string "OK"
// @Router /health-check [get]
func (c *Controller) healthCheck(ctx fiber.Ctx) error {
	result := c.service.CheckHealth()
	return ctx.SendString(result)
}
