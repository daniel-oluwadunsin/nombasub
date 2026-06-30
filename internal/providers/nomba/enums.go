package nomba

type PaymentMethod string

const (
	PaymentMethodCard PaymentMethod = "Card"
)

type TransferStatus string

const (
	TransferStatusPending   TransferStatus = "PENDING_BILLING"
	TransferStatusCompleted TransferStatus = "SUCCESS"
	TransferStatusRefund    TransferStatus = "REFUND"
)

type WebhookEventType string

const (
	WebhookEventTypePaymentSuccess  WebhookEventType = "payment_success"
	WebhookEventTypePaymentFailed   WebhookEventType = "payment_failed"
	WebhookEventTypePaymentReversal WebhookEventType = "payment_reversal"
	WebhookEventTypePayoutSuccess   WebhookEventType = "payout_success"
	WebhookEventTypePayoutFailed    WebhookEventType = "payout_failed"
	WebhookEventTypePayoutRefund    WebhookEventType = "payout_refund"
)

type TransactionType string

const (
	TransactionTypeTransfer       TransactionType = "transfer"
	TransactionTypeOnlineCheckout TransactionType = "online_checkout"
)
