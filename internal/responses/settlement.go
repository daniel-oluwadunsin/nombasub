package responses

import "time"

type SettlementPayoutsResponse struct {
	Data    []SettlementPayoutListItem `json:"data"`
	Meta    PaginationMeta             `json:"meta"`
	Metrics SettlementPayoutMetrics    `json:"metrics"`
}

type SettlementPayoutMetrics struct {
	TotalPaidOut                      float64    `json:"totalPaidOut"`
	PendingSettlementAmount           float64    `json:"pendingSettlementAmount"`
	PendingSettlementTransactionCount int64      `json:"pendingSettlementTransactionCount"`
	LastPaidOutDate                   *time.Time `json:"lastPaidOutDate"`
	Currency                          string     `json:"currency"`
}

type SettlementPayoutListItem struct {
	ID                 string     `json:"id"`
	Amount             float64    `json:"amount"`
	Currency           string     `json:"currency"`
	Reference          string     `json:"reference"`
	Status             string     `json:"status"`
	NombaTransactionID *string    `json:"nombaTransactionId"`
	FailureReason      *string    `json:"failureReason"`
	ProcessedAt        *time.Time `json:"processedAt"`
	CreatedAt          time.Time  `json:"createdAt"`
	SettlementCount    int        `json:"settlementCount"`
	RecipientAccountID *string    `json:"recipientAccountId"`
	IsVirtual          bool       `json:"isVirtual"`
}

type SettlementPayoutDetail struct {
	SettlementPayoutListItem
	UpdatedAt   time.Time            `json:"updatedAt"`
	Settlements []SettlementResponse `json:"settlements"`
}

type SettlementResponse struct {
	ID                 string    `json:"id"`
	Amount             float64   `json:"amount"`
	Currency           string    `json:"currency"`
	SettlementDate     time.Time `json:"settlementDate"`
	Status             string    `json:"status"`
	Reference          string    `json:"reference"`
	FailureReason      *string   `json:"failureReason"`
	Purpose            string    `json:"purpose"`
	SubscriptionID     *string   `json:"subscriptionId"`
	InvoiceID          *string   `json:"invoiceId"`
	SettlementPayoutID *string   `json:"settlementPayoutId"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}
