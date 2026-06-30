package requests

type AuthTenantRequest struct {
	AccountID    string `json:"accountId" binding:"required"`
	ClientID     string `json:"clientId" binding:"required"`
	ClientSecret string `json:"clientSecret" binding:"required"`
}

type SetWebhookUrlRequest struct {
	WebhookUrl string `json:"webhookUrl" binding:"required,url"`
}
