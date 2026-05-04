package users

import (
	"testing"
	"time"

	"fiberest/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestUserRoleConstants tests the UserRole constants
func TestUserRoleConstants(t *testing.T) {
	tests := []struct {
		name     string
		role     models.UserRole
		expected string
	}{
		{
			name:     "admin role",
			role:     models.RoleAdmin,
			expected: "ADMIN",
		},
		{
			name:     "user role",
			role:     models.RoleUser,
			expected: "USER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.role))
		})
	}
}

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
			name:     "ErrUserAlreadyExists",
			err:      ErrUserAlreadyExists,
			expected: "user already exists",
		},
		{
			name:     "ErrAdminExists",
			err:      ErrAdminExists,
			expected: "admin user already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualError(t, tt.err, tt.expected)
		})
	}
}

// TestUserFields tests that User struct has all expected fields
func TestUserFields(t *testing.T) {
	user := models.User{
		Email: "test@example.com",
		Name:  "Test User",
		Role:  models.RoleUser,
	}

	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, models.RoleUser, user.Role)
}

// TestUserTableName tests the TableName method for User
func TestUserTableName(t *testing.T) {
	user := models.User{}
	assert.Equal(t, "users", user.TableName())
}

// TestAccountTableName tests the TableName method for Account
func TestAccountTableName(t *testing.T) {
	account := models.Account{}
	assert.Equal(t, "accounts", account.TableName())
}

// TestSessionTableName tests the TableName method for Session
func TestSessionTableName(t *testing.T) {
	session := models.Session{}
	assert.Equal(t, "sessions", session.TableName())
}

// TestAccountTypeConstants tests the AccountType constants
func TestAccountTypeConstants(t *testing.T) {
	tests := []struct {
		name        string
		accountType models.AccountType
		expected    string
	}{
		{
			name:        "email account type",
			accountType: models.AccountTypeEmail,
			expected:    "EMAIL",
		},
		{
			name:        "google account type",
			accountType: models.AccountTypeGoogle,
			expected:    "GOOGLE",
		},
		{
			name:        "github account type",
			accountType: models.AccountTypeGitHub,
			expected:    "GITHUB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.accountType))
		})
	}
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
		{
			name: "session expiring in 1 second",
			session: models.Session{
				ExpiresAt: time.Now().Add(1 * time.Second),
			},
			expected: true,
		},
		{
			name: "session expired 1 second ago",
			session: models.Session{
				ExpiresAt: time.Now().Add(-1 * time.Second),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.session.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBaseModelFields tests that BaseModel is embedded correctly
func TestBaseModelFields(t *testing.T) {
	user := models.User{}
	assert.NotNil(t, user.BaseModel)

	account := models.Account{}
	assert.NotNil(t, account.BaseModel)

	session := models.Session{}
	assert.NotNil(t, session.BaseModel)
}

// TestSessionFields tests that Session struct has all expected fields
func TestSessionFields(t *testing.T) {
	userID := uuid.New()
	session := models.Session{
		SessionToken: "test-token",
		UserID:       userID,
		User:         models.User{},
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		IPAddress:    "127.0.0.1",
		UserAgent:    "TestAgent",
		RememberMe:   true,
	}

	assert.NotEmpty(t, session.SessionToken)
	assert.Equal(t, userID, session.UserID)
	assert.NotEmpty(t, session.ExpiresAt)
	assert.Equal(t, "127.0.0.1", session.IPAddress)
	assert.Equal(t, "TestAgent", session.UserAgent)
	assert.True(t, session.RememberMe)
}

// TestAccountFields tests that Account struct has all expected fields
func TestAccountFields(t *testing.T) {
	userUUID := uuid.New()
	account := models.Account{
		UserID:    userUUID,
		User:      models.User{},
		Type:      models.AccountTypeEmail,
		Password:  "hashed-password",
		IsPrimary: true,
	}

	assert.Equal(t, userUUID, account.UserID)
	assert.Equal(t, models.AccountTypeEmail, account.Type)
	assert.Equal(t, "hashed-password", account.Password)
	assert.True(t, account.IsPrimary)
}
