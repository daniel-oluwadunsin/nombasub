package models

type PaymentSourceType string

const (
	PaymentSourceTypeCard PaymentSourceType = "card"
	PaymentSourceTypeBank PaymentSourceType = "bank"
)

type PaymentSourceStatus string

const (
	PaymentSourceStatusActive   PaymentSourceStatus = "active"
	PaymentSourceStatusInactive PaymentSourceStatus = "inactive"
)

type PaymentSource struct {
	BaseModel
	TenantID   string              `gorm:"column:tenant_id;type:uuid;not null" json:"-"`
	CustomerID string              `gorm:"column:customer_id;type:uuid;not null" json:"customerId"`
	Type       PaymentSourceType   `gorm:"column:type;type:text;not null" json:"type"`
	Card       *CardPaymentSource  `gorm:"embedded;embeddedPrefix:card_" json:"card,omitempty"`
	Bank       *BankPaymentSource  `gorm:"embedded;embeddedPrefix:bank_" json:"bank,omitempty"`
	Status     PaymentSourceStatus `gorm:"column:status;type:text;not null;default:'active'" json:"status"`

	Customer *Customer `gorm:"foreignKey:CustomerID;references:ID" json:"customer,omitempty"`
}

type CardPaymentSource struct {
	Type               string  `gorm:"column:type;type:text;" json:"type"`
	Pan                *string `gorm:"column:pan;type:text;" json:"pan"`
	Last4Digits        *string `gorm:"column:last4_digits;type:text;" json:"last4Digits"`
	Currency           *string `gorm:"column:currency;type:text;" json:"currency"`
	AuthorizationToken *string `gorm:"column:authorization_token;type:text;" json:"authorizationToken"`
}

type MandateStatus string

const (
	MandateStatusAdviceSent MandateStatus = "ADVICE_SENT"
	MandateStatusActive     MandateStatus = "ACTIVE"
	MandateStatusSuspend    MandateStatus = "SUSPEND"
	MandateStatusDeleted    MandateStatus = "DELETED"
)

type BankPaymentSource struct {
	Name      *string `gorm:"column:bank_name;type:text;" json:"bankName"`
	Code      *string `gorm:"column:bank_code;type:text;" json:"bankCode"`
	Last4     *string `gorm:"column:last4;type:text;" json:"last4"`
	MandateID *string `gorm:"column:mandate_id;type:text;" json:"mandateId"`
}
