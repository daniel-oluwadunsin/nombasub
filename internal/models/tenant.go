package models

type Tenant struct {
	BaseModel
	BusinessName  *string `gorm:"column:business_name;type:text" json:"businessName"`
	AccountID     string  `gorm:"column:account_id;type:text;not null" json:"accountId"` // sub account ID from nomba
	WebhookUrl    *string `gorm:"column:webhook_url;type:text" json:"webhookUrl"`
	WebhookSecret *string `gorm:"column:webhook_secret;type:text" json:"webhookSecret"`
	ApiKey        string  `gorm:"column:api_key;type:text;not null" json:"-"`
}

func (Tenant) TableName() string {
	return TableNameTenant
}
