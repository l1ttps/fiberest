package auth

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"fiberest/internal/common/types"
	"fiberest/internal/common/validators"
	"fiberest/internal/configs"
	"fiberest/internal/middlewares"
	"fiberest/internal/models"
	"fiberest/internal/modules/auth/dto"
	"fiberest/pkg/http_error"
)

// Controller handles authentication-related requests
type Controller struct {
	service AuthService
	cfg     *configs.Config
}

// NewController creates a new auth controller
func NewController(service AuthService, cfg *configs.Config) *Controller {
	return &Controller{service: service, cfg: cfg}
}

// RegisterRoutes is invoked by fx to register auth routes.
// Public routes are registered without the guard; everything else sits behind it,
// so no path allow-list is needed.
func RegisterRoutes(app *fiber.App, controller *Controller) {
	auth := app.Group("/auth")

	auth.Post("/init", controller.initAdmin)
	auth.Post("/login", middlewares.LoginLimiter(), controller.login)

	protected := auth.Group("/", middlewares.AuthGuard(controller.service))
	protected.Post("/logout", middlewares.Limiter(5, 60), controller.logout)
	protected.Post("/change-password", middlewares.Limiter(5, 60), controller.changePassword)
	protected.Get("/session", controller.session)
	protected.Get("/sessions", controller.getSessions)
	protected.Delete("/sessions/:id", controller.revokeSession)
}

// mapError translates service sentinels into HTTP errors. Unknown errors become
// a generic 500 whose cause is logged, never returned to the client.
func mapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrInvalidCredential):
		return http_error.Unauthorized(err.Error())
	case errors.Is(err, ErrWrongPassword):
		return http_error.BadRequest(err.Error())
	case errors.Is(err, ErrAdminExists):
		return http_error.BadRequest(err.Error())
	case errors.Is(err, ErrAccountBanned):
		return http_error.Forbidden(err.Error())
	case errors.Is(err, ErrUserNotFound):
		return http_error.NotFound(err.Error())
	case errors.Is(err, ErrSessionNotFound):
		return http_error.Unauthorized("Session not found or invalid")
	default:
		return http_error.Internal(err)
	}
}

func (c *Controller) setSessionCookie(ctx fiber.Ctx, token string, expiresAt time.Time) {
	ctx.Cookie(&fiber.Cookie{
		Name:     middlewares.SessionCookie,
		Value:    token,
		Expires:  expiresAt,
		HTTPOnly: true,
		Secure:   c.cfg.Auth.CookieSecure,
		SameSite: "Lax",
		Path:     "/",
	})
}

func (c *Controller) clearSessionCookie(ctx fiber.Ctx) {
	ctx.Cookie(&fiber.Cookie{
		Name:     middlewares.SessionCookie,
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		Secure:   c.cfg.Auth.CookieSecure,
		SameSite: "Lax",
		Path:     "/",
	})
}

func toSessionResponse(s *models.Session) dto.SessionResponse {
	return dto.SessionResponse{
		ID:         s.ID.String(),
		UserID:     s.UserID.String(),
		ExpiresAt:  s.ExpiresAt,
		IPAddress:  s.IPAddress,
		UserAgent:  s.UserAgent,
		RememberMe: s.RememberMe,
		CreatedAt:  s.CreatedAt,
	}
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
	if err := validators.ParseAndValidate(ctx, &req); err != nil {
		return validators.Invalid(err)
	}

	response, err := c.service.CreateAdmin(ctx.Context(), req)
	if err != nil {
		return mapError(err)
	}

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
// @Failure 429 {object} http_error.ErrorResponse
// @Router /auth/login [post]
func (c *Controller) login(ctx fiber.Ctx) error {
	var req dto.LoginRequest
	if err := validators.ParseAndValidate(ctx, &req); err != nil {
		return validators.Invalid(err)
	}

	response, err := c.service.Login(ctx.Context(), req, ctx.IP(), ctx.Get("User-Agent"))
	if err != nil {
		return mapError(err)
	}

	expiresAt, err := time.Parse(time.RFC3339, response.ExpiresAt)
	if err != nil {
		return http_error.Internal(err)
	}
	c.setSessionCookie(ctx, response.SessionToken, expiresAt)

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
	session, ok := middlewares.Session(ctx)
	if !ok {
		return http_error.Unauthorized("No active session")
	}

	if err := c.service.Logout(ctx.Context(), session.SessionToken); err != nil {
		return http_error.Internal(err)
	}

	c.clearSessionCookie(ctx)

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Logout successful"})
}

// session handles GET /auth/session request
// @Summary Get current session
// @Description Returns the current active session details. The session token is never included.
// @Tags Auth
// @Produce json
// @Success 200 {object} dto.SessionResponse
// @Failure 401 {object} http_error.ErrorResponse
// @Router /auth/session [get]
func (c *Controller) session(ctx fiber.Ctx) error {
	session, ok := middlewares.Session(ctx)
	if !ok {
		return http_error.Unauthorized("No active session")
	}
	return ctx.Status(fiber.StatusOK).JSON(toSessionResponse(session))
}

// changePassword handles POST /auth/change-password request
// @Summary Change user password
// @Description Changes the authenticated user's password. Requires the current password. Every other session of the user is revoked.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.ChangePasswordRequest true "Change password request"
// @Success 200 {object} dto.ChangePasswordResponse
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 401 {object} http_error.ErrorResponse
// @Router /auth/change-password [post]
func (c *Controller) changePassword(ctx fiber.Ctx) error {
	session, ok := middlewares.Session(ctx)
	if !ok {
		return http_error.Unauthorized("No active session")
	}

	var req dto.ChangePasswordRequest
	if err := validators.ParseAndValidate(ctx, &req); err != nil {
		return validators.Invalid(err)
	}

	response, err := c.service.ChangePassword(ctx.Context(), session.UserID, session.SessionToken, req)
	if err != nil {
		return mapError(err)
	}

	return ctx.Status(fiber.StatusOK).JSON(response)
}

// getSessions handles GET /auth/sessions request
// @Summary Get many sessions
// @Description Retrieves a paginated list of sessions for the currently authenticated user.
// @Tags Auth
// @Produce json
// @Param limit query int false "Items per page" minimum(1) maximum(100) default(10)
// @Param page query int false "Page number" minimum(1) default(1)
// @Success 200 {object} dto.GetManySessionsResponse
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 401 {object} http_error.ErrorResponse
// @Router /auth/sessions [get]
func (c *Controller) getSessions(ctx fiber.Ctx) error {
	userID, ok := middlewares.UserID(ctx)
	if !ok {
		return http_error.Unauthorized("User not authenticated")
	}

	var req dto.GetManySessionsRequest
	if err := validators.GetQuery(ctx, &req); err != nil {
		return validators.Invalid(err)
	}

	sessions, total, err := c.service.FindSessionsByUserID(ctx.Context(), userID, req.Limit, req.Page)
	if err != nil {
		return http_error.Internal(err)
	}

	data := make([]dto.SessionResponse, len(sessions))
	for i := range sessions {
		data[i] = toSessionResponse(&sessions[i])
	}

	return ctx.Status(fiber.StatusOK).JSON(dto.GetManySessionsResponse{
		Data:     data,
		PageMeta: types.NewPageMeta(req.Limit, req.Page, total),
	})
}

// revokeSession handles DELETE /auth/sessions/:id request
// @Summary Revoke a session by ID
// @Description Deletes a specific session for the authenticated user. The session must belong to the current user.
// @Tags Auth
// @Produce json
// @Param id path string true "Session ID (UUID)"
// @Success 200 {object} map[string]string
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 401 {object} http_error.ErrorResponse
// @Router /auth/sessions/{id} [delete]
func (c *Controller) revokeSession(ctx fiber.Ctx) error {
	userID, ok := middlewares.UserID(ctx)
	if !ok {
		return http_error.Unauthorized("User not authenticated")
	}

	var params struct {
		ID string `param:"id" validate:"required,uuid"`
	}
	if err := validators.GetParam(ctx, &params); err != nil {
		return validators.Invalid(err)
	}

	sessionUUID, err := uuid.Parse(params.ID)
	if err != nil {
		return http_error.BadRequest("Invalid session ID format")
	}

	session, err := c.service.FindSessionByID(ctx.Context(), sessionUUID)
	if err != nil {
		return mapError(err)
	}

	if session.UserID.String() != userID {
		return http_error.BadRequest("Session does not belong to user")
	}

	if err := c.service.DeleteSessionByID(ctx.Context(), sessionUUID); err != nil {
		return http_error.Internal(err)
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Session revoked successfully"})
}
