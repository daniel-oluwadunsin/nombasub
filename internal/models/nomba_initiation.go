package models

import "time"

type NombaInitiationPurpose string

const (
	NombaInitiationPurposeCardSubscriptionPayment NombaInitiationPurpose = "card_subscription_payment"
	NombaInitiationPurposeWalletToWalletTransfer  NombaInitiationPurpose = "wallet_to_wallet_transfer"
	NombaInitiationPurposeChargeCardPayment       NombaInitiationPurpose = "charge_card_payment"
	NombaInitiationPurposeDirectDebitSubscription NombaInitiationPurpose = "direct_debit_subscription"
	NombaInitiationPurposeDirectDebitCharge       NombaInitiationPurpose = "direct_debit_charge"
	NombaInitiationPurposeUpdateCard              NombaInitiationPurpose = "update_card"
)

type NombaInitiationStatus string

const (
	NombaInitiationStatusPending   NombaInitiationStatus = "pending"
	NombaInitiationStatusCompleted NombaInitiationStatus = "completed"
	NombaInitiationStatusFailed    NombaInitiationStatus = "failed"
)

type NombaInitiation struct {
	BaseModel
	TenantID           string                 `json:"tenant_id" gorm:"column:tenant_id;type:uuid;not null"`
	Amount             float64                `json:"amount" gorm:"column:amount;type:numeric(10,2);not null"`
	Currency           string                 `json:"currency" gorm:"column:currency;type:varchar(3);not null"`
	Reference          string                 `json:"reference" gorm:"column:reference;type:varchar(100);not null"`
	NombaOrderID       *string                `json:"nomba_order_id" gorm:"column:nomba_order_id;type:varchar(100)"`
	Purpose            NombaInitiationPurpose `json:"purpose" gorm:"column:purpose;type:varchar(50);not null"`
	Status             NombaInitiationStatus  `json:"status" gorm:"column:status;type:varchar(20);not null"`
	Metadata           map[string]interface{} `json:"metadata" gorm:"column:metadata;type:jsonb;serializer:json"`
	NombaTransactionId *string                `json:"nombaTransactionId" gorm:"column:nomba_transaction_id;type:varchar(300)"`
	PaymentIntentId    *string                `json:"paymentIntentId" gorm:"column:payment_intent_id;type:uuid"`

	// Bookkeeping for the per-mandate exponential backoff in PollPendingMandates.
	LastPolledAt *time.Time `json:"lastPolledAt" gorm:"column:last_polled_at;type:timestamp"`
	PollAttempts int        `json:"pollAttempts" gorm:"column:poll_attempts;type:int;not null;default:0"`
}

func (NombaInitiation) TableName() string {
	return TableNameNombaInitiation
}
