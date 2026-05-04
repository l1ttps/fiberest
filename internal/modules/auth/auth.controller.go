package auth

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"fiberest/internal/common/validators"
	"fiberest/internal/middlewares"
	"fiberest/internal/modules/auth/dto"
	"fiberest/pkg/http_error"
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

	// POST /auth/change-password - Change password
	auth.Post("/change-password", middlewares.Limiter(5, 60), controller.changePassword)

	// GET /auth/session - Current session
	auth.Get("/session", controller.session)

	// GET /auth/sessions - Get many sessions for a user
	auth.Get("/sessions", controller.getSessions)

	// DELETE /auth/sessions/:id - Revoke a session by ID
	auth.Delete("/sessions/:id", controller.revokeSession)
}

// initAdmin handles POST /auth/init request
// @Summary Initialize first administrator account
// @Description Creates the very first administrator (admin) account for the system using the provided email and password. This endpoint can only be called successfully once, when no admin accounts exist in the system yet.
// @Tags Auth
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
// @Description Authenticates a user using their email and password credentials. On successful authentication, creates a session and sets an HTTP-only cookie for subsequent authenticated requests. Set "remember" to true for extended session duration (30 days).
// @Tags Auth
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

	// Parse expiration time from response
	expiresAt, err := time.Parse(time.RFC3339, response.ExpiresAt)
	if err != nil {
		return http_error.InternalServerError(ctx, "invalid expiration format")
	}

	// Set session cookie with the token from service
	ctx.Cookie(&fiber.Cookie{
		Name:     "session_id",
		Value:    response.SessionToken,
		Expires:  expiresAt,
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
// @Tags Auth
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
// @Tags Auth
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

// changePassword handles POST /auth/change-password request
// @Summary Change user password
// @Description Changes the authenticated user's password. Requires the current password to be verified before setting the new password.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.ChangePasswordRequest true "Change password request"
// @Success 200 {object} dto.ChangePasswordResponse
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 401 {object} http_error.ErrorResponse
// @Router /auth/change-password [post]
func (c *Controller) changePassword(ctx fiber.Ctx) error {
	sessionToken := ctx.Cookies("session_id")
	if sessionToken == "" {
		return http_error.Unauthorized(ctx, "No active session")
	}

	session, err := c.service.FindSessionBySessionId(ctx.Context(), sessionToken)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return http_error.Unauthorized(ctx, "Session not found or invalid")
		}
		return http_error.InternalServerError(ctx, err.Error())
	}

	if !session.IsValid() {
		return http_error.Unauthorized(ctx, "Session expired")
	}

	var req dto.ChangePasswordRequest
	if err := validators.ParseAndValidate(ctx, &req); err != nil {
		return validators.ResponseError(ctx, err)
	}

	response, err := c.service.ChangePassword(ctx.Context(), session.UserID, req)
	if err != nil {
		if errors.Is(err, ErrWrongPassword) {
			return http_error.BadRequest(ctx, err.Error())
		}
		return http_error.InternalServerError(ctx, err.Error())
	}

	return ctx.Status(fiber.StatusOK).JSON(response)
}

// getSessions handles GET /auth/sessions request
// @Summary Get many sessions
// @Description Retrieves a paginated list of sessions for the currently authenticated user.
// @Tags Auth
// @Produce json
// @Param request query dto.GetManySessionsRequest true "Get sessions request"
// @Success 200 {object} dto.GetManySessionsResponse
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 401 {object} http_error.ErrorResponse
// @Router /auth/sessions [get]
func (c *Controller) getSessions(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("user_id").(string)
	if !ok || userID == "" {
		return http_error.Unauthorized(ctx, "User not authenticated")
	}

	var req dto.GetManySessionsRequest

	// Parse and validate query parameters
	if err := validators.GetQuery(ctx, &req); err != nil {
		return validators.ResponseError(ctx, err)
	}

	// Set default values if not provided
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Page <= 0 {
		req.Page = 1
	}

	sessions, total, err := c.service.FindSessionsByUserID(ctx.Context(), userID, req.Limit, req.Page)
	if err != nil {
		return http_error.InternalServerError(ctx, err.Error())
	}

	data := make([]dto.SessionResponse, len(sessions))
	for i, s := range sessions {
		data[i] = dto.SessionResponse{
			ID:         s.ID.String(),
			UserID:     s.UserID.String(),
			ExpiresAt:  s.ExpiresAt,
			IPAddress:  s.IPAddress,
			UserAgent:  s.UserAgent,
			RememberMe: s.RememberMe,
			CreatedAt:  s.CreatedAt,
		}
	}

	hasNextPage := int64(req.Page*req.Limit) < total

	return ctx.Status(fiber.StatusOK).JSON(dto.GetManySessionsResponse{
		Data:        data,
		Limit:       req.Limit,
		Page:        req.Page,
		HasNextPage: hasNextPage,
		Total:       total,
	})
}

// revokeSession handles DELETE /auth/sessions/:id request
// @Summary Revoke a session by ID
// @Description Deletes a specific session for the authenticated user. The session must belong to the current user.
// @Tags Auth
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 401 {object} http_error.ErrorResponse
// @Router /auth/sessions/{id} [delete]
func (c *Controller) revokeSession(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("user_id").(string)
	if !ok || userID == "" {
		return http_error.Unauthorized(ctx, "User not authenticated")
	}

	sessionID := ctx.Params("id")
	if sessionID == "" {
		return http_error.BadRequest(ctx, "Session ID is required")
	}

	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return http_error.BadRequest(ctx, "Invalid session ID format")
	}

	session, err := c.service.FindSessionByID(ctx.Context(), sessionUUID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return http_error.BadRequest(ctx, "Session not found")
		}
		return http_error.InternalServerError(ctx, err.Error())
	}

	if session.UserID.String() != userID {
		return http_error.BadRequest(ctx, "Session does not belong to user")
	}

	if err := c.service.DeleteSessionByID(ctx.Context(), sessionUUID); err != nil {
		return http_error.InternalServerError(ctx, err.Error())
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Session revoked successfully",
	})
}
