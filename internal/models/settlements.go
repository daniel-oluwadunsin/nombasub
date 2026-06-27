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
	TenantID       string                 `gorm:"column:tenant_id;type:uuid;not null" json:"-"`
	Amount         float64                `gorm:"column:amount;type:numeric(10,2);not null" json:"amount"`
	Currency       string                 `gorm:"column:currency;type:varchar(3);not null" json:"currency"`
	SettlementTime time.Time              `gorm:"column:settlement_date;type:varchar(10);not null" json:"settlementDate"`
	Status         SettlementStatus       `gorm:"column:status;type:text;not null;default:'pending'" json:"status"`
	Reference      string                 `gorm:"column:reference;type:varchar(100);not null" json:"reference"`
	FailureReason  *string                `gorm:"column:failure_reason;type:text;" json:"failureReason"`
	Purpose        NombaInitiationPurpose `gorm:"column:purpose;type:text;not null" json:"purpose"`
}

func (Settlement) TableName() string {
	return TableNameSettlement
}
