package responses

import (
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/models"
)

type CustomerDetailResponse struct {
	Customer       CustomerProfileResponse        `json:"customer"`
	Summary        CustomerSummaryResponse        `json:"summary"`
	Subscriptions  []CustomerSubscriptionResponse `json:"subscriptions"`
	PaymentSources []CustomerPaymentSourceDetail  `json:"paymentSources"`
}

type CustomerProfileResponse struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        *string   `json:"name"`
	Email       string    `json:"email"`
	PhoneNumber *string   `json:"phoneNumber"`
	ExternalRef *string   `json:"externalRef"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CustomerSummaryResponse struct {
	LifetimeValue         int64      `json:"lifetimeValue"`
	Currency              string     `json:"currency"`
	ActiveSubscriptions   int64      `json:"activeSubscriptions"`
	EarliestInvoicePaidAt *time.Time `json:"earliestInvoicePaidAt"`
	DateJoined            time.Time  `json:"dateJoined"`
	TotalPaidInvoices     int64      `json:"totalPaidInvoices"`
	TotalSubscriptions    int        `json:"totalSubscriptions"`
	TotalPaymentSources   int        `json:"totalPaymentSources"`
}

type CustomerSubscriptionResponse struct {
	ID                string                       `json:"id"`
	Code              string                       `json:"code"`
	Status            string                       `json:"status"`
	Amount            int64                        `json:"amount"`
	Currency          string                       `json:"currency"`
	Interval          string                       `json:"interval"`
	IntervalCount     *int                         `json:"intervalCount"`
	StartedAt         *time.Time                   `json:"startedAt"`
	NextChargeAt      *time.Time                   `json:"nextChargeAt"`
	LastChargedAt     *time.Time                   `json:"lastChargedAt"`
	TotalInvoicesPaid int64                        `json:"totalInvoicesPaid"`
	Plan              CustomerPlanResponse         `json:"plan"`
	PaymentSource     *CustomerPaymentSourceDetail `json:"paymentSource"`
}

type CustomerPlanResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type CustomerPaymentSourceDetail struct {
	ID                 string                    `json:"id"`
	Type               string                    `json:"type"`
	Status             string                    `json:"status"`
	CreatedAt          time.Time                 `json:"createdAt"`
	Card               *models.CardPaymentSource `json:"card,omitempty"`
	Bank               *models.BankPaymentSource `json:"bank,omitempty"`
	ExpiresSoon        bool                      `json:"expiresSoon"`
	ExpirationMailSent bool                      `json:"expirationMailSent"`
}
