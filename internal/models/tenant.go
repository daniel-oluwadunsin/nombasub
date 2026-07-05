package models

import "time"

type Tenant struct {
	BaseModel
	BusinessName         *string    `gorm:"column:business_name;type:text" json:"businessName"`
	AccountID            string     `gorm:"column:account_id;type:text;not null" json:"accountId"` // sub account ID from nomba
	WebhookUrl           *string    `gorm:"column:webhook_url;type:text" json:"webhookUrl"`
	WebhookSecret        *string    `gorm:"column:webhook_secret;type:text" json:"webhookSecret"`
	ApiKey               string     `gorm:"column:api_key;type:text;not null" json:"-"`
	Password             *string    `gorm:"column:password;type:text" json:"-"`
	AccessToken          *string    `gorm:"column:access_token;type:text" json:"-"`
	AccessTokenExpiresAt *time.Time `gorm:"column:access_token_expires_at;type:timestamp" json:"-"`
}

func (Tenant) TableName() string {
	return TableNameTenant
}
