// Package http_error defines the single error type used across the app and the
// Fiber error handler that renders it.
//
// Handlers return errors instead of writing responses, so internal details
// (SQL errors, DSNs, stack traces) never reach clients: the handler logs the
// full cause and emits a generic message.
package http_error

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// RequestIDHeader mirrors middlewares.RequestIDHeader. Duplicated on purpose:
// pkg must not import internal.
const RequestIDHeader = "X-Request-ID"

// ErrorResponse is the JSON body returned for every failed request.
type ErrorResponse struct {
	Status    int    `json:"status"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
	Details   any    `json:"details,omitempty"`
}

// Error is an error carrying HTTP semantics.
type Error struct {
	Status  int
	Code    string
	Message string
	Details any
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap exposes the cause to errors.Is/As.
func (e *Error) Unwrap() error { return e.Cause }

// WithDetails attaches a machine-readable payload (e.g. field errors).
func (e *Error) WithDetails(details any) *Error {
	e.Details = details
	return e
}

// New builds an Error.
func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// Wrap builds an Error around a cause.
func Wrap(cause error, status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message, Cause: cause}
}

func fallback(msg, def string) string {
	if msg == "" {
		return def
	}
	return msg
}

// 4xx helpers. Messages reach the client, so keep them safe.

func BadRequest(msg string) *Error {
	return New(http.StatusBadRequest, "BAD_REQUEST", fallback(msg, "Bad request"))
}

func Unauthorized(msg string) *Error {
	return New(http.StatusUnauthorized, "UNAUTHORIZED", fallback(msg, "Unauthorized"))
}

func Forbidden(msg string) *Error {
	return New(http.StatusForbidden, "FORBIDDEN", fallback(msg, "Forbidden"))
}

func NotFound(msg string) *Error {
	return New(http.StatusNotFound, "NOT_FOUND", fallback(msg, "Resource not found"))
}

func Conflict(msg string) *Error {
	return New(http.StatusConflict, "CONFLICT", fallback(msg, "Conflict"))
}

func TooManyRequests(msg string) *Error {
	return New(http.StatusTooManyRequests, "TOO_MANY_REQUESTS", fallback(msg, "Too many requests"))
}

// Internal wraps a cause that must never be shown to the client.
func Internal(cause error) *Error {
	return Wrap(cause, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
}

func requestID(c fiber.Ctx) string {
	if id := c.GetRespHeader(RequestIDHeader); id != "" {
		return id
	}
	return c.Get(RequestIDHeader)
}

func write(c fiber.Ctx, e *Error) error {
	status := e.Status
	if status == 0 {
		status = http.StatusInternalServerError
	}
	code := e.Code
	if code == "" {
		code = "INTERNAL_ERROR"
	}
	return c.Status(status).JSON(ErrorResponse{
		Status:    status,
		Code:      code,
		Message:   e.Message,
		RequestID: requestID(c),
		Details:   e.Details,
	})
}

// NewHandler returns the app-wide fiber.ErrorHandler. Unknown errors become a
// generic 500 and are logged with the cause; *Error values render as-is.
func NewHandler(logger *slog.Logger) fiber.ErrorHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(c fiber.Ctx, err error) error {
		var e *Error
		if errors.As(err, &e) {
			if e.Status >= http.StatusInternalServerError {
				logger.Error("request failed",
					slog.String("error", e.Error()),
					slog.String("method", c.Method()),
					slog.String("path", c.Path()),
					slog.String("request_id", requestID(c)),
				)
			}
			return write(c, e)
		}

		// Fiber's own errors (404 unmatched route, 405, body too large) keep
		// their status; anything else is an unexpected 500.
		var fe *fiber.Error
		if errors.As(err, &fe) {
			return write(c, New(fe.Code, codeFromStatus(fe.Code), fe.Message))
		}

		logger.Error("unhandled error",
			slog.String("error", err.Error()),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.String("request_id", requestID(c)),
		)
		return write(c, Internal(err))
	}
}

func codeFromStatus(status int) string {
	if s := http.StatusText(status); s != "" {
		return strings.ToUpper(strings.ReplaceAll(s, " ", "_"))
	}
	return "ERROR"
}
