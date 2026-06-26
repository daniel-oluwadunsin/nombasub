package models

type Customer struct {
	BaseModel
	TenantID    string  `gorm:"column:tenant_id;type:uuid;not null" json:"-"`
	Name        *string `gorm:"column:name;type:text" json:"name"`
	Email       string  `gorm:"column:email;type:text;not null" json:"email"`
	PhoneNumber *string `gorm:"column:phone_number;type:text" json:"phoneNumber"`
	Code        string  `gorm:"column:code;type:text;not null" json:"code"`
	ExternalRef *string `gorm:"column:external_ref;type:text;" json:"externalRef"`

	PaymentSources *[]PaymentSource `gorm:"foreignKey:CustomerID;references:ID" json:"paymentSources,omitempty"`
	Subscriptions  *[]Subscription  `gorm:"foreignKey:CustomerID;references:ID" json:"subscriptions,omitempty"`
	Payments       *[]PaymentIntent `gorm:"foreignKey:CustomerID;references:ID" json:"payments,omitempty"`
}

func (Customer) TableName() string {
	return TableNameCustomer
}
