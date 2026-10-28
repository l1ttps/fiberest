// Package validators binds and validates request data with go-playground/validator.
package validators

import (
	"strings"

	"fiberest/pkg/http_error"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

// Validator is the shared validator instance.
var Validator = validator.New()

// ValidationError is a single field-level failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors is returned by ValidateStruct and serialized as error details.
type ValidationErrors []ValidationError

// Error implements the error interface.
func (ve ValidationErrors) Error() string {
	messages := make([]string, 0, len(ve))
	for _, err := range ve {
		messages = append(messages, err.Message)
	}
	return strings.Join(messages, "; ")
}

// Defaulter is implemented by request structs that fill omitted fields.
type Defaulter interface {
	SetDefaults()
}

// ValidateStruct validates a struct, returning ValidationErrors on failure.
func ValidateStruct(s any) error {
	if err := Validator.Struct(s); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return formatValidationErrors(validationErrors)
		}
		return err
	}
	return nil
}

// Invalid converts a bind/validation failure into a 400 the error handler renders.
func Invalid(err error) error {
	return http_error.BadRequest(err.Error()).WithDetails(err)
}

func formatValidationErrors(errs validator.ValidationErrors) ValidationErrors {
	out := make(ValidationErrors, 0, len(errs))
	for _, err := range errs {
		out = append(out, ValidationError{Field: err.Field(), Message: getErrorMessage(err)})
	}
	return out
}

// getErrorMessage returns a user-friendly message for a field error.
func getErrorMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return err.Field() + " is required"
	case "email":
		return err.Field() + " must be a valid email address"
	case "uuid":
		return err.Field() + " must be a valid UUID"
	case "oneof":
		return err.Field() + " must be one of: " + strings.ReplaceAll(err.Param(), " ", ", ")
	case "min":
		return err.Field() + " must be at least " + err.Param() + " characters"
	case "max":
		return err.Field() + " must be at most " + err.Param() + " characters"
	default:
		return err.Field() + " is invalid"
	}
}

// ParseAndValidate binds the JSON body, applies defaults and validates.
func ParseAndValidate(ctx fiber.Ctx, dest any) error {
	if err := ctx.Bind().Body(dest); err != nil {
		return err
	}
	return finish(dest)
}

// GetQuery binds query parameters, applies defaults and validates.
func GetQuery(ctx fiber.Ctx, dest any) error {
	if err := ctx.Bind().Query(dest); err != nil {
		return err
	}
	return finish(dest)
}

// GetParam binds path parameters (`param:"key"` tags) and validates.
func GetParam(ctx fiber.Ctx, dest any) error {
	if err := ctx.Bind().URI(dest); err != nil {
		return err
	}
	return finish(dest)
}

func finish(dest any) error {
	if d, ok := dest.(Defaulter); ok {
		d.SetDefaults()
	}
	return ValidateStruct(dest)
}
