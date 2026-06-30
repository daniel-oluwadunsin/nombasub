package models

type WebhookDeliveryStatus string

const (
	WebhookDeliveryStatusPending   WebhookDeliveryStatus = "PENDING"
	WebhookDeliveryStatusDelivered WebhookDeliveryStatus = "DELIVERED"
	WebhookDeliveryStatusFailed    WebhookDeliveryStatus = "FAILED"
)

type WebhookDelivery struct {
	BaseModel
	TenantID        string                `gorm:"column:tenant_id;type:uuid;not null" json:"tenantId"`
	EndpointURL     string                `gorm:"column:endpoint_url;type:text;not null" json:"endpointUrl"`
	Payload         string                `gorm:"column:payload;type:text;not null" json:"payload"`
	Status          WebhookDeliveryStatus `gorm:"column:status;type:text;not null" json:"status"`
	AttempsCount    int                   `gorm:"column:retry_count;type:int;not null;default:0" json:"retryCount"`
	MaxAttemptCount int                   `gorm:"column:max_attempt_count;type:int;not null;default:3" json:"maxAttemptCount"`
}

func (WebhookDelivery) TableName() string {
	return TableNameWebhookDelivery
}

type WebhookDeliveryAttempt struct {
	BaseModel
	WebhookDeliveryID string `gorm:"column:webhook_delivery_id;type:uuid;not null" json:"webhookDeliveryId"`
	StatusCode        int    `gorm:"column:status_code;type:int;not null" json:"statusCode"`
	ResponseBody      string `gorm:"column:response_body;type:text;" json:"responseBody"`
	AttemptCount      int    `gorm:"column:attempt_count;type:int;not null" json:"attemptCount"`
}

func (WebhookDeliveryAttempt) TableName() string {
	return TableNameWebhookDeliveryAttempt
}
