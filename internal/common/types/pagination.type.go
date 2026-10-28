package types

import "math"

// GetManyRequest is the shared pagination/search query for list endpoints.
type GetManyRequest struct {
	Limit  int    `query:"limit" validate:"omitempty,min=1,max=100" example:"10"`
	Page   int    `query:"page" validate:"omitempty,min=1" example:"1"`
	Search string `query:"search" validate:"omitempty,max=100"`
}

// SetDefaults fills omitted pagination fields. validators.GetQuery calls it
// before validation, so ?limit and ?page are optional.
func (r *GetManyRequest) SetDefaults() {
	if r.Limit == 0 {
		r.Limit = 10
	}
	if r.Page == 0 {
		r.Page = 1
	}
}

// PageMeta is embedded in every paginated response; the single place that
// computes page counts.
type PageMeta struct {
	Limit       int   `json:"limit"`
	Page        int   `json:"page"`
	PageCount   int   `json:"pageCount"`
	HasNextPage bool  `json:"hasNextPage"`
	Total       int64 `json:"total"`
}

// NewPageMeta computes pagination metadata for limit/page/total.
func NewPageMeta(limit, page int, total int64) PageMeta {
	var pageCount int
	if limit > 0 {
		pageCount = int(math.Ceil(float64(total) / float64(limit)))
	}
	return PageMeta{
		Limit:       limit,
		Page:        page,
		PageCount:   pageCount,
		HasNextPage: page < pageCount,
		Total:       total,
	}
}
