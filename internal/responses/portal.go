package responses

import "time"

type PortalSessionInitiatedResponse struct {
	SessionID     string     `json:"sessionId"`
	CustomerEmail string     `json:"customerEmail"`
	CodeExpiresAt *time.Time `json:"codeExpiresAt"`
}

type PortalSessionResponse struct {
	AccessToken string                  `json:"accessToken,omitempty"`
	ExpiresAt   *time.Time              `json:"expiresAt,omitempty"`
	Tenant      PortalTenantResponse    `json:"tenant"`
	Customer    CustomerProfileResponse `json:"customer"`
}

type PortalCardUpdateResponse struct {
	CheckoutLink string `json:"checkoutLink"`
}

type PortalTenantResponse struct {
	ID           string  `json:"id"`
	BusinessName *string `json:"businessName"`
	AccountID    string  `json:"accountId"`
}

type PortalAnalyticsResponse struct {
	Currency               string                   `json:"currency"`
	TotalSubscriptions     int64                    `json:"totalSubscriptions"`
	ActiveSubscriptions    int64                    `json:"activeSubscriptions"`
	TotalCards             int64                    `json:"totalCards"`
	TotalDirectDebits      int64                    `json:"totalDirectDebits"`
	TotalSpent             int64                    `json:"totalSpent"`
	AmountSpentTrend       []PortalAmountTrendPoint `json:"amountSpentTrend"`
	SubscriptionStatusData []PortalBreakdownItem    `json:"subscriptionStatusData"`
	PaymentSourceData      []PortalBreakdownItem    `json:"paymentSourceData"`
}

type PortalAmountTrendPoint struct {
	Date   string `json:"date"`
	Amount int64  `json:"amount"`
}

type PortalBreakdownItem struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}
