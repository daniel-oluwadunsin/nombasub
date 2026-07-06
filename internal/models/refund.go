package models

import "time"

type Refund struct {
	BaseModel
	TenantID           string                 `gorm:"column:tenant_id;not null" json:"tenantId"`
	PaymentID          *string                `gorm:"column:payment_id;" json:"paymentId"`
	InvoiceID          *string                `gorm:"column:invoice_id;" json:"invoiceId"`
	NombaTransactionId *string                `gorm:"column:nomba_transaction_id;" json:"nombaTransactionId"`
	Amount             float64                `gorm:"column:amount;not null" json:"amount"`
	Currency           string                 `gorm:"column:currency;not null" json:"currency"`
	Reason             *string                `gorm:"column:reason;" json:"reason"`
	InitiatedAt        time.Time              `gorm:"column:initiated_at;type:timestamp" json:"initiatedAt"`
	ETAFrom            time.Time              `gorm:"column:eta_from;type:timestamp" json:"etaFrom"`
	ETATo              time.Time              `gorm:"column:eta_to;type:timestamp" json:"etaTo"`
	Metadata           map[string]interface{} `gorm:"column:metadata;type:jsonb;serializer:json" json:"metadata"`
	Card               *CardPaymentSource     `gorm:"embedded;embeddedPrefix:card_" json:"card,omitempty"`
	Bank               *BankPaymentSource     `gorm:"embedded;embeddedPrefix:bank_" json:"bank,omitempty"`

	Invoice *Invoice       `gorm:"foreignKey:InvoiceID;references:ID" json:"invoice,omitempty"`
	Payment *PaymentIntent `gorm:"foreignKey:PaymentID;references:ID" json:"payment,omitempty"`
}

func (r *Refund) TableName() string {
	return TableNameRefund
}
