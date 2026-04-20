package middlewares

import (
	"fmt"
	"strings"

	"fiberest/internal/configs"
	"fiberest/pkg/http_error"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

// publicRoutes defines the list of paths that do not require authentication
var publicRoutes = []string{
	"/users/login",
	"/swagger*",
	"/admin/init",
}

// isPublicRoute checks if the current path is listed in publicRoutes, supporting trailing or leading wildcards
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

// AuthGuard validates the JWT token from the Authorization header
func AuthGuard(cfg *configs.Config) fiber.Handler {
	return func(c fiber.Ctx) error {
		if isPublicRoute(c.Path()) {
			return c.Next()
		}

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return http_error.Unauthorized(c, "Missing authorization header")
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return http_error.Unauthorized(c, "Invalid authorization header format")
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(cfg.GetString("TOKEN_SECRET")), nil
		})

		if err != nil || !token.Valid {
			return http_error.Unauthorized(c, "Invalid or expired token")
		}

		return c.Next()
	}
}
