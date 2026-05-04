// Package validators provides reusable validation utilities using go-playground/validator
package validators

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"fiberest/pkg/http_error"

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

// GetBody binds request body and validates the struct
func GetBody(ctx fiber.Ctx, dest interface{}) error {
	return ParseAndValidate(ctx, dest)
}

// GetQuery binds query parameters and validates the struct
func GetQuery(ctx fiber.Ctx, dest interface{}) error {
	if err := ctx.Bind().Query(dest); err != nil {
		return err
	}
	return ValidateStruct(dest)
}

// GetParam binds URL path parameters and validates the struct
// Uses struct tags with `param:"key"` to map path parameters to struct fields
func GetParam(ctx fiber.Ctx, dest interface{}) error {
	val := reflect.ValueOf(dest)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	
	if val.Kind() != reflect.Struct {
		return errors.New("dest must be a struct or struct pointer")
	}
	
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		paramKey := field.Tag.Get("param")
		if paramKey == "" {
			// Use field name as param key if tag not specified
			paramKey = field.Name
		}
		
		paramValue := ctx.Params(paramKey)
		if paramValue == "" {
			// Check if field is required via validate tag
			if validateTag := field.Tag.Get("validate"); strings.Contains(validateTag, "required") {
				return fmt.Errorf("%s is required", paramKey)
			}
			// Continue if not required
			continue
		}
		
		// Set the field value
		if err := setFieldValue(val.Field(i), paramValue, field.Type); err != nil {
			return fmt.Errorf("failed to set param %s: %w", paramKey, err)
		}
	}
	
	return ValidateStruct(dest)
}

// setFieldValue sets a reflect.Value from a string based on the field's type
func setFieldValue(field reflect.Value, value string, typ reflect.Type) error {
	if !field.CanSet() {
		return fmt.Errorf("cannot set field")
	}
	
	switch typ.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, typ.Bits())
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)
	default:
		return fmt.Errorf("unsupported type: %s", typ.Kind())
	}
	
	return nil
}

func ResponseError(ctx fiber.Ctx, err error) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(http_error.ErrorResponse{
		Status:  fiber.StatusBadRequest,
		Message: err.Error(),
	})
}
