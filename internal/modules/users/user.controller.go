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

	// GET /users - Get paginated list of users
	users.Get("/", controller.getManyUsers)

	// GET /users/:id - Get user by ID
	users.Get("/:id", controller.getUserByID)
}

// initAdmin handles POST /users/init request
// @Summary Initialize admin account
// @Description Create the first admin account with the provided email and password
// @Tags users
// @Accept json
// @Produce json
// @Param request body dto.InitAdminRequest true "Admin initialization request"
// @Success 201 {object} dto.InitAdminResponse
// @Failure 400 {object} http_error.ErrorResponse
// @Router /users/init [post]
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

// getManyUsers handles GET /users request
// @Summary Get paginated list of users
// @Description Retrieve a paginated list of all users with pagination metadata
// @Tags users
// @Accept json
// @Produce json
// @Param limit query int false "Number of items per page (default: 10, max: 100)" minimum(1) maximum(100) default(10)
// @Param page query int false "Page number (default: 1)" minimum(1) default(1)
// @Param search query string false "Search keyword for filtering users"
// @Param role query string false "Filter by user role (ADMIN or USER)" Enums(ADMIN, USER)
// @Success 200 {object} dto.GetManyUsersExample
// @Failure 400 {object} http_error.ErrorResponse
// @Router /users [get]
func (c *Controller) getManyUsers(ctx fiber.Ctx) error {
	var req dto.GetManyUsersRequest

	// Parse query parameters
	if err := ctx.Bind().Query(&req); err != nil {
		return validators.ResponseError(ctx, err)
	}

	// Set default values if not provided
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Page <= 0 {
		req.Page = 1
	}

	// Validate parsed data
	if err := validators.ValidateStruct(&req); err != nil {
		return validators.ResponseError(ctx, err)
	}

	// Call service to get users
	response, err := c.service.GetManyUsers(req)
	if err != nil {
		return http_error.BadRequest(ctx, err.Error())
	}

	// Return success response
	return ctx.Status(fiber.StatusOK).JSON(response)
}

// getUserByID handles GET /users/:id request
// @Summary Get user by ID
// @Description Retrieve a single user by their unique ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} models.User
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 404 {object} http_error.ErrorResponse
// @Router /users/{id} [get]
func (c *Controller) getUserByID(ctx fiber.Ctx) error {
	// Get user ID from path parameter
	userID := ctx.Params("id")
	if userID == "" {
		return http_error.BadRequest(ctx, "user ID is required")
	}

	// Call service to find user
	user, err := c.service.FindByID(userID)
	if err != nil {
		if err.Error() == "user not found" {
			return http_error.NotFound(ctx, "user not found")
		}
		return http_error.BadRequest(ctx, err.Error())
	}

	// Return success response
	return ctx.Status(fiber.StatusOK).JSON(user)
}
