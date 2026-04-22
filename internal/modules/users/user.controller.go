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

	// Create a route group for /users (admin only)
	users := app.Group("/users")

	// PATCH /users/me - Update own information (authenticated users only)
	users.Patch("/me", controller.patchMe)

	adminOnly := users.Group("/", middlewares.RoleGuard(models.RoleAdmin))
	// GET /users - Get paginated list of users
	adminOnly.Get("/", controller.getManyUsers)

	// GET /users/:id - Get user by ID
	adminOnly.Get("/:id", controller.getUserByID)

	// PUT /users/:id - Update user by ID
	adminOnly.Put("/:id", controller.updateUserByID)

	// DELETE /users/:id - Delete user by ID
	adminOnly.Delete("/:id", controller.deleteUserByID)

	// POST /users/set-password/:id - Set password for user
	adminOnly.Post("/set-password/:id", controller.setPassword)
}

// getManyUsers handles GET /users request
// @Summary Get paginated users list with filters
// @Description Returns a paginated list of all users in the system with support for pagination, search by name/email, and filtering by user role (ADMIN or USER). Response includes comprehensive pagination metadata: total users count, current page number, total pages, and items per page.
// @Tags Users
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
// @Tags Users
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

// updateUserByID handles PUT /users/:id request
// @Summary Update user by ID
// @Description Updates user information (email, name, role) by user ID. Only provided fields will be updated. Returns updated user data.
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body dto.UpdateUserRequest true "User update data"
// @Success 200 {object} models.User
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 404 {object} http_error.ErrorResponse
// @Failure 409 {object} http_error.ErrorResponse
// @Router /users/{id} [put]
func (c *Controller) updateUserByID(ctx fiber.Ctx) error {
	// Get user ID from path parameter
	userID := ctx.Params("id")
	if userID == "" {
		return http_error.BadRequest(ctx, "user ID is required")
	}

	// Parse request body
	var req dto.UpdateUserRequest
	if err := validators.ParseAndValidate(ctx, &req); err != nil {
		return validators.ResponseError(ctx, err)
	}

	// Call service to update user
	user, err := c.service.UpdateUser(ctx.Context(), userID, req)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return http_error.NotFound(ctx, err.Error())
		}
		if errors.Is(err, ErrUserAlreadyExists) {
			return http_error.Conflict(ctx, err.Error())
		}
		return http_error.InternalServerError(ctx, err.Error())
	}

	// Return success response
	return ctx.Status(fiber.StatusOK).JSON(user)
}

// deleteUserByID handles DELETE /users/:id request
// @Summary Delete user by ID
// @Description Permanently removes a user from the system by their unique identifier. Returns 204 No Content on success.
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 204 "User deleted successfully"
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 404 {object} http_error.ErrorResponse
// @Router /users/{id} [delete]
func (c *Controller) deleteUserByID(ctx fiber.Ctx) error {
	// Get user ID from path parameter
	userID := ctx.Params("id")
	if userID == "" {
		return http_error.BadRequest(ctx, "user ID is required")
	}

	// Call service to delete user
	err := c.service.DeleteUser(ctx.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return http_error.NotFound(ctx, err.Error())
		}
		return http_error.InternalServerError(ctx, err.Error())
	}

	// Return success response (204 No Content)
	return ctx.Status(fiber.StatusNoContent).SendString("")
}

// setPassword handles POST /users/set-password/:id request
// @Summary Set password for a user
// @Description Sets or updates the password for a user's EMAIL authentication account. Only admins can set passwords for any user.
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body dto.SetPasswordRequest true "New password (min 8 characters)"
// @Success 200 {object} map[string]string "Password set successfully"
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 404 {object} http_error.ErrorResponse
// @Router /users/set-password/{id} [post]
func (c *Controller) setPassword(ctx fiber.Ctx) error {
	// Get user ID from path parameter
	userID := ctx.Params("id")
	if userID == "" {
		return http_error.BadRequest(ctx, "user ID is required")
	}

	// Parse request body
	var req dto.SetPasswordRequest
	if err := validators.ParseAndValidate(ctx, &req); err != nil {
		return validators.ResponseError(ctx, err)
	}

	// Call service to set password
	err := c.service.SetPassword(ctx.Context(), userID, req.Password)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return http_error.NotFound(ctx, err.Error())
		}
		return http_error.InternalServerError(ctx, err.Error())
	}

	// Return success response
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Password set successfully",
	})
}

// patchMe handles PATCH /users/me request
// @Summary Update my profile
// @Description Allows an authenticated user to update their profile (name and email). User ID is extracted from the session token.
// @Tags Users
// @Accept json
// @Produce json
// @Param request body dto.UpdateMyProfileRequest true "Profile data (name required, email optional)"
// @Success 200 {object} models.User
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 401 {object} http_error.ErrorResponse
// @Failure 409 {object} http_error.ErrorResponse
// @Router /users/me [patch]
func (c *Controller) patchMe(ctx fiber.Ctx) error {
	// Get user ID from context (set by AuthGuard middleware)
	userID, _ := ctx.Locals("user_id").(string)

	// Parse request body
	var req dto.UpdateMyProfileRequest
	if err := validators.ParseAndValidate(ctx, &req); err != nil {
		return validators.ResponseError(ctx, err)
	}

	// Call service to update profile
	user, err := c.service.UpdateMyProfile(ctx.Context(), userID, req)
	if err != nil {
		return http_error.InternalServerError(ctx, err.Error())
	}

	// Return success response
	return ctx.Status(fiber.StatusOK).JSON(user)
}
