package dto

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
}

// LoginResponse represents the response after successful login
type LoginResponse struct {
	Message      string `json:"message" example:"Login successful"`
	SessionToken string `json:"-"` // Not exposed in JSON, used for setting cookie
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
