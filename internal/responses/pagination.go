package responses

type PaginatedResponse[T any] struct {
	Data            []T  `json:"data"`
	TotalCount      int  `json:"totalCount"`
	Page            *int `json:"page,omitempty"`
	Limit           *int `json:"limit,omitempty"`
	TotalPages      int  `json:"totalPages"`
	HasNextPage     bool `json:"hasNextPage"`
	HasPreviousPage bool `json:"hasPreviousPage"`
}

func NewPaginatedResponse[T any](page int, limit int, totalCount int, data []T) *PaginatedResponse[T] {
	totalPages := 0
	if limit > 0 {
		totalPages = (totalCount + limit - 1) / limit
	}

	hasNextPage := page < totalPages
	hasPreviousPage := page > 1

	return &PaginatedResponse[T]{
		Data:            data,
		TotalCount:      totalCount,
		Page:            &page,
		Limit:           &limit,
		TotalPages:      totalPages,
		HasNextPage:     hasNextPage,
		HasPreviousPage: hasPreviousPage,
	}
}
