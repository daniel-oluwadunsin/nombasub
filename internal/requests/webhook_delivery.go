package requests

type WebhookDeliveriesQuery struct {
	PaginationQuery
	Status    *string `form:"status"`
	EventType *string `form:"eventType"`
	From      *string `form:"from"`
	To        *string `form:"to"`
}
