package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError is a structured application error with HTTP semantics.
type AppError struct {
	Code    string
	Message string
	Status  int
	Details any
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	if e.Code != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Cause }

// New creates an AppError without a cause.
func New(code string, status int, msg string) *AppError {
	return &AppError{Code: code, Status: status, Message: msg}
}

// Wrap creates an AppError wrapping a cause.
func Wrap(cause error, code string, status int, msg string) *AppError {
	return &AppError{Code: code, Status: status, Message: msg, Cause: cause}
}

// WithDetails attaches details payload (e.g. validation field errors).
func (e *AppError) WithDetails(details any) *AppError {
	e.Details = details
	return e
}

// MapStatus returns HTTP status for err (AppError.Status or 500).
func MapStatus(err error) int {
	var ae *AppError
	if errors.As(err, &ae) {
		if ae.Status != 0 {
			return ae.Status
		}
	}
	return http.StatusInternalServerError
}

// CodeOf returns Code when err is *AppError.
func CodeOf(err error) string {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae.Code
	}
	return "INTERNAL_ERROR"
}

// Helpers

func BadRequest(msg string) *AppError {
	return New("BAD_REQUEST", http.StatusBadRequest, msg)
}
func Unauthorized(msg string) *AppError {
	if msg == "" {
		msg = "Unauthorized"
	}
	return New("UNAUTHORIZED", http.StatusUnauthorized, msg)
}
func Forbidden(msg string) *AppError {
	if msg == "" {
		msg = "Forbidden"
	}
	return New("FORBIDDEN", http.StatusForbidden, msg)
}
func NotFound(msg string) *AppError {
	if msg == "" {
		msg = "Resource not found"
	}
	return New("NOT_FOUND", http.StatusNotFound, msg)
}
func Conflict(msg string) *AppError {
	return New("CONFLICT", http.StatusConflict, msg)
}
func UnprocessableEntity(msg string) *AppError {
	return New("UNPROCESSABLE_ENTITY", http.StatusUnprocessableEntity, msg)
}
func TooManyRequests(msg string) *AppError {
	if msg == "" {
		msg = "Too many requests"
	}
	return New("TOO_MANY_REQUESTS", http.StatusTooManyRequests, msg)
}
func Internal(msg string) *AppError {
	if msg == "" {
		msg = "Internal server error"
	}
	return New("INTERNAL_ERROR", http.StatusInternalServerError, msg)
}
func BadGateway(msg string) *AppError {
	if msg == "" {
		msg = "Bad gateway"
	}
	return New("BAD_GATEWAY", http.StatusBadGateway, msg)
}
func ServiceUnavailable(msg string) *AppError {
	if msg == "" {
		msg = "Service unavailable"
	}
	return New("SERVICE_UNAVAILABLE", http.StatusServiceUnavailable, msg)
}

// ValidationFailed builds a 400 with structured field details.
func ValidationFailed(msg string, details any) *AppError {
	if msg == "" {
		msg = "Validation failed"
	}
	return (&AppError{Code: "VALIDATION_ERROR", Status: http.StatusBadRequest, Message: msg}).WithDetails(details)
}
