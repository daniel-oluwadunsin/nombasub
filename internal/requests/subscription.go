package requests

type CreateSubscriptionRequest struct {
	CustomerEmailOrCode string  `json:"customerEmailOrCode" binding:"required"`
	PlanCode            string  `json:"planCode" binding:"required"`
	CardToken           *string `json:"cardToken"`
	MandateID           *string `json:"mandateId"`
}

type GetSubscriptionQuery struct {
	PaginationQuery
	Customer *string `form:"customer"`
	Plan     *string `form:"plan"`
}
