package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestSessionIsValid tests the IsValid method on Session model
func TestSessionIsValid(t *testing.T) {
	tests := []struct {
		name     string
		session  Session
		expected bool
	}{
		{
			name: "valid session with future expiration",
			session: Session{
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
			expected: true,
		},
		{
			name: "valid session with long expiration",
			session: Session{
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			},
			expected: true,
		},
		{
			name: "expired session",
			session: Session{
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "session expired long ago",
			session: Session{
				ExpiresAt: time.Now().Add(-7 * 24 * time.Hour),
			},
			expected: false,
		},
		{
			name: "session expiring exactly now",
			session: Session{
				ExpiresAt: time.Now(),
			},
			expected: false,
		},
		{
			name: "session expiring in 1 second",
			session: Session{
				ExpiresAt: time.Now().Add(1 * time.Second),
			},
			expected: true,
		},
		{
			name: "session expiring 1 second ago",
			session: Session{
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

// TestUserRoleConstants tests the UserRole constants
func TestUserRoleConstants(t *testing.T) {
	tests := []struct {
		name     string
		role     UserRole
		expected string
	}{
		{
			name:     "admin role",
			role:     RoleAdmin,
			expected: "ADMIN",
		},
		{
			name:     "user role",
			role:     RoleUser,
			expected: "USER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.role))
		})
	}
}

// TestAccountTypeConstants tests the AccountType constants
func TestAccountTypeConstants(t *testing.T) {
	tests := []struct {
		name        string
		accountType AccountType
		expected    string
	}{
		{
			name:        "email account type",
			accountType: AccountTypeEmail,
			expected:    "EMAIL",
		},
		{
			name:        "google account type",
			accountType: AccountTypeGoogle,
			expected:    "GOOGLE",
		},
		{
			name:        "github account type",
			accountType: AccountTypeGitHub,
			expected:    "GITHUB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.accountType))
		})
	}
}

// TestUserTableName tests the TableName method for User
func TestUserTableName(t *testing.T) {
	user := User{}
	assert.Equal(t, "users", user.TableName())
}

// TestAccountTableName tests the TableName method for Account
func TestAccountTableName(t *testing.T) {
	account := Account{}
	assert.Equal(t, "accounts", account.TableName())
}

// TestSessionTableName tests the TableName method for Session
func TestSessionTableName(t *testing.T) {
	session := Session{}
	assert.Equal(t, "sessions", session.TableName())
}

// TestSessionFields tests that Session struct has all expected fields
func TestSessionFields(t *testing.T) {
	session := Session{
		SessionToken: "test-token",
		UserID:       uuid.New(),
		User:         User{},
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		IPAddress:    "127.0.0.1",
		UserAgent:    "TestAgent",
		RememberMe:   true,
	}

	assert.NotEmpty(t, session.SessionToken)
	assert.NotEmpty(t, session.UserID)
	assert.NotEmpty(t, session.ExpiresAt)
	assert.Equal(t, "127.0.0.1", session.IPAddress)
	assert.Equal(t, "TestAgent", session.UserAgent)
	assert.True(t, session.RememberMe)
}

// TestUserFields tests that User struct has all expected fields
func TestUserFields(t *testing.T) {
	user := User{
		Email: "test@example.com",
		Name:  "Test User",
		Role:  RoleUser,
	}

	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, RoleUser, user.Role)
}

// TestAccountFields tests that Account struct has all expected fields
func TestAccountFields(t *testing.T) {
	userUUID := uuid.New()
	account := Account{
		UserID:    userUUID,
		User:      User{},
		Type:      AccountTypeEmail,
		Password:  "hashed-password",
		IsPrimary: true,
	}

	assert.Equal(t, userUUID, account.UserID)
	assert.Equal(t, AccountTypeEmail, account.Type)
	assert.Equal(t, "hashed-password", account.Password)
	assert.True(t, account.IsPrimary)
}

// TestBaseModelFields tests that BaseModel is embedded correctly
func TestBaseModelFields(t *testing.T) {
	user := User{}
	assert.NotNil(t, user.BaseModel)

	account := Account{}
	assert.NotNil(t, account.BaseModel)

	session := Session{}
	assert.NotNil(t, session.BaseModel)
}
