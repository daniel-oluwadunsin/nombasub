package requests

type SignUpTenantRequest struct {
	BusinessName string `json:"businessName" binding:"required"`
	AccountID    string `json:"accountId" binding:"required"`
	Password     string `json:"password" binding:"required"`
}

type LoginTenantRequest struct {
	AccountID string `json:"accountId" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

type SetWebhookUrlRequest struct {
	WebhookUrl string `json:"webhookUrl" binding:"required,url"`
}

type UpdateTenantSettingsRequest struct {
	BusinessName *string `json:"businessName" binding:"omitempty"`
	WebhookUrl   *string `json:"webhookUrl" binding:"omitempty,url"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}
