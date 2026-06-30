package models

import "time"

type PaymentIntentStatus string

const (
	PaymentIntentStatusPendingBilling PaymentIntentStatus = "PENDING_BILLING"
	PaymentIntentStatusSuccess        PaymentIntentStatus = "SUCCESS"
	PaymentIntentStatusFailed         PaymentIntentStatus = "PAYMENT_FAILED"
	PaymentIntentStatusRefund         PaymentIntentStatus = "REFUND"
	PaymentIntentStatusCancelled      PaymentIntentStatus = "CANCELLED"
)

type PaymentIntent struct {
	BaseModel
	TenantID                           string              `gorm:"column:tenant_id;type:uuid;not null" json:"-"`
	CustomerID                         string              `gorm:"column:customer_id;type:uuid;not null" json:"customerId"`
	SubscriptionID                     string              `gorm:"column:subscription_id;type:uuid;not null" json:"subscriptionId"`
	PlanID                             string              `gorm:"column:plan_id;type:uuid;not null" json:"planId"`
	PlanVersionID                      string              `gorm:"column:plan_version_id;type:uuid;not null" json:"planVersionId"`
	PaymentSourceID                    string              `gorm:"column:payment_source_id;type:uuid;not null" json:"paymentSourceId"`
	PaymentSourceType                  PaymentSourceType   `gorm:"column:payment_source_type;type:text;not null" json:"paymentSourceType"`
	Code                               string              `gorm:"column:code;type:text;not null" json:"code"`
	Reference                          string              `gorm:"column:reference;type:text;" json:"reference"`
	IdempotencyKey                     *string             `gorm:"column:idempotency_key;type:text;" json:"idempotencyKey"`
	ProviderResponse                   *string             `gorm:"column:provider_response;type:text;" json:"providerResponse"`
	ProviderTransactionID              *string             `gorm:"column:provider_transaction_id;type:text;" json:"providerTransactionId"`
	ProviderTransactionReference       *string             `gorm:"column:provider_transaction_reference;type:text;" json:"providerTransactionReference"`
	ProviderRefundTransactionID        *string             `gorm:"column:provider_refund_transaction_id;type:text;" json:"providerRefundTransactionId"`
	ProviderRefundTransactionReference *string             `gorm:"column:provider_refund_transaction_reference;type:text;" json:"providerRefundTransactionReference"`
	Amount                             int64               `gorm:"column:amount;type:bigint;not null" json:"amount"`
	Currency                           string              `gorm:"column:currency;type:text;not null" json:"currency"`
	FailureReason                      *string             `gorm:"column:failure_reason;type:text;" json:"failureReason"`
	Status                             PaymentIntentStatus `gorm:"column:status;type:text;not null;default:'PENDING_BILLING'" json:"status"`
	AttemptedAt                        *time.Time          `gorm:"column:attempted_at;type:timestamp;" json:"attemptedAt"`
	CompletedAt                        *time.Time          `gorm:"column:completed_at;type:timestamp;" json:"completedAt"`
	FailedAt                           *time.Time          `gorm:"column:failed_at;type:timestamp;" json:"failedAt"`
}

func (PaymentIntent) TableName() string {
	return TableNamePaymentIntent
}
