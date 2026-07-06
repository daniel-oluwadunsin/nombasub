package responses

import (
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/models"
)

type SubscriptionPlanResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type SubscriptionCustomerResponse struct {
	ID    string  `json:"id"`
	Code  string  `json:"code"`
	Name  *string `json:"name"`
	Email string  `json:"email"`
}

type SubscriptionInvoiceResponse struct {
	ID           string               `json:"id"`
	Code         string               `json:"code"`
	Status       models.InvoiceStatus `json:"status"`
	CheckoutLink *string              `json:"checkoutLink"`
}

type SubscriptionResponse struct {
	ID                       string                       `json:"id"`
	Code                     string                       `json:"code"`
	Status                   models.SubscriptionStatus    `json:"status"`
	Amount                   int64                        `json:"amount"`
	Currency                 string                       `json:"currency"`
	Interval                 models.PlanInterval          `json:"interval"`
	IntervalCount            *int                         `json:"intervalCount"`
	StartedAt                *time.Time                   `json:"startedAt"`
	CancelledAt              *time.Time                   `json:"cancelledAt"`
	PausedAt                 *time.Time                   `json:"pausedAt"`
	CurrentBillingCycleStart *time.Time                   `json:"currentBillingCycleStart"`
	CurrentBillingCycleEnd   *time.Time                   `json:"currentBillingCycleEnd"`
	LatestInvoiceID          *string                      `json:"latestInvoiceId"`
	AllowRetries             bool                         `json:"allowRetries"`
	CanGenerateCheckoutLink  bool                         `json:"canGenerateCheckoutLink"`
	Payments                 int64                        `json:"payments"`
	LifetimeValue            int64                        `json:"lifetimeValue"`
	SubscribedForDays        int                          `json:"subscribedForDays"`
	Plan                     SubscriptionPlanResponse     `json:"plan"`
	Customer                 SubscriptionCustomerResponse `json:"customer"`
	PaymentSource            *CustomerPaymentSourceDetail `json:"paymentSource"`
	LatestInvoice            *SubscriptionInvoiceResponse `json:"latestInvoice"`
	CreatedAt                time.Time                    `json:"createdAt"`
	UpdatedAt                time.Time                    `json:"updatedAt"`
}

type GenerateCheckoutLinkResponse struct {
	CheckoutLink string `json:"checkoutLink"`
}
