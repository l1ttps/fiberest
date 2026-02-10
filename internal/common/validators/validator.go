// Package validators provides reusable validation utilities using go-playground/validator
package validators

import (
	"fiberest/pkg/http_error"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

// Validator is a singleton instance of the validator
var Validator = validator.New()

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

// Error implements the error interface
func (ve ValidationErrors) Error() string {
	var messages []string
	for _, err := range ve {
		messages = append(messages, err.Message)
	}
	return strings.Join(messages, "; ")
}

// ValidateStruct validates a struct using the singleton validator
// Returns ValidationErrors if validation fails, nil if successful
func ValidateStruct(s interface{}) error {
	if err := Validator.Struct(s); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return formatValidationErrors(validationErrors)
		}
		return err
	}
	return nil
}

// formatValidationErrors converts validator.ValidationErrors to ValidationErrors
func formatValidationErrors(errs validator.ValidationErrors) ValidationErrors {
	var errors ValidationErrors
	for _, err := range errs {
		errors = append(errors, ValidationError{
			Field:   err.Field(),
			Message: getErrorMessage(err),
		})
	}
	return errors
}

// getErrorMessage returns a user-friendly error message for a validation error
func getErrorMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return err.Field() + " is required"
	case "email":
		return err.Field() + " must be a valid email address"
	case "min":
		return err.Field() + " must be at least " + err.Param() + " characters"
	case "max":
		return err.Field() + " must be at most " + err.Param() + " characters"
	default:
		return err.Field() + " is invalid"
	}
}

func ParseAndValidate(ctx fiber.Ctx, dest interface{}) error {
	// Parse request body
	if err := ctx.Bind().Body(dest); err != nil {
		return err
	}

	// Validate parsed data
	return ValidateStruct(dest)
}

func ResponseError(ctx fiber.Ctx, err error) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(http_error.ErrorResponse{
		Status:  fiber.StatusBadRequest,
		Message: err.Error(),
	})
}
