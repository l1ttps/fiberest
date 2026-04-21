package users

import (
	"errors"
	"fiberest/internal/common/validators"
	"fiberest/internal/middlewares"
	"fiberest/internal/models"
	"fiberest/internal/modules/users/dto"
	"fiberest/pkg/http_error"

	"github.com/gofiber/fiber/v3"
)

// Controller handles user-related requests
type Controller struct {
	service UserService
}

// NewController creates a new user controller
func NewController(app *fiber.App, service UserService) *Controller {
	return &Controller{
		service: service,
	}
}

// UserRoutes is invoked by fx to register user routes
func UserRoutes(app *fiber.App, controller *Controller) {
	// Create a route group for /users
	users := app.Group("/users", middlewares.RoleGuard(models.RoleAdmin))

	// GET /users - Get paginated list of users
	users.Get("/", controller.getManyUsers)

	// GET /users/:id - Get user by ID
	users.Get("/:id", controller.getUserByID)
}

// getManyUsers handles GET /users request
// @Summary Get paginated users list with filters
// @Description Returns a paginated list of all users in the system with support for pagination, search by name/email, and filtering by user role (ADMIN or USER). Response includes comprehensive pagination metadata: total users count, current page number, total pages, and items per page.
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
	response, err := c.service.GetManyUsers(ctx.Context(), req)
	if err != nil {
		return http_error.InternalServerError(ctx, err.Error())
	}

	// Return success response
	return ctx.Status(fiber.StatusOK).JSON(response)
}

// getUserByID handles GET /users/:id request
// @Summary Get user details by ID
// @Description Retrieves complete detailed information about a specific user using their unique identifier. Returns a 404 Not Found error if no user exists with the provided ID.
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
	user, err := c.service.FindByID(ctx.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return http_error.NotFound(ctx, err.Error())
		}
		return http_error.InternalServerError(ctx, err.Error())
	}

	// Return success response
	return ctx.Status(fiber.StatusOK).JSON(user)
}
