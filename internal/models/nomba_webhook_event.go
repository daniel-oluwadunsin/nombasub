package models

import "time"

type NombaWebhookProcessingStatus string

const (
	NombaWebhookProcessingStatusPending   NombaWebhookProcessingStatus = "PENDING"
	NombaWebhookProcessingStatusProcessed NombaWebhookProcessingStatus = "PROCESSED"
	NombaWebhookProcessingStatusFailed    NombaWebhookProcessingStatus = "FAILED"
)

type NombaWebhookEvent struct {
	BaseModel
	EventType   string                       `gorm:"column:event_type;type:text;not null" json:"eventType"`
	Payload     string                       `gorm:"column:payload;type:text;not null" json:"payload"`
	Status      NombaWebhookProcessingStatus `gorm:"column:status;type:text;not null" json:"status"`
	CompletedAt *time.Time                   `gorm:"column:completed_at;type:timestamp;" json:"completedAt"`
	FailedAt    *time.Time                   `gorm:"column:failed_at;type:timestamp;" json:"failedAt"`
}

func (NombaWebhookEvent) TableName() string {
	return TableNameNombaWebhookEvent
}
