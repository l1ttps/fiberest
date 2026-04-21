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

	// POST /auth/logout - User logout
	auth.Post("/logout", middlewares.Limiter(5, 60), controller.logout)

	// GET /auth/session - Current session
	auth.Get("/session", controller.session)
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
// @Description Authenticates a user using their email and password credentials. On successful authentication, creates a session and sets an HTTP-only cookie for subsequent authenticated requests.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login request"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 401 {object} http_error.ErrorResponse
// @Router /auth/login [post]
func (c *Controller) login(ctx fiber.Ctx) error {
	var req dto.LoginRequest

	if err := validators.ParseAndValidate(ctx, &req); err != nil {
		return validators.ResponseError(ctx, err)
	}

	// Get client IP and User-Agent for session tracking
	ipAddress := ctx.IP()
	userAgent := ctx.Get("User-Agent")

	response, err := c.service.Login(ctx.Context(), req, ipAddress, userAgent)
	if err != nil {
		return http_error.BadRequest(ctx, err.Error())
	}

	// Set session cookie with the token from service
	ctx.Cookie(&fiber.Cookie{
		Name:     "session_id",
		Value:    response.SessionToken,
		Expires:  time.Now().Add(constants.SessionDuration),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		Path:     "/",
	})

	return ctx.Status(fiber.StatusOK).JSON(response)
}

// logout handles POST /auth/logout request
// @Summary User logout
// @Description Invalidates the current session by deleting it from the database and clearing the session cookie.
// @Tags auth
// @Success 200 {object} map[string]string
// @Failure 401 {object} http_error.ErrorResponse
// @Router /auth/logout [post]
func (c *Controller) logout(ctx fiber.Ctx) error {
	// Get session token from cookie
	sessionToken := ctx.Cookies("session_id")
	if sessionToken == "" {
		return http_error.Unauthorized(ctx, "No active session")
	}

	// Delete session from database
	if err := c.service.Logout(ctx.Context(), sessionToken); err != nil {
		return http_error.InternalServerError(ctx, err.Error())
	}

	// Clear session cookie by setting expired
	ctx.Cookie(&fiber.Cookie{
		Name:     "session_id",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour), // Past expiration
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		Path:     "/",
	})

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Logout successful",
	})
}

// session handles GET /auth/session request
// @Summary Get current session
// @Description Returns the current active session details.
// @Tags auth
// @Produce json
// @Success 200 {object} models.Session
// @Failure 401 {object} http_error.ErrorResponse
// @Router /auth/session [get]
func (c *Controller) session(ctx fiber.Ctx) error {
	// Get session token from cookie
	sessionToken := ctx.Cookies("session_id")
	if sessionToken == "" {
		return http_error.Unauthorized(ctx, "No active session")
	}

	// Find session
	session, err := c.service.FindSessionBySessionId(ctx.Context(), sessionToken)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return http_error.Unauthorized(ctx, "Session not found or invalid")
		}
		return http_error.InternalServerError(ctx, err.Error())
	}

	// Check if session is valid
	if !session.IsValid() {
		return http_error.Unauthorized(ctx, "Session expired")
	}

	return ctx.Status(fiber.StatusOK).JSON(session)
}
