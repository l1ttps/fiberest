package users

import (
	"fiberest/internal/common/validators"
	"fiberest/internal/modules/users/dto"
	"fiberest/pkg/http_error"

	"github.com/gofiber/fiber/v3"
)

// Controller handles user-related requests
type Controller struct {
	service *Service
}

// UserController creates a new user controller
func UserController(app *fiber.App, service *Service) *Controller {
	return &Controller{
		service: service,
	}
}

// UserRoutes is invoked by fx to register user routes
// This follows the same pattern as RegisterFiberLifecycle in server module
func UserRoutes(app *fiber.App, controller *Controller) {
	// Create a route group for /users
	users := app.Group("/users")

	// POST /users/init - Initialize admin account
	users.Post("/init", controller.initAdmin)
}

// initAdmin handles POST /users/init request
// It creates a new admin user with the provided email and password
func (c *Controller) initAdmin(ctx fiber.Ctx) error {
	var req dto.InitAdminRequest

	// Parse and validate request in one step
	if err := validators.ParseAndValidate(ctx, &req); err != nil {
		return validators.ResponseError(ctx, err)
	}

	// Call service to create admin and get response
	response, error := c.service.CreateAdmin(req)

	if error != nil {
		return http_error.BadRequest(ctx, error.Error())
	}

	// Return success response
	return ctx.Status(fiber.StatusCreated).JSON(response)
}
