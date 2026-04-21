package auth

import (
	"errors"
	"fiberest/internal/common/constants"
	"fiberest/internal/common/validators"
	"fiberest/internal/middlewares"
	"fiberest/internal/modules/auth/dto"
	"fiberest/pkg/http_error"
	"time"

	"github.com/gofiber/fiber/v3"
)

// Controller handles authentication-related requests
type Controller struct {
	service AuthService
}

// NewController creates a new auth controller
func NewController(app *fiber.App, service AuthService) *Controller {
	return &Controller{
		service: service,
	}
}

// RegisterRoutes is invoked by fx to register auth routes
func RegisterRoutes(app *fiber.App, controller *Controller) {
	// Create a route group for /auth
	auth := app.Group("/auth")

	// POST /auth/init - Initialize admin account
	auth.Post("/init", controller.initAdmin)

	// POST /auth/login - User login
	auth.Post("/login", middlewares.Limiter(5, 60), controller.login)

	// POST /auth/refresh-token - Refresh JWT tokens
	auth.Post("/refresh-token", middlewares.Limiter(3, 60), controller.refreshToken)
}

// initAdmin handles POST /auth/init request
// @Summary Initialize first administrator account
// @Description Creates the very first administrator (admin) account for the system using the provided email and password. This endpoint can only be called successfully once, when no admin accounts exist in the system yet.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.InitAdminRequest true "Admin initialization request"
// @Success 201 {object} dto.InitAdminResponse
// @Failure 400 {object} http_error.ErrorResponse
// @Router /auth/init [post]
func (c *Controller) initAdmin(ctx fiber.Ctx) error {
	var req dto.InitAdminRequest

	// Parse and validate request in one step
	if err := validators.ParseAndValidate(ctx, &req); err != nil {
		return validators.ResponseError(ctx, err)
	}

	// Call service to create admin and get response
	response, err := c.service.CreateAdmin(ctx.Context(), req)

	if err != nil {
		if errors.Is(err, ErrAdminExists) {
			return http_error.BadRequest(ctx, err.Error())
		}
		return http_error.InternalServerError(ctx, err.Error())
	}

	// Return success response
	return ctx.Status(fiber.StatusCreated).JSON(response)
}

// login handles POST /auth/login request
// @Summary User authentication login
// @Description Authenticates a user using their email and password credentials. On successful authentication, returns both JWT access token and refresh token, and sets secure HTTP-only cookies for subsequent authenticated requests.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login request"
// @Success 200 {object} dto.TokenResponse
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 401 {object} http_error.ErrorResponse
// @Router /auth/login [post]
func (c *Controller) login(ctx fiber.Ctx) error {
	var req dto.LoginRequest

	if err := validators.ParseAndValidate(ctx, &req); err != nil {
		return validators.ResponseError(ctx, err)
	}

	response, err := c.service.Login(ctx.Context(), req)
	if err != nil {
		return http_error.BadRequest(ctx, err.Error())
	}

	ctx.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    response.AccessToken,
		Expires:  time.Now().Add(constants.AccessTokenDuration),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	})
	ctx.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    response.RefreshToken,
		Expires:  time.Now().Add(constants.RefreshTokenDuration),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	})

	return ctx.Status(fiber.StatusOK).JSON(response)
}

// refreshToken handles POST /auth/refresh-token request
// @Summary Refresh JWT authentication tokens
// @Description Issues a new pair of JWT tokens (access token and refresh token) using a valid existing refresh token. This endpoint allows users to maintain their authenticated session without re-entering login credentials when their access token expires.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "Refresh token request"
// @Success 200 {object} dto.TokenResponse
// @Failure 400 {object} http_error.ErrorResponse
// @Router /auth/refresh-token [post]
func (c *Controller) refreshToken(ctx fiber.Ctx) error {
	var req dto.RefreshTokenRequest

	if err := validators.ParseAndValidate(ctx, &req); err != nil {
		return validators.ResponseError(ctx, err)
	}

	response, err := c.service.RefreshToken(ctx.Context(), req)
	if err != nil {
		return http_error.BadRequest(ctx, err.Error())
	}

	ctx.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    response.AccessToken,
		Expires:  time.Now().Add(constants.AccessTokenDuration),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	})
	ctx.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    response.RefreshToken,
		Expires:  time.Now().Add(constants.RefreshTokenDuration),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	})

	return ctx.Status(fiber.StatusOK).JSON(response)
}
