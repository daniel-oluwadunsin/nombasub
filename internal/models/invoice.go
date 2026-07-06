package models

import "time"

type InvoiceStatus string

const (
	InvoiceStatusDraft    InvoiceStatus = "draft"
	InvoiceStatusOpen     InvoiceStatus = "open"
	InvoiceStatusPaid     InvoiceStatus = "paid"
	InvoiceStatusFailed   InvoiceStatus = "failed"
	InvoiceStatusRefunded InvoiceStatus = "refunded"
)

type Invoice struct {
	BaseModel
	TenantID             string        `gorm:"column:tenant_id;type:uuid;not null" json:"-"`
	SubscriptionID       string        `gorm:"column:subscription_id;type:uuid;not null" json:"subscriptionId"`
	CustomerID           string        `gorm:"column:customer_id;type:uuid;not null" json:"customerId"`
	Code                 string        `gorm:"column:code;type:text;not null" json:"code"`
	Status               InvoiceStatus `gorm:"column:status;type:text;not null;default:'draft'" json:"status"`
	AmountDue            int64         `gorm:"column:amount_due;type:bigint;not null" json:"amountDue"`
	AmountPaid           int64         `gorm:"column:amount_paid;type:bigint;not null" json:"amountPaid"`
	AmountRemaining      int64         `gorm:"column:amount_remaining;type:bigint;not null" json:"amountRemaining"`
	Currency             string        `gorm:"column:currency;type:text;not null" json:"currency"`
	BillingPeriodStart   *time.Time    `gorm:"column:billing_period_start;type:timestamp;" json:"billingPeriodStart"`
	BillingPeriodEnd     *time.Time    `gorm:"column:billing_period_end;type:timestamp;" json:"billingPeriodEnd"`
	DueAt                *time.Time    `gorm:"column:due_at;type:timestamp;" json:"dueAt"`
	PaidAt               *time.Time    `gorm:"column:paid_at;type:timestamp;" json:"paidAt"`
	FailedAt             *time.Time    `gorm:"column:failed_at;type:timestamp;" json:"failedAt"`
	RefundedAt           *time.Time    `gorm:"column:refunded_at;type:timestamp;" json:"refundedAt"`
	NextPaymentAttemptAt *time.Time    `gorm:"column:next_payment_attempt_at;type:timestamp;" json:"nextPaymentAttemptAt"`
	AttemptCount         int           `gorm:"column:attempt_count;type:int;not null;default:0" json:"attemptCount"`
	FailureReason        *string       `gorm:"column:failure_reason;type:text;" json:"failureReason"`
	CheckoutLink         *string       `gorm:"column:checkout_link;type:text;" json:"checkoutLink"`

	Subscription *Subscription `gorm:"foreignKey:SubscriptionID;references:ID" json:"subscription,omitempty"`
	Customer     *Customer     `gorm:"foreignKey:CustomerID;references:ID" json:"customer,omitempty"`
}

func (Invoice) TableName() string {
	return TableNameInvoices
}
