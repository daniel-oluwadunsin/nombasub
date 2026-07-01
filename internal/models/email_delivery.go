package models

type EmailTemplateName string

const (
	EmailTemplateSubscriptionCreated      EmailTemplateName = "subscription_created.html"
	EmailTemplateCheckoutPaymentRequired  EmailTemplateName = "checkout_payment_required.html"
	EmailTemplateSubscriptionActivated    EmailTemplateName = "subscription_activated.html"
	EmailTemplateTrialStarted             EmailTemplateName = "trial_started.html"
	EmailTemplateTrialEndingSoon          EmailTemplateName = "trial_ending_soon.html"
	EmailTemplateTrialEndedBillingStarted EmailTemplateName = "trial_ended_billing_started.html"
	EmailTemplateUpcomingInvoice          EmailTemplateName = "upcoming_invoice.html"
	EmailTemplateInvoiceCreated           EmailTemplateName = "invoice_created.html"
	EmailTemplatePaymentSuccessful        EmailTemplateName = "payment_successful.html"
	EmailTemplatePaymentReceipt           EmailTemplateName = "payment_receipt.html"
	EmailTemplateInvoicePaid              EmailTemplateName = "invoice_paid.html"
	EmailTemplateSubscriptionCardExpiring EmailTemplateName = "subscription_card_expiring.html"
	EmailTemplateSubscriptionPaused       EmailTemplateName = "subscription_paused.html"
)

type EmailContext struct {
	Preheader          string `json:"preheader"`
	Title              string `json:"title"`
	GreetingName       string `json:"greetingName"`
	Intro              string `json:"intro"`
	Body               string `json:"body"`
	BusinessName       string `json:"businessName"`
	CustomerEmail      string `json:"customerEmail"`
	PlanName           string `json:"planName"`
	PlanCode           string `json:"planCode"`
	SubscriptionCode   string `json:"subscriptionCode"`
	SubscriptionStatus string `json:"subscriptionStatus"`
	InvoiceCode        string `json:"invoiceCode"`
	InvoiceStatus      string `json:"invoiceStatus"`
	Amount             string `json:"amount"`
	Currency           string `json:"currency"`
	DueDate            string `json:"dueDate"`
	BillingPeriod      string `json:"billingPeriod"`
	TrialStartDate     string `json:"trialStartDate"`
	TrialEndDate       string `json:"trialEndDate"`
	PaymentDate        string `json:"paymentDate"`
	ReceiptReference   string `json:"receiptReference"`
	CheckoutURL        string `json:"checkoutUrl"`
	CardLast4          string `json:"cardLast4"`
	CardExpiry         string `json:"cardExpiry"`
	PrimaryActionLabel string `json:"primaryActionLabel"`
	PrimaryActionURL   string `json:"primaryActionUrl"`
	SecondaryNote      string `json:"secondaryNote"`
}

type EmailDeliveryStatus string

const (
	EmailDeliveryStatusPending   EmailDeliveryStatus = "PENDING"
	EmailDeliveryStatusDelivered EmailDeliveryStatus = "DELIVERED"
	EmailDeliveryStatusFailed    EmailDeliveryStatus = "FAILED"
)

type EmailDelivery struct {
	BaseModel
	Recipient      string              `gorm:"column:recipient;type:text;not null" json:"recipient"`
	Subject        string              `gorm:"column:subject;type:text;not null" json:"subject"`
	TemplateName   EmailTemplateName   `gorm:"column:template_name;type:text;not null" json:"templateName"`
	Context        EmailContext        `gorm:"column:context;type:jsonb;serializer:json;not null" json:"context"`
	IdempotencyKey string              `gorm:"column:idempotency_key;type:text;not null;uniqueIndex" json:"idempotencyKey"`
	Status         EmailDeliveryStatus `gorm:"column:status;type:text;not null;default:'PENDING'" json:"status"`
	AttemptCount   int                 `gorm:"column:attempt_count;type:int;not null;default:0" json:"attemptCount"`
	FailureReason  *string             `gorm:"column:failure_reason;type:text;" json:"failureReason"`
}

func (EmailDelivery) TableName() string {
	return TableNameEmailDelivery
}
