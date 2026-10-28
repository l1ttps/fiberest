package middlewares

import (
	"context"
	"fmt"
	"time"

	"fiberest/internal/models"
	"fiberest/pkg/http_error"

	"github.com/gofiber/fiber/v3"
)

// AuthService is the slice of the auth service the guard needs.
// auth.AuthService satisfies it structurally, so no import cycle.
type AuthService interface {
	FindSessionBySessionId(ctx context.Context, sessionToken string) (*models.Session, error)
	UpdateExpiresSession(ctx context.Context, session *models.Session) error
	DeleteSession(ctx context.Context, sessionToken string) error
}

// SessionCookie is the cookie that carries the session token.
const SessionCookie = "session_id"

const (
	localUserKey    = "user"
	localUserIDKey  = "user_id"
	localSessionKey = "session"
)

// AuthGuard rejects requests without a valid, unexpired session.
// It is applied per route group, so there is no path allow-list to maintain.
func AuthGuard(authService AuthService) fiber.Handler {
	return func(c fiber.Ctx) error {
		sessionToken := c.Cookies(SessionCookie)
		if sessionToken == "" {
			return http_error.Unauthorized("Missing session token")
		}

		session, err := authService.FindSessionBySessionId(c.Context(), sessionToken)
		if err != nil {
			return http_error.Unauthorized("Invalid session")
		}

		// Auto-extend remember-me session if eligible (best effort).
		_ = authService.UpdateExpiresSession(c.Context(), session)

		if !session.IsValid() {
			_ = authService.DeleteSession(c.Context(), sessionToken)
			return http_error.Unauthorized("Session expired")
		}

		if session.User.BanUntil != nil && time.Now().Before(*session.User.BanUntil) {
			return http_error.Forbidden(fmt.Sprintf(
				"Your account is banned until %s. Reason: %s",
				session.User.BanUntil.Format(time.RFC3339), session.User.BanReason))
		}

		c.Locals(localUserKey, &session.User)
		c.Locals(localUserIDKey, session.UserID.String())
		c.Locals(localSessionKey, session)

		return c.Next()
	}
}

// RoleGuard ensures the authenticated user has one of the required roles.
func RoleGuard(allowedRoles ...models.UserRole) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, ok := User(c)
		if !ok {
			return http_error.Unauthorized("User not authenticated")
		}
		for _, role := range allowedRoles {
			if user.Role == role {
				return c.Next()
			}
		}
		return http_error.Forbidden(fmt.Sprintf("Role %s is not allowed", user.Role))
	}
}

// User returns the authenticated user set by AuthGuard.
func User(c fiber.Ctx) (*models.User, bool) {
	user, ok := c.Locals(localUserKey).(*models.User)
	return user, ok && user != nil
}

// UserID returns the authenticated user's ID set by AuthGuard.
func UserID(c fiber.Ctx) (string, bool) {
	id, ok := c.Locals(localUserIDKey).(string)
	return id, ok && id != ""
}

// Session returns the authenticated session set by AuthGuard.
func Session(c fiber.Ctx) (*models.Session, bool) {
	session, ok := c.Locals(localSessionKey).(*models.Session)
	return session, ok && session != nil
}
