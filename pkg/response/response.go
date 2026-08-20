package response

import (
	"errors"
	"math"
	"net/http"

	"fiberest/internal/middlewares"
	"fiberest/pkg/apperror"

	"github.com/gofiber/fiber/v3"
)

// Envelope is the standard API response envelope.
type Envelope struct {
	Success bool         `json:"success"`
	Data    any          `json:"data,omitempty"`
	Error   *ErrorDetail `json:"error,omitempty"`
	Meta    *Meta        `json:"meta,omitempty"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type Meta struct {
	RequestID  string          `json:"requestId,omitempty"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

func requestID(c fiber.Ctx) string { return middlewares.GetRequestID(c) }

func buildMeta(c fiber.Ctx, pagination *PaginationMeta) *Meta {
	rid := requestID(c)
	if rid == "" && pagination == nil {
		return nil
	}
	m := &Meta{RequestID: rid, Pagination: pagination}
	return m
}

func ok(c fiber.Ctx, status int, data any, pagination *PaginationMeta) error {
	return c.Status(status).JSON(Envelope{
		Success: true,
		Data:    data,
		Meta:    buildMeta(c, pagination),
	})
}

// OK writes 200 envelope with data.
func OK(c fiber.Ctx, data any) error { return ok(c, http.StatusOK, data, nil) }

// Created writes 201 envelope with data.
func Created(c fiber.Ctx, data any) error { return ok(c, http.StatusCreated, data, nil) }

// NoContent writes 204 with no body (no envelope per HTTP spec).
func NoContent(c fiber.Ctx) error { return c.Status(http.StatusNoContent).SendString("") }

// Paginated writes 200 envelope with pagination meta. totalPages = ceil(total/limit).
func Paginated(c fiber.Ctx, data any, page, limit int, total int64) error {
	var totalPages int
	if limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}
	meta := &PaginationMeta{Page: page, Limit: limit, Total: total, TotalPages: totalPages}
	return ok(c, http.StatusOK, data, meta)
}

// Error writes the envelope for err with status derived from *apperror.AppError or 500.
// Always includes requestId when available from c.Locals.
func Error(c fiber.Ctx, err error) error {
	var ae *apperror.AppError
	if errors.As(err, &ae) {
		status := ae.Status
		if status == 0 {
			status = http.StatusInternalServerError
		}
		code := ae.Code
		if code == "" {
			code = "INTERNAL_ERROR"
		}
		return c.Status(status).JSON(Envelope{
			Success: false,
			Error:   &ErrorDetail{Code: code, Message: ae.Message, Details: ae.Details},
			Meta:    buildMeta(c, nil),
		})
	}
	return c.Status(http.StatusInternalServerError).JSON(Envelope{
		Success: false,
		Error:   &ErrorDetail{Code: "INTERNAL_ERROR", Message: err.Error()},
		Meta:    buildMeta(c, nil),
	})
}

// ValidationError is a convenience for validation failures with details.
func ValidationError(c fiber.Ctx, msg string, details any) error {
	return Error(c, apperror.ValidationFailed(msg, details))
}
