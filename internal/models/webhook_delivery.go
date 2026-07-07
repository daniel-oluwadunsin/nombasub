package models

type WebhookDeliveryStatus string

const (
	WebhookDeliveryStatusPending   WebhookDeliveryStatus = "PENDING"
	WebhookDeliveryStatusDelivered WebhookDeliveryStatus = "DELIVERED"
	WebhookDeliveryStatusFailed    WebhookDeliveryStatus = "FAILED"
)

type WebhookDeliveryEventType string

const (
	WebhookDeliveryEventTypeCustomerUpdated WebhookDeliveryEventType = "customer.updated"

	WebhookDeliveryEventTypePaymentMethodAttached WebhookDeliveryEventType = "payment_method.attached"
	WebhookDeliveryEventTypePaymentMethodDetached WebhookDeliveryEventType = "payment_method.detached"
	WebhookDeliveryEventTypePaymentMethodUpdated  WebhookDeliveryEventType = "payment_method.updated"
	WebhookDeliveryEventTypePaymentMethodDisabled WebhookDeliveryEventType = "payment_method.disabled"

	// invoices
	WebhookDeliveryEventTypeInvoiceUpcoming            WebhookDeliveryEventType = "invoice.upcoming"
	WebhookDeliveryEventTypeInvoiceCreated             WebhookDeliveryEventType = "invoice.created"
	WebhookDeliveryEventTypeInvoicePaymentAttempted    WebhookDeliveryEventType = "invoice.payment_attempted"
	WebhookDeliveryEventTypeInvoicePaid                WebhookDeliveryEventType = "invoice.paid"
	WebhookDeliveryEventTypeInvoicePaymentFailed       WebhookDeliveryEventType = "invoice.payment_failed"
	WebhookDeliveryEventTypeInvoiceMarkedUncollectible WebhookDeliveryEventType = "invoice.marked_uncollectible"
	WebhookDeliveryEventTypeInvoiceVoided              WebhookDeliveryEventType = "invoice.voided"
	WebhookDeliveryEventTypeInvoiceRefunded            WebhookDeliveryEventType = "invoice.refunded"
	WebhookDeliveryEventTypeCheckoutCreated            WebhookDeliveryEventType = "checkout.created"
	WebhookDeliveryEventTypeOrderSuccess               WebhookDeliveryEventType = "payment_success" // same as nomba incoming webhook

	// subscriptions
	WebhookDeliveryEventTypeSubscriptionCreated        WebhookDeliveryEventType = "subscription.created"
	WebhookDeliveryEventTypeSubscriptionPastDue        WebhookDeliveryEventType = "subscription.past_due"
	WebhookDeliveryEventTypeSubscriptionPaused         WebhookDeliveryEventType = "subscription.paused"
	WebhookDeliveryEventTypeSubscriptionCanceled       WebhookDeliveryEventType = "subscription.canceled"
	WebhookDeliveryEventTypeSubscriptionCompleted      WebhookDeliveryEventType = "subscription.completed"
	WebhookDeliveryEventTypeSubscriptionPaymentUpdated WebhookDeliveryEventType = "subscription.payment_method_updated"
	WebhookDeliveryEventTypeSubscriptionCardExpiring   WebhookDeliveryEventType = "subscription.card_expiring"
	WebhookDeliveryEventTypeSubscriptionTrialEnding    WebhookDeliveryEventType = "subscription.trial_ending_soon"
	WebhookDeliveryEventTypeSubscriptionBillingStarted WebhookDeliveryEventType = "subscription.billing_started"

	// settlements
	WebhookDeliveryEventTypeSettlementPayoutInitiated WebhookDeliveryEventType = "settlement.payout_initiated"
	WebhookDeliveryEventTypeSettlementPayoutFailed    WebhookDeliveryEventType = "settlement.payout_failed"

	// mandates (direct debit)
	WebhookDeliveryEventTypeMandateCreated          WebhookDeliveryEventType = "mandate.created"
	WebhookDeliveryEventTypeMandateActivated        WebhookDeliveryEventType = "mandate.activated"
	WebhookDeliveryEventTypeMandateActivationFailed WebhookDeliveryEventType = "mandate.activation_failed"
	WebhookDeliveryEventTypeMandateSuspended        WebhookDeliveryEventType = "mandate.suspended"
	WebhookDeliveryEventTypeMandateDeleted          WebhookDeliveryEventType = "mandate.deleted"
	WebhookDeliveryEventTypeMandateDebitSuccess     WebhookDeliveryEventType = "mandate.debit_success"
	WebhookDeliveryEventTypeMandateDebitFailed      WebhookDeliveryEventType = "mandate.debit_failed"
)

type WebhookDelivery struct {
	BaseModel
	TenantID        string                   `gorm:"column:tenant_id;type:uuid;not null" json:"-"`
	EndpointURL     string                   `gorm:"column:endpoint_url;type:text;not null" json:"endpointUrl"`
	EventType       WebhookDeliveryEventType `gorm:"column:event_type;type:text" json:"eventType"`
	Payload         string                   `gorm:"column:payload;type:text;not null" json:"payload"`
	Status          WebhookDeliveryStatus    `gorm:"column:status;type:text;not null" json:"status"`
	AttempsCount    int                      `gorm:"column:retry_count;type:int;not null;default:0" json:"retryCount"`
	MaxAttemptCount int                      `gorm:"column:max_attempt_count;type:int;not null;default:3" json:"maxAttemptCount"`
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
