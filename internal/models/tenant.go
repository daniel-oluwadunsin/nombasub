package models

import "time"

type Tenant struct {
	BaseModel
	AccountID    string  `gorm:"column:account_id;type:text;not null" json:"accountId"`
	ClientID     string  `gorm:"column:client_id;type:text;not null" json:"clientId"`
	ClientSecret string  `gorm:"column:client_secret;type:text;not null" json:"-"`
	WebhookUrl   *string `gorm:"column:webhook_url;type:text" json:"webhookUrl"`

	AccessToken          *string    `gorm:"column:access_token;type:text" json:"accessToken"`
	RefreshToken         *string    `gorm:"column:refresh_token;type:text" json:"refreshToken"`
	AccessTokenExpiresAt *time.Time `gorm:"column:access_token_expires_at;type:timestamp" json:"accessTokenExpiresAt"`
	ApiKey               string     `gorm:"column:api_key;type:text;not null" json:"-"`

	Nonce      string `gorm:"column:encryption_nonce;type:text;not null" json:"nonce"`
	Algorithm  string `gorm:"column:encryption_algorithm;type:text;not null" json:"algorithm"`
	KeyVersion string `gorm:"column:encryption_key_version;type:text;not null" json:"keyVersion"`
}

func (Tenant) TableName() string {
	return TableNameTenant
}
