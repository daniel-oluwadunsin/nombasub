package responses

import (
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/models"
)

type InvoicePlanResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type InvoiceSubscriptionResponse struct {
	ID   string `json:"id"`
	Code string `json:"code"`
}

type InvoicePaymentIntentResponse struct {
	ID                     string                       `json:"id"`
	Code                   string                       `json:"code"`
	Status                 models.PaymentIntentStatus   `json:"status"`
	ProviderTransactionID  *string                      `json:"providerTransactionId"`
	ProviderTransactionRef *string                      `json:"providerTransactionReference"`
	PaymentSource          *CustomerPaymentSourceDetail `json:"paymentSource"`
}

type InvoiceResponse struct {
	ID                 string                        `json:"id"`
	Code               string                        `json:"code"`
	Status             models.InvoiceStatus          `json:"status"`
	AmountDue          int64                         `json:"amountDue"`
	AmountPaid         int64                         `json:"amountPaid"`
	AmountRemaining    int64                         `json:"amountRemaining"`
	Currency           string                        `json:"currency"`
	BillingPeriodStart *time.Time                    `json:"billingPeriodStart"`
	BillingPeriodEnd   *time.Time                    `json:"billingPeriodEnd"`
	DueAt              *time.Time                    `json:"dueAt"`
	PaidAt             *time.Time                    `json:"paidAt"`
	FailedAt           *time.Time                    `json:"failedAt"`
	RefundedAt         *time.Time                    `json:"refundedAt"`
	CheckoutLink       *string                       `json:"checkoutLink"`
	FailureReason      *string                       `json:"failureReason"`
	CanBeRefunded      bool                          `json:"canBeRefunded"`
	Plan               InvoicePlanResponse           `json:"plan"`
	Customer           CustomerProfileResponse       `json:"customer"`
	Subscription       InvoiceSubscriptionResponse   `json:"subscription"`
	PaymentIntent      *InvoicePaymentIntentResponse `json:"paymentIntent"`
	Refund             *RefundResponse               `json:"refund"`
	CreatedAt          time.Time                     `json:"createdAt"`
	UpdatedAt          time.Time                     `json:"updatedAt"`
}

type GenerateInvoiceCheckoutLinkResponse struct {
	CheckoutLink string `json:"checkoutLink"`
	Sent         bool   `json:"sent"`
}
