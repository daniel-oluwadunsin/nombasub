package responses

import (
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/models"
)

type RefundsResponse struct {
	Data []RefundResponse `json:"data"`
	Meta PaginationMeta   `json:"meta"`
}

type RefundResponse struct {
	ID                 string                    `json:"id"`
	PaymentID          *string                   `json:"paymentId"`
	InvoiceID          *string                   `json:"invoiceId"`
	NombaTransactionID *string                   `json:"nombaTransactionId"`
	Amount             float64                   `json:"amount"`
	Currency           string                    `json:"currency"`
	Reason             *string                   `json:"reason"`
	InitiatedAt        time.Time                 `json:"initiatedAt"`
	ETAFrom            time.Time                 `json:"etaFrom"`
	ETATo              time.Time                 `json:"etaTo"`
	Metadata           map[string]interface{}    `json:"metadata"`
	Card               *models.CardPaymentSource `json:"card,omitempty"`
	Bank               *models.BankPaymentSource `json:"bank,omitempty"`
	CreatedAt          time.Time                 `json:"createdAt"`
	UpdatedAt          time.Time                 `json:"updatedAt"`
	Invoice            *RefundInvoiceResponse    `json:"invoice"`
}

type RefundInvoiceResponse struct {
	ID              string     `json:"id"`
	Code            string     `json:"code"`
	Status          string     `json:"status"`
	AmountDue       int64      `json:"amountDue"`
	AmountPaid      int64      `json:"amountPaid"`
	AmountRemaining int64      `json:"amountRemaining"`
	Currency        string     `json:"currency"`
	PaidAt          *time.Time `json:"paidAt"`
	RefundedAt      *time.Time `json:"refundedAt"`
	CreatedAt       time.Time  `json:"createdAt"`
}
