package models

import "time"

type PlanInterval string

const (
	PlanIntervalDaily     PlanInterval = "daily"
	PlanIntervalWeekly    PlanInterval = "weekly"
	PlanIntervalBiWeekly  PlanInterval = "bi-weekly"
	PlanIntervalMonthly   PlanInterval = "monthly"
	PlanIntervalQuarterly PlanInterval = "quarterly"
	PlanIntervalYearly    PlanInterval = "yearly"
)

type PlanStatus string

const (
	PlanStatusActive   PlanStatus = "active"
	PlanStatusInactive PlanStatus = "inactive"
)

type Plan struct {
	BaseModel
	Name            string       `gorm:"column:name;type:text;not null" json:"name"`
	Description     *string      `gorm:"column:description;type:text" json:"description"`
	TenantID        string       `gorm:"column:tenant_id;type:text;not null" json:"-"`
	Code            string       `gorm:"column:code;type:text;not null" json:"code"`
	Amount          int64        `gorm:"column:amount;type:bigint;not null" json:"amount"`
	Currency        string       `gorm:"column:currency;type:text;not null" json:"currency"`
	Interval        PlanInterval `gorm:"column:interval;type:text;not null" json:"interval"`
	IntervalCount   *int         `gorm:"column:interval_count;type:int;" json:"intervalCount"`
	TrialPeriodDays int          `gorm:"column:trial_period_days;type:int;default:0" json:"trialPeriodDays"`
	InvoiceLimit    *int         `gorm:"column:invoice_limit;type:int;" json:"invoiceLimit"`
	Status          PlanStatus   `gorm:"column:status;type:text;not null;default:'active'" json:"status"`
	ArchivedAt      *time.Time   `gorm:"column:archived_at;type:timestamp" json:"archivedAt"`
}

func (Plan) TableName() string {
	return TableNamePlan
}

type PlanVersion struct {
	BaseModel
	PlanID          string       `gorm:"column:plan_id;type:text;not null" json:"planId"`
	Index           int          `gorm:"column:index;type:int;not null" json:"index"`
	Name            string       `gorm:"column:name;type:text;not null" json:"name"`
	Description     *string      `gorm:"column:description;type:text" json:"description"`
	Code            string       `gorm:"column:code;type:text;not null" json:"code"`
	Amount          int64        `gorm:"column:amount;type:bigint;not null" json:"amount"`
	Interval        PlanInterval `gorm:"column:interval;type:text;not null" json:"interval"`
	IntervalCount   *int         `gorm:"column:interval_count;type:int;" json:"intervalCount"`
	TrialPeriodDays int          `gorm:"column:trial_period_days;type:int;default:0" json:"trialPeriodDays"`
	InvoiceLimit    *int         `gorm:"column:invoice_limit;type:int;" json:"invoiceLimit"`
	Status          PlanStatus   `gorm:"column:status;type:text;not null;default:'active'" json:"status"`
	ArchivedAt      *time.Time   `gorm:"column:archived_at;type:timestamp" json:"archivedAt"`
}

func (PlanVersion) TableName() string {
	return TableNamePlanVersion
}
