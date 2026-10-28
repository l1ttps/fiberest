package validators

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"fiberest/internal/common/types"
	"fiberest/pkg/http_error"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// TestValidateStruct tests the ValidateStruct function
type TestStruct struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
	Age   int    `validate:"min=18,max=100"`
}

func TestValidateStruct(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name: "valid struct",
			input: TestStruct{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   25,
			},
			wantErr: false,
		},
		{
			name: "missing required field",
			input: TestStruct{
				Email: "john@example.com",
				Age:   25,
			},
			wantErr: true,
		},
		{
			name: "invalid email format",
			input: TestStruct{
				Name:  "John Doe",
				Email: "invalid-email",
				Age:   25,
			},
			wantErr: true,
		},
		{
			name: "age below minimum",
			input: TestStruct{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   16,
			},
			wantErr: true,
		},
		{
			name: "age above maximum",
			input: TestStruct{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   150,
			},
			wantErr: true,
		},
		{
			name: "all fields invalid",
			input: TestStruct{
				Name:  "",
				Email: "not-an-email",
				Age:   5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				// Verify it's a ValidationErrors type
				var validationErr ValidationErrors
				assert.True(t, errors.As(err, &validationErr))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestFormatValidationErrors tests the formatValidationErrors function
func TestFormatValidationErrors(t *testing.T) {
	// This test requires creating validator.ValidationErrors
	// Since it's internal to the validator package, we'll test through ValidateStruct
	t.Run("multiple validation errors", func(t *testing.T) {
		input := TestStruct{
			Name:  "",
			Email: "invalid",
			Age:   5,
		}

		err := ValidateStruct(input)
		assert.Error(t, err)

		var validationErr ValidationErrors
		require.True(t, errors.As(err, &validationErr))
		assert.NotEmpty(t, validationErr)
		assert.Greater(t, len(validationErr), 1)

		// Check that each error has a field and message
		for _, e := range validationErr {
			assert.NotEmpty(t, e.Field)
			assert.NotEmpty(t, e.Message)
		}
	})
}

// TestValidationErrorMethods tests ValidationError methods
type TestStructSimple struct {
	Field1 string `validate:"required"`
	Field2 string `validate:"required"`
}

func TestValidationErrorMethods(t *testing.T) {
	t.Run("Error method returns concatenated messages", func(t *testing.T) {
		input := TestStructSimple{
			Field1: "",
			Field2: "",
		}

		err := ValidateStruct(input)
		assert.Error(t, err)

		var validationErr ValidationErrors
		require.True(t, errors.As(err, &validationErr))

		errorMsg := validationErr.Error()
		assert.Contains(t, errorMsg, "Field1")
		assert.Contains(t, errorMsg, "Field2")
		assert.Contains(t, errorMsg, "is required")
	})

	// Test ValidationErrors slice behavior
	t.Run("empty validation errors", func(t *testing.T) {
		var emptyErr ValidationErrors
		assert.Empty(t, emptyErr.Error())
	})
}

// TestGetErrorMessage tests the getErrorMessage function indirectly
type TestStructTags struct {
	RequiredField string `validate:"required"`
	EmailField    string `validate:"email"`
	MinField      string `validate:"min=5"`
	MaxField      string `validate:"max=10"`
}

func TestGetErrorMessage(t *testing.T) {
	tests := []struct {
		name          string
		input         interface{}
		expectedField string
		expectedMsg   string
	}{
		{
			name: "required field error",
			input: TestStructTags{
				EmailField: "test@example.com",
				MinField:   "12345",
				MaxField:   "1234567890",
			},
			expectedField: "RequiredField",
			expectedMsg:   "RequiredField is required",
		},
		{
			name: "email field error",
			input: TestStructTags{
				RequiredField: "test",
				EmailField:    "not-an-email",
				MinField:      "12345",
				MaxField:      "1234567890",
			},
			expectedField: "EmailField",
			expectedMsg:   "EmailField must be a valid email address",
		},
		{
			name: "min field error",
			input: TestStructTags{
				RequiredField: "test",
				EmailField:    "test@example.com",
				MinField:      "123",
				MaxField:      "1234567890",
			},
			expectedField: "MinField",
			expectedMsg:   "MinField must be at least 5 characters",
		},
		{
			name: "max field error",
			input: TestStructTags{
				RequiredField: "test",
				EmailField:    "test@example.com",
				MinField:      "12345",
				MaxField:      "12345678901",
			},
			expectedField: "MaxField",
			expectedMsg:   "MaxField must be at most 10 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.input)
			assert.Error(t, err)

			var validationErr ValidationErrors
			require.True(t, errors.As(err, &validationErr))

			found := false
			for _, e := range validationErr {
				if e.Field == tt.expectedField {
					assert.Equal(t, tt.expectedMsg, e.Message)
					found = true
					break
				}
			}
			assert.True(t, found, "Expected field %s not found in validation errors", tt.expectedField)
		})
	}
}

// TestParseAndValidate tests the ParseAndValidate function
func TestParseAndValidate(t *testing.T) {
	type LoginRequest struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}

	tests := []struct {
		name        string
		body        string
		wantErr     bool
		expectedErr error
	}{
		{
			name:    "valid request",
			body:    `{"email": "user@example.com", "password": "password123"}`,
			wantErr: false,
		},
		{
			name:    "invalid email",
			body:    `{"email": "invalid-email", "password": "password123"}`,
			wantErr: true,
		},
		{
			name:    "missing password",
			body:    `{"email": "user@example.com"}`,
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    `{}`,
			wantErr: true,
		},
		{
			name:    "malformed JSON",
			body:    `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock Fiber context
			app := fiber.New(fiber.Config{ErrorHandler: http_error.NewHandler(nil)})
			c := app.AcquireCtx(&fasthttp.RequestCtx{})
			defer app.ReleaseCtx(c)

			c.Request().Header.SetMethod("POST")
			c.Request().Header.SetContentType("application/json")
			c.Request().SetBodyString(tt.body)

			var req LoginRequest
			err := ParseAndValidate(c, &req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "user@example.com", req.Email)
				assert.Equal(t, "password123", req.Password)
			}
		})
	}
}

// TestInvalid tests that bind/validation failures become 400 http_errors.
func TestInvalid(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "simple error", err: errors.New("test error")},
		{name: "validation error", err: ValidationErrors{{Field: "email", Message: "invalid email"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e *http_error.Error
			require.ErrorAs(t, Invalid(tt.err), &e)
			assert.Equal(t, fiber.StatusBadRequest, e.Status)
			assert.NotNil(t, e.Details, "field details must ride along")
		})
	}
}

// TestGetQueryPaginationDefaults is the regression test for `GET /users`
// returning 400 when ?limit/?page are omitted.
func TestGetQueryPaginationDefaults(t *testing.T) {
	type query struct {
		types.GetManyRequest
		Role string `query:"role" validate:"omitempty,oneof=ADMIN USER"`
	}

	app := fiber.New(fiber.Config{ErrorHandler: http_error.NewHandler(nil)})
	var got query
	app.Get("/users", func(c fiber.Ctx) error {
		got = query{}
		if err := GetQuery(c, &got); err != nil {
			return Invalid(err)
		}
		return c.SendString("ok")
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/users", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "no query params must not be a 400")
	assert.Equal(t, 10, got.Limit)
	assert.Equal(t, 1, got.Page)

	// Out-of-range values are still rejected.
	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/users?limit=101", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// TestGetParamValidatesUUID covers the "invalid path id becomes a 500" bug.
func TestGetParamValidatesUUID(t *testing.T) {
	type params struct {
		ID string `param:"id" validate:"required,uuid"`
	}

	app := fiber.New(fiber.Config{ErrorHandler: http_error.NewHandler(nil)})
	app.Get("/users/:id", func(c fiber.Ctx) error {
		var p params
		if err := GetParam(c, &p); err != nil {
			return Invalid(err)
		}
		return c.SendString(p.ID)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/users/not-a-uuid", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/users/1b4e28ba-2fa1-11d2-883f-0016d3cca427", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}


// TestValidationErrors_Error tests the Error method with various scenarios
func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name     string
		errors   ValidationErrors
		expected string
	}{
		{
			name: "single error",
			errors: ValidationErrors{
				{Field: "email", Message: "must be a valid email address"},
			},
			expected: "must be a valid email address",
		},
		{
			name: "multiple errors",
			errors: ValidationErrors{
				{Field: "email", Message: "must be a valid email address"},
				{Field: "password", Message: "is required"},
			},
			expected: "must be a valid email address; is required",
		},
		{
			name:     "no errors",
			errors:   ValidationErrors{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.errors.Error())
		})
	}
}
