package models

import "time"

type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"
	SubscriptionStatusExpired  SubscriptionStatus = "expired"
)

type Subscription struct {
	BaseModel
	TenantID                     string            `gorm:"column:tenant_id;type:uuid;not null" json:"-"`
	CustomerID                   string            `gorm:"column:customer_id;type:uuid;not null" json:"customerId"`
	PlanID                       string            `gorm:"column:plan_id;type:uuid;not null" json:"planId"`
	Code                         string            `gorm:"column:code;type:text;not null" json:"code"`
	PlanVersionID                string            `gorm:"column:plan_version_id;type:uuid;not null" json:"planVersionId"`
	PaymentSourceID              string            `gorm:"column:payment_source_id;type:uuid;not null" json:"paymentSourceId"`
	PaymentSourceType            PaymentSourceType `gorm:"column:payment_source_type;type:text;not null" json:"paymentSourceType"`
	Interval                     PlanInterval      `gorm:"column:interval;type:text;not null" json:"interval"`
	Amount                       int64             `gorm:"column:amount;type:bigint;not null" json:"amount"`
	Currency                     string            `gorm:"column:currency;type:text;not null" json:"currency"`
	IntervalCount                *int              `gorm:"column:interval_count;type:int;" json:"intervalCount"`
	TrialPeriodDays              int               `gorm:"column:trial_period_days;type:int;default:0" json:"trialPeriodDays"`
	TrialStartDate               *time.Time        `gorm:"column:trial_start_date;type:timestamp;" json:"trialStartDate"`
	TrialEndDate                 *time.Time        `gorm:"column:trial_end_date;type:timestamp;" json:"trialEndDate"`
	CurrentBillingCycleStart     *time.Time        `gorm:"column:current_billing_cycle_start;type:timestamp;" json:"currentBillingCycleStart"`
	CurrentBillingCycleEnd       *time.Time        `gorm:"column:current_billing_cycle_end;type:timestamp;" json:"currentBillingCycleEnd"`
	CancelledAtEndOfBillingCycle bool              `gorm:"column:cancelled_at_end_of_billing_cycle;type:boolean;default:false" json:"cancelledAtEndOfBillingCycle"`
	CancelledAt                  *time.Time        `gorm:"column:cancelled_at;type:timestamp;" json:"cancelledAt"`
	CompletedAt                  *time.Time        `gorm:"column:completed_at;type:timestamp;" json:"completedAt"`
	PausedAt                     *time.Time        `gorm:"column:paused_at;type:timestamp;" json:"pausedAt"`
	InvoiceLimit                 *int              `gorm:"column:invoice_limit;type:int;" json:"invoiceLimit"`
	InvoiceCount                 int               `gorm:"column:invoice_count;type:int;default:0" json:"invoiceCount"`
	LatestInvoiceID              *string           `gorm:"column:latest_invoice_id;type:uuid;" json:"latestInvoiceId"`

	Status SubscriptionStatus `gorm:"column:status;type:text;not null;default:'active'" json:"status"`
}

func (Subscription) TableName() string {
	return TableNameSubscription
}
