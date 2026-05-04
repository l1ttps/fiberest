package dto

import (
	"fiberest/internal/common/types"
	"time"
)

// UserResponse represents a user in the response (without sensitive data like password)
type UserResponse struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Name      string     `json:"name"`
	Role      string     `json:"role"`
	BanReason string     `json:"banReason"`
	BanUntil  *time.Time `json:"banUntil"`
}

// GetManyUsersRequest extends the common GetManyRequest for user-specific queries.
// It embeds types.GetManyRequest to inherit pagination fields (Limit, Page, Search).
type GetManyUsersRequest struct {
	types.GetManyRequest
	// Role is an optional filter for user role (ADMIN or USER)
	Role string `query:"role" validate:"omitempty,oneof=ADMIN USER"`
}

// GetManyUsersExample is a concrete type for Swagger documentation
type GetManyUsersExample struct {
	Data        []UserResponse `json:"data"`
	Limit       int            `json:"limit"`
	Page        int            `json:"page"`
	HasNextPage bool           `json:"hasNextPage"`
	Total       int64          `json:"total"`
}

// UpdateMyProfileRequest defines the payload for updating own profile
type UpdateMyProfileRequest struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
}

// UpdateUserRequest defines the payload for updating a user (admin only)
type UpdateUserRequest struct {
	Email string `json:"email" validate:"omitempty,email"`
	Name  string `json:"name" validate:"omitempty,min=1,max=255"`
	Role  string `json:"role" validate:"omitempty,oneof=ADMIN USER"`
}

// CreateUserRequest defines the payload for creating a new user (admin only)
type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
	Name     string `json:"name" validate:"required,min=1,max=255"`
	Role     string `json:"role" validate:"omitempty,oneof=ADMIN USER"`
}

// SetPasswordRequest defines the payload for setting a user's password
type SetPasswordRequest struct {
	Password string `json:"password" validate:"required,min=8,max=72"`
}

// BanUserRequest defines the payload for banning a user
type BanUserRequest struct {
	Reason string `json:"reason" validate:"required,min=1,max=500"`
	Until  string `json:"until" validate:"omitempty"`
}
