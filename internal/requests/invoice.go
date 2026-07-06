package requests

type GetInvoiceQuery struct {
	PaginationQuery
	Status *string `form:"status"`
}

type GenerateInvoiceCheckoutLinkRequest struct {
	SendEmail bool `json:"sendEmail"`
}
