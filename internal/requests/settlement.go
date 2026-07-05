package requests

type SettlementPayoutsQuery struct {
	PaginationQuery
	Status *string `form:"status"`
	From   *string `form:"from"`
	To     *string `form:"to"`
}
