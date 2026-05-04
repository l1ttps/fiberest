package middlewares

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fiberest/internal/models"
	"fiberest/pkg/http_error"

	"github.com/gofiber/fiber/v3"
)

// AuthService interface defines required methods for auth guard
type AuthService interface {
	FindSessionBySessionId(ctx context.Context, sessionToken string) (*models.Session, error)
	UpdateExpiresSession(ctx context.Context, session *models.Session) error
	DeleteSession(ctx context.Context, sessionToken string) error
}

// publicRoutes defines the list of paths that do not require authentication
var publicRoutes = []string{
	"health-check",
	"/auth/login",
	"/auth/init",
	"/docs/*",
}

// isPublicRoute checks if the current path is listed in publicRoutes
func isPublicRoute(path string) bool {
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		path = "/"
	}

	for _, route := range publicRoutes {
		routeNormalized := strings.TrimSuffix(route, "/")
		if routeNormalized == "" {
			routeNormalized = "/"
		}

		if routeNormalized == path {
			return true
		}
		if strings.HasSuffix(routeNormalized, "*") && strings.HasPrefix(path, strings.TrimSuffix(routeNormalized, "*")) {
			return true
		}
		if strings.HasPrefix(routeNormalized, "*") && strings.HasSuffix(path, strings.TrimPrefix(routeNormalized, "*")) {
			return true
		}
	}
	return false
}

// AuthGuard validates the session token from the session_id cookie
func AuthGuard(authService AuthService) fiber.Handler {
	return func(c fiber.Ctx) error {
		if isPublicRoute(c.Path()) {
			return c.Next()
		}

		// Extract session token from cookie
		sessionToken := c.Cookies("session_id")
		if sessionToken == "" {
			return http_error.Unauthorized(c, "Missing session token")
		}

		// Find session without validation
		session, err := authService.FindSessionBySessionId(c.Context(), sessionToken)
		if err != nil {
			return http_error.Unauthorized(c, "Invalid session")
		}

		// Auto-extend remember-me session if eligible (best effort, ignore errors)
		_ = authService.UpdateExpiresSession(c.Context(), session)

		// Validate session after potential extension
		if !session.IsValid() {
			// Delete expired session
			_ = authService.DeleteSession(c.Context(), sessionToken)
			return http_error.Unauthorized(c, "Session expired")
		}

		// Check if user is banned
		if session.User.BanUntil != nil && time.Now().Before(*session.User.BanUntil) {
			return http_error.Forbidden(c, fmt.Sprintf("Your account is banned until %s. Reason: %s", session.User.BanUntil.Format(time.RFC3339), session.User.BanReason))
		}

		// Store user info in context for downstream handlers
		c.Locals("user", &session.User)
		c.Locals("user_id", session.UserID.String())

		return c.Next()
	}
}

// RoleGuard ensures the authenticated user has one of the required roles
func RoleGuard(allowedRoles ...models.UserRole) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, ok := c.Locals("user").(*models.User)
		if !ok {
			return http_error.Unauthorized(c, "User not authenticated")
		}

		for _, role := range allowedRoles {
			if user.Role == role {
				return c.Next()
			}
		}

		return http_error.Forbidden(c, fmt.Sprintf("Role %s is not allowed", user.Role))
	}
}
