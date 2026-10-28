package auth

import (
	"errors"
	"testing"
	"time"

	"fiberest/internal/configs"
	"fiberest/internal/models"
	"fiberest/pkg/http_error"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func statusOf(t *testing.T, err error) int {
	t.Helper()
	var e *http_error.Error
	require.True(t, errors.As(err, &e), "expected *http_error.Error, got %T", err)
	return e.Status
}

// TestErrorConstants pins the messages controllers map to HTTP statuses.
func TestErrorConstants(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{name: "ErrUserNotFound", err: ErrUserNotFound, expected: "user not found"},
		{name: "ErrAdminExists", err: ErrAdminExists, expected: "admin user already exists"},
		{name: "ErrInvalidCredential", err: ErrInvalidCredential, expected: "invalid email or password"},
		{name: "ErrSessionNotFound", err: ErrSessionNotFound, expected: "session not found"},
		{name: "ErrWrongPassword", err: ErrWrongPassword, expected: "current password is incorrect"},
		{name: "ErrAccountBanned", err: ErrAccountBanned, expected: "account is banned"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualError(t, tt.err, tt.expected)
		})
	}
}

// TestSessionDurationUsesConfig guards against the config field being ignored.
func TestSessionDurationUsesConfig(t *testing.T) {
	cfg := &configs.Config{}
	cfg.Auth.SessionDuration = 48 * time.Hour
	assert.Equal(t, 48*time.Hour, (&service{cfg: cfg}).sessionDuration())

	// Falls back to a sane default when config is absent.
	assert.Equal(t, 7*24*time.Hour, (&service{}).sessionDuration())
}

// TestUpdateExpiresSession_SkipsNonRememberMe: plain sessions are never extended.
func TestUpdateExpiresSession_SkipsNonRememberMe(t *testing.T) {
	session := &models.Session{
		ExpiresAt:  time.Now().Add(time.Hour),
		RememberMe: false,
		BaseModel:  models.BaseModel{UpdatedAt: time.Now().Add(-48 * time.Hour)},
	}
	before := session.ExpiresAt

	require.NoError(t, (&service{}).UpdateExpiresSession(t.Context(), session))
	assert.Equal(t, before, session.ExpiresAt)
}

// TestMapError maps service sentinels to HTTP statuses.
func TestMapError(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{err: ErrInvalidCredential, status: 401},
		{err: ErrWrongPassword, status: 400},
		{err: ErrAdminExists, status: 400},
		{err: ErrUserNotFound, status: 404},
		{err: ErrSessionNotFound, status: 401},
	}
	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			assert.Equal(t, tt.status, statusOf(t, mapError(tt.err)))
		})
	}
}
