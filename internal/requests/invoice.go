package requests

type GetInvoiceQuery struct {
	PaginationQuery
	Status     *string `form:"status"`
	CustomerID *string `form:"-"`
}

type GenerateInvoiceCheckoutLinkRequest struct {
	SendEmail bool `json:"sendEmail"`
}
