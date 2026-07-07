package models

import "time"

type PortalSession struct {
	BaseModel
	TenantID             string     `gorm:"column:tenant_id;type:uuid;not null" json:"-"`
	CustomerID           string     `gorm:"column:customer_id;type:uuid;not null" json:"customerId"`
	CodeHash             *string    `gorm:"column:code_hash;type:text;" json:"-"`
	CodeExpiresAt        *time.Time `gorm:"column:code_expires_at;type:timestamp;" json:"codeExpiresAt"`
	VerifiedAt           *time.Time `gorm:"column:verified_at;type:timestamp;" json:"verifiedAt"`
	AccessTokenHash      *string    `gorm:"column:access_token_hash;type:text;" json:"-"`
	AccessTokenExpiresAt *time.Time `gorm:"column:access_token_expires_at;type:timestamp;" json:"accessTokenExpiresAt"`
	RevokedAt            *time.Time `gorm:"column:revoked_at;type:timestamp;" json:"revokedAt"`

	Tenant   *Tenant   `gorm:"foreignKey:TenantID;references:ID" json:"tenant,omitempty"`
	Customer *Customer `gorm:"foreignKey:CustomerID;references:ID" json:"customer,omitempty"`
}

func (PortalSession) TableName() string {
	return TableNamePortalSession
}
