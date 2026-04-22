package dto

import (
	"time"

	"fiberest/internal/common/types"
)

// InitAdminRequest represents the request body for creating an admin user
type InitAdminRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// InitAdminResponse represents the response after creating an admin user
type InitAdminResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// LoginRequest represents the request body for user login
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	Remember bool   `json:"remember"` // Optional: if true, extends session duration
}

// LoginResponse represents the response after successful login
type LoginResponse struct {
	Message      string `json:"message" example:"Login successful"`
	SessionToken string `json:"-"`         // Not exposed in JSON, used for setting cookie
	ExpiresAt    string `json:"expiresAt"` // Session expiration time (ISO 8601 format)
}

// RefreshTokenRequest represents the request body for token refresh
// DEPRECATED: No longer used in session-based auth
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

// ChangePasswordRequest represents the request body for changing password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required"`
	NewPassword     string `json:"newPassword" validate:"required,min=8"`
}

// ChangePasswordResponse represents the response after changing password
type ChangePasswordResponse struct {
	Message string `json:"message" example:"Password changed successfully"`
}

// GetManySessionsRequest extends the common GetManyRequest
type GetManySessionsRequest struct {
	types.GetManyRequest
}

func (r *GetManySessionsRequest) SetDefaults() {
	if r.Limit == 0 {
		r.Limit = 10
	}
	if r.Page == 0 {
		r.Page = 1
	}
}

// GetManySessionsResponse represents the paginated sessions response
type GetManySessionsResponse struct {
	Data        []SessionResponse `json:"data"`
	Limit       int               `json:"limit"`
	Page        int               `json:"page"`
	HasNextPage bool              `json:"hasNextPage"`
	Total       int64             `json:"total"`
}

// SessionResponse represents session data in responses
type SessionResponse struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userId"`
	ExpiresAt  time.Time `json:"expiresAt"`
	IPAddress  string    `json:"ipAddress"`
	UserAgent  string    `json:"userAgent"`
	RememberMe bool      `json:"rememberMe"`
	CreatedAt  time.Time `json:"createdAt"`
}
