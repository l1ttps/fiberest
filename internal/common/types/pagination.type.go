package types

// GetManyRequest is a common pagination request structure
// that can be reused for any get many API.
type GetManyRequest struct {
	Limit  int    `query:"limit" validate:"min=1,max=100" example:"10"`
	Page   int    `query:"page" validate:"min=1" example:"1"`
	Search string `query:"search" validate:"omitempty,max=100"`
}

// GetManyResponse is a generic pagination response structure
// that can be reused for any resource type.
type GetManyResponse[T any] struct {
	Data        []T   `json:"data"`
	Limit       int   `json:"limit"`
	Page        int   `json:"page"`
	HasNextPage bool  `json:"hasNextPage"`
	PageCount   int   `json:"pageCount"`
	Total       int64 `json:"total"`
}
