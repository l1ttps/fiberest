package middlewares

import (
	"context"
	"fmt"
	"strings"

	"fiberest/internal/models"
	"fiberest/pkg/http_error"

	"github.com/gofiber/fiber/v3"
)

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
func AuthGuard(authService interface {
	FindValidSession(ctx context.Context, sessionToken string) (*models.Session, error)
}) fiber.Handler {
	return func(c fiber.Ctx) error {
		if isPublicRoute(c.Path()) {
			return c.Next()
		}

		// Extract session token from cookie
		sessionToken := c.Cookies("session_id")
		if sessionToken == "" {
			return http_error.Unauthorized(c, "Missing session token")
		}

		// Validate session
		session, err := authService.FindValidSession(c.Context(), sessionToken)
		if err != nil {
			return http_error.Unauthorized(c, "Invalid or expired session")
		}

		// Store user info in context for downstream handlers
		c.Locals("user", &session.User)
		c.Locals("user_id", session.UserID)

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
