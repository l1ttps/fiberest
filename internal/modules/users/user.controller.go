package users

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"fiberest/internal/common/types"
	"fiberest/internal/common/validators"
	"fiberest/internal/middlewares"
	"fiberest/internal/models"
	"fiberest/internal/modules/users/dto"
	"fiberest/pkg/http_error"
)

// Controller handles user-related requests
type Controller struct {
	service UserService
}

// NewController creates a new user controller
func NewController(service UserService) *Controller {
	return &Controller{service: service}
}

// idParams is the shared path-parameter struct; IDs are UUIDs, so a malformed
// one is a 400 instead of a database error.
type idParams struct {
	ID string `param:"id" validate:"required,uuid"`
}

// UserRoutes is invoked by fx to register user routes.
// The guard lives on the groups, so unauthenticated calls to unknown paths still
// reach the 404 handler.
func UserRoutes(app *fiber.App, controller *Controller, authSvc middlewares.AuthService) {
	users := app.Group("/users")

	users.Patch("/me", middlewares.AuthGuard(authSvc), controller.patchMe)

	adminOnly := users.Group("/",
		middlewares.AuthGuard(authSvc),
		middlewares.RoleGuard(models.RoleAdmin),
	)
	adminOnly.Get("/", controller.getManyUsers)
	adminOnly.Get("/:id", controller.getUserByID)
	adminOnly.Put("/:id", controller.updateUserByID)
	adminOnly.Post("/:id/ban", controller.banUser)
	adminOnly.Post("/un-ban/:id", controller.unbanUser)
	adminOnly.Delete("/:id", controller.deleteUserByID)
	adminOnly.Post("/set-password/:id", controller.setPassword)
	adminOnly.Post("/", controller.createUser)
}

// mapError translates service sentinels into HTTP errors; unknown errors become
// a generic 500 whose cause is logged, never returned to the client.
func mapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrUserNotFound):
		return http_error.NotFound(err.Error())
	case errors.Is(err, ErrUserAlreadyExists):
		return http_error.Conflict(err.Error())
	default:
		return http_error.Internal(err)
	}
}

// getManyUsers handles GET /users request
// @Summary Get many users
// @Description Returns a paginated list of all users in the system with support for pagination, search by name/email, and filtering by user role (ADMIN or USER).
// @Tags Admin
// @Accept json
// @Produce json
// @Param limit query int false "Number of items per page (default: 10, max: 100)" minimum(1) maximum(100) default(10)
// @Param page query int false "Page number (default: 1)" minimum(1) default(1)
// @Param search query string false "Search keyword for filtering users"
// @Param role query string false "Filter by user role (ADMIN or USER)" Enums(ADMIN, USER)
// @Success 200 {object} dto.GetManyUsersExample
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 401 {object} http_error.ErrorResponse
// @Failure 403 {object} http_error.ErrorResponse
// @Router /users [get]
func (c *Controller) getManyUsers(ctx fiber.Ctx) error {
	var req dto.GetManyUsersRequest
	if err := validators.GetQuery(ctx, &req); err != nil {
		return validators.Invalid(err)
	}

	items, total, err := c.service.GetManyUsers(ctx.Context(), req)
	if err != nil {
		return http_error.Internal(err)
	}

	return ctx.Status(fiber.StatusOK).JSON(dto.GetManyUsersExample{
		Data:     items,
		PageMeta: types.NewPageMeta(req.Limit, req.Page, total),
	})
}

// getUserByID handles GET /users/:id request
// @Summary Get user details by ID
// @Description Retrieves complete detailed information about a specific user using their unique identifier. Returns a 404 Not Found error if no user exists with the provided ID.
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "User ID (UUID)"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 404 {object} http_error.ErrorResponse
// @Router /users/{id} [get]
func (c *Controller) getUserByID(ctx fiber.Ctx) error {
	var params idParams
	if err := validators.GetParam(ctx, &params); err != nil {
		return validators.Invalid(err)
	}

	user, err := c.service.FindByID(ctx.Context(), params.ID)
	if err != nil {
		return mapError(err)
	}

	return ctx.Status(fiber.StatusOK).JSON(dto.NewUserResponse(user))
}

// updateUserByID handles PUT /users/:id request
// @Summary Update user by ID
// @Description Updates user information (email, name, role) by user ID. Only provided fields will be updated. Returns updated user data.
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "User ID (UUID)"
// @Param request body dto.UpdateUserRequest true "User update data"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 404 {object} http_error.ErrorResponse
// @Failure 409 {object} http_error.ErrorResponse
// @Router /users/{id} [put]
func (c *Controller) updateUserByID(ctx fiber.Ctx) error {
	var params idParams
	if err := validators.GetParam(ctx, &params); err != nil {
		return validators.Invalid(err)
	}

	var req dto.UpdateUserRequest
	if err := validators.ParseAndValidate(ctx, &req); err != nil {
		return validators.Invalid(err)
	}

	user, err := c.service.UpdateUser(ctx.Context(), params.ID, req)
	if err != nil {
		return mapError(err)
	}

	return ctx.Status(fiber.StatusOK).JSON(dto.NewUserResponse(user))
}

// deleteUserByID handles DELETE /users/:id request
// @Summary Delete user by ID
// @Description Permanently removes a user from the system by their unique identifier. Returns 204 No Content on success.
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "User ID (UUID)"
// @Success 204 "User deleted successfully"
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 404 {object} http_error.ErrorResponse
// @Router /users/{id} [delete]
func (c *Controller) deleteUserByID(ctx fiber.Ctx) error {
	var params idParams
	if err := validators.GetParam(ctx, &params); err != nil {
		return validators.Invalid(err)
	}

	if err := c.service.DeleteUser(ctx.Context(), params.ID); err != nil {
		return mapError(err)
	}

	return ctx.Status(fiber.StatusNoContent).SendString("")
}

// createUser handles POST /users request
// @Summary Create a new user
// @Description Creates a new user with email, password, name and role. Only admins can create users. Default role is USER.
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body dto.CreateUserRequest true "User creation data"
// @Success 201 {object} map[string]string "User created successfully"
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 409 {object} http_error.ErrorResponse
// @Router /users [post]
func (c *Controller) createUser(ctx fiber.Ctx) error {
	var req dto.CreateUserRequest
	if err := validators.ParseAndValidate(ctx, &req); err != nil {
		return validators.Invalid(err)
	}

	role := models.RoleUser
	if req.Role != "" {
		role = models.UserRole(req.Role)
	}

	if _, err := c.service.CreateUserWithPassword(ctx.Context(), req.Email, req.Password, req.Name, role); err != nil {
		return mapError(err)
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "User created successfully"})
}

// banUser handles POST /users/:id/ban request
// @Summary Ban a user
// @Description Bans a user with a reason and an optional expiration date. Only admins can perform this action.
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "User ID (UUID)"
// @Param request body dto.BanUserRequest true "Ban details"
// @Success 200 {object} map[string]string "User banned successfully"
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 404 {object} http_error.ErrorResponse
// @Router /users/{id}/ban [post]
func (c *Controller) banUser(ctx fiber.Ctx) error {
	var params idParams
	if err := validators.GetParam(ctx, &params); err != nil {
		return validators.Invalid(err)
	}

	var req dto.BanUserRequest
	if err := validators.ParseAndValidate(ctx, &req); err != nil {
		return validators.Invalid(err)
	}

	var until *time.Time
	if req.Until != "" {
		t, err := time.Parse(time.RFC3339, req.Until)
		if err != nil {
			return http_error.BadRequest("Invalid date format for 'until'. Use RFC3339 (e.g., 2026-12-31T23:59:59Z)")
		}
		until = &t
	}

	if err := c.service.BanUser(ctx.Context(), params.ID, req.Reason, until); err != nil {
		return mapError(err)
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "User banned successfully"})
}

// unbanUser handles POST /users/un-ban/:id request
// @Summary Unban a user
// @Description Lifts the ban from a user. Only admins can perform this action.
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "User ID (UUID)"
// @Success 200 {object} map[string]string "User unbanned successfully"
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 404 {object} http_error.ErrorResponse
// @Router /users/un-ban/{id} [post]
func (c *Controller) unbanUser(ctx fiber.Ctx) error {
	var params idParams
	if err := validators.GetParam(ctx, &params); err != nil {
		return validators.Invalid(err)
	}

	if err := c.service.UnbanUser(ctx.Context(), params.ID); err != nil {
		return mapError(err)
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "User unbanned successfully"})
}

// setPassword handles POST /users/set-password/:id request
// @Summary Set password for a user
// @Description Sets or updates the password for a user's EMAIL authentication account. Only admins can set passwords for any user.
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "User ID (UUID)"
// @Param request body dto.SetPasswordRequest true "New password (min 8 characters)"
// @Success 200 {object} map[string]string "Password set successfully"
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 404 {object} http_error.ErrorResponse
// @Router /users/set-password/{id} [post]
func (c *Controller) setPassword(ctx fiber.Ctx) error {
	var params idParams
	if err := validators.GetParam(ctx, &params); err != nil {
		return validators.Invalid(err)
	}

	var req dto.SetPasswordRequest
	if err := validators.ParseAndValidate(ctx, &req); err != nil {
		return validators.Invalid(err)
	}

	if err := c.service.SetPassword(ctx.Context(), params.ID, req.Password); err != nil {
		return mapError(err)
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Password set successfully"})
}

// patchMe handles PATCH /users/me request
// @Summary Update my profile
// @Description Allows an authenticated user to update their profile (name). User ID comes from the session.
// @Tags Users
// @Accept json
// @Produce json
// @Param request body dto.UpdateMyProfileRequest true "Profile data"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 401 {object} http_error.ErrorResponse
// @Router /users/me [patch]
func (c *Controller) patchMe(ctx fiber.Ctx) error {
	userID, ok := middlewares.UserID(ctx)
	if !ok {
		return http_error.Unauthorized("User not authenticated")
	}

	var req dto.UpdateMyProfileRequest
	if err := validators.ParseAndValidate(ctx, &req); err != nil {
		return validators.Invalid(err)
	}

	user, err := c.service.UpdateMyProfile(ctx.Context(), userID, req)
	if err != nil {
		return mapError(err)
	}

	return ctx.Status(fiber.StatusOK).JSON(dto.NewUserResponse(user))
}
