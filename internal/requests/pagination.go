package requests

type PaginationQuery struct {
	Page   *int    `json:"page" form:"page"`
	Limit  *int    `json:"limit" form:"limit"`
	Search *string `json:"search" form:"search"`
}
