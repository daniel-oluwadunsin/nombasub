package responses

import (
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/models"
)

type InitializeDirectDebitResponse struct {
	MandateID           string `json:"mandateId"`
	MerchantReference   string `json:"merchantReference"`
	CustomerPhoneNumber string `json:"customerPhoneNumber"`
	Description         string `json:"description"`
}

type PaymentIntentListItem struct {
	ID                     string                     `json:"id"`
	Code                   string                     `json:"code"`
	Status                 models.PaymentIntentStatus `json:"status"`
	Amount                 int64                      `json:"amount"`
	Currency               string                     `json:"currency"`
	Reference              string                     `json:"reference"`
	FailureReason          *string                    `json:"failureReason"`
	CustomerID             string                     `json:"customerId"`
	SubscriptionID         string                     `json:"subscriptionId"`
	InvoiceID              *string                    `json:"invoiceId"`
	PaymentSourceType      *models.PaymentSourceType  `json:"paymentSourceType"`
	ProviderTransactionID  *string                    `json:"providerTransactionId"`
	ProviderTransactionRef *string                    `json:"providerTransactionReference"`
	AttemptedAt            *time.Time                 `json:"attemptedAt"`
	CompletedAt            *time.Time                 `json:"completedAt"`
	FailedAt               *time.Time                 `json:"failedAt"`
	CreatedAt              time.Time                  `json:"createdAt"`
}

type PaymentIntentsResponse struct {
	Data []PaymentIntentListItem `json:"data"`
	Meta PaginationMeta          `json:"meta"`
}
