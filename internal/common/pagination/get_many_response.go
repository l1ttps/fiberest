package pagination

import (
	"fiberest/internal/common/types"
	"math"
)

func GetManyResponse[T any](query types.GetManyRequest, items []T, total int64) types.GetManyResponse[T] {
	// Calculate total pages
	totalPages := int(math.Ceil(float64(total) / float64(query.Limit)))
	return types.GetManyResponse[T]{
		Total:       total,
		Data:        items,
		HasNextPage: query.Page < totalPages,
		Limit:       query.Limit,
		PageCount:   totalPages,
		Page:        query.Page,
	}
}
