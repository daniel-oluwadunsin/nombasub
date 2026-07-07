package requests

type InitiatePortalSessionRequest struct {
	TenantID   string `json:"tenantId" binding:"required"`
	CustomerID string `json:"customerId" binding:"required"`
}

type VerifyPortalSessionRequest struct {
	TenantID   string `json:"tenantId" binding:"required"`
	CustomerID string `json:"customerId" binding:"required"`
	Code       string `json:"code" binding:"required,len=6"`
}

type UpdatePortalProfileRequest struct {
	Name        *string `json:"name"`
	PhoneNumber *string `json:"phoneNumber"`
}

type UpdatePortalSubscriptionPaymentMethodRequest struct {
	PaymentSourceID string `json:"paymentSourceId" binding:"required"`
}
