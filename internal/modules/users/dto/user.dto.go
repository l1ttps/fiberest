package dto

import "fiberest/internal/common/types"

// UserResponse represents a user in the response (without sensitive data like password)
type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
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
