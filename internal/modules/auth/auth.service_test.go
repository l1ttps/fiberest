package auth

import (
	"testing"
	"time"

	"fiberest/internal/common/constants"
	"fiberest/internal/models"

	"github.com/stretchr/testify/assert"
)

// TestErrorConstants tests that error constants are properly defined
func TestErrorConstants(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrUserNotFound",
			err:      ErrUserNotFound,
			expected: "user not found",
		},
		{
			name:     "ErrAdminExists",
			err:      ErrAdminExists,
			expected: "admin user already exists",
		},
		{
			name:     "ErrInvalidCredential",
			err:      ErrInvalidCredential,
			expected: "invalid email or password",
		},
		{
			name:     "ErrSessionNotFound",
			err:      ErrSessionNotFound,
			expected: "session not found",
		},
		{
			name:     "ErrWrongPassword",
			err:      ErrWrongPassword,
			expected: "current password is incorrect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualError(t, tt.err, tt.expected)
		})
	}
}

// TestConstants tests session duration constant
func TestConstants(t *testing.T) {
	t.Run("SessionDuration is 7 days", func(t *testing.T) {
		assert.Equal(t, 7*24*time.Hour, constants.SessionDuration)
	})
}

// TestSessionIsValid tests the IsValid method on Session model
func TestSessionIsValid(t *testing.T) {
	tests := []struct {
		name     string
		session  models.Session
		expected bool
	}{
		{
			name: "valid session",
			session: models.Session{
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
			expected: true,
		},
		{
			name: "expired session",
			session: models.Session{
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "session expiring now",
			session: models.Session{
				ExpiresAt: time.Now(),
			},
			expected: false,
		},
		{
			name: "session with future expiration",
			session: models.Session{
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.session.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}
