package models

import "time"

type SettlementStatus string

const (
	SettlementStatusPending   SettlementStatus = "pending"
	SettlementStatusCompleted SettlementStatus = "completed"
	SettlementStatusFailed    SettlementStatus = "failed"
)

type Settlement struct {
	BaseModel
	TenantID           string                 `gorm:"column:tenant_id;type:uuid;not null" json:"-"`
	Amount             float64                `gorm:"column:amount;type:numeric(10,2);not null" json:"amount"`
	Currency           string                 `gorm:"column:currency;type:varchar(3);not null" json:"currency"`
	SettlementTime     time.Time              `gorm:"column:settlement_date;type:timestamp;not null" json:"settlementDate"`
	Status             SettlementStatus       `gorm:"column:status;type:text;not null;default:'pending'" json:"status"`
	Reference          string                 `gorm:"column:reference;type:varchar(100);not null" json:"reference"`
	FailureReason      *string                `gorm:"column:failure_reason;type:text;" json:"failureReason"`
	Purpose            NombaInitiationPurpose `gorm:"column:purpose;type:text;not null" json:"purpose"`
	SubscriptionID     *string                `gorm:"column:subscription_id;type:uuid;" json:"subscriptionId"`
	InvoiceID          *string                `gorm:"column:invoice_id;type:uuid;" json:"invoiceId"`
	SettlementPayoutID *string                `gorm:"column:settlement_payout_id;type:uuid;" json:"settlementPayoutId"`
}

func (Settlement) TableName() string {
	return TableNameSettlement
}

type SettlementPayoutStatus string

const (
	SettlementPayoutStatusPending   SettlementPayoutStatus = "pending"
	SettlementPayoutStatusCompleted SettlementPayoutStatus = "completed"
	SettlementPayoutStatusFailed    SettlementPayoutStatus = "failed"
)

type SettlementPayout struct {
	BaseModel
	TenantID           string                 `gorm:"column:tenant_id;type:uuid;not null" json:"-"`
	Amount             float64                `gorm:"column:amount;type:numeric(10,2);not null" json:"amount"`
	Currency           string                 `gorm:"column:currency;type:varchar(3);not null" json:"currency"`
	Reference          string                 `gorm:"column:reference;type:varchar(100);not null" json:"reference"`
	Status             SettlementPayoutStatus `gorm:"column:status;type:text;not null;default:'pending'" json:"status"`
	NombaTransactionID *string                `gorm:"column:nomba_transaction_id;type:text;" json:"nombaTransactionId"`
	FailureReason      *string                `gorm:"column:failure_reason;type:text;" json:"failureReason"`
	ProcessedAt        *time.Time             `gorm:"column:processed_at;type:timestamp;" json:"processedAt"`
	SettlementCount    int                    `gorm:"column:settlement_count;type:int;not null;default:0" json:"settlementCount"`

	Settlements []Settlement `gorm:"foreignKey:SettlementPayoutID" json:"settlements,omitempty"`
}

func (SettlementPayout) TableName() string {
	return TableNameSettlementPayout
}
