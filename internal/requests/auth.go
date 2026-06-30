package requests

type SignUpTenantRequest struct {
	BusinessName string `json:"businessName" binding:"required"`
	AccountID    string `json:"accountId" binding:"required"`
}

type LoginTenantRequest struct {
	AccountID string `json:"accountId" binding:"required"`
}

type SetWebhookUrlRequest struct {
	WebhookUrl string `json:"webhookUrl" binding:"required,url"`
}
