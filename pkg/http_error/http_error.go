package http_error

import "github.com/gofiber/fiber/v3"

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// Response sends an error response with the given status code and message
func Response(ctx fiber.Ctx, status int, message string) error {
	return ctx.Status(status).JSON(ErrorResponse{
		Status:  status,
		Message: message,
	})
}

// BadRequest returns a 400 Bad Request error
func BadRequest(ctx fiber.Ctx, message string) error {
	return Response(ctx, fiber.StatusBadRequest, message)
}

// Unauthorized returns a 401 Unauthorized error
func Unauthorized(ctx fiber.Ctx, message string) error {
	if message == "" {
		message = "Unauthorized"
	}
	return Response(ctx, fiber.StatusUnauthorized, message)
}

// Forbidden returns a 403 Forbidden error
func Forbidden(ctx fiber.Ctx, message string) error {
	if message == "" {
		message = "Forbidden"
	}
	return Response(ctx, fiber.StatusForbidden, message)
}

// NotFound returns a 404 Not Found error
func NotFound(ctx fiber.Ctx, message string) error {
	if message == "" {
		message = "Resource not found"
	}
	return Response(ctx, fiber.StatusNotFound, message)
}

// Conflict returns a 409 Conflict error
func Conflict(ctx fiber.Ctx, message string) error {
	return Response(ctx, fiber.StatusConflict, message)
}

// UnprocessableEntity returns a 422 Unprocessable Entity error
func UnprocessableEntity(ctx fiber.Ctx, message string) error {
	return Response(ctx, fiber.StatusUnprocessableEntity, message)
}

// TooManyRequests returns a 429 Too Many Requests error (Rate Limit)
func TooManyRequests(ctx fiber.Ctx, message string) error {
	if message == "" {
		message = "Too many requests"
	}
	return Response(ctx, fiber.StatusTooManyRequests, message)
}

// InternalServerError returns a 500 Internal Server Error
func InternalServerError(ctx fiber.Ctx, message string) error {
	if message == "" {
		message = "Internal server error"
	}
	return Response(ctx, fiber.StatusInternalServerError, message)
}

// BadGateway returns a 502 Bad Gateway error
func BadGateway(ctx fiber.Ctx, message string) error {
	if message == "" {
		message = "Bad gateway"
	}
	return Response(ctx, fiber.StatusBadGateway, message)
}

// ServiceUnavailable returns a 503 Service Unavailable error
func ServiceUnavailable(ctx fiber.Ctx, message string) error {
	if message == "" {
		message = "Service unavailable"
	}
	return Response(ctx, fiber.StatusServiceUnavailable, message)
}

// Custom returns an error response with a custom status code
func Custom(ctx fiber.Ctx, status int, message string) error {
	return Response(ctx, status, message)
}
