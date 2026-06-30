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

type Frequency string

const (
	FrequencyVariable          Frequency = "VARIABLE"
	FrequencyWeekly            Frequency = "WEEKLY"
	FrequencyMonthly           Frequency = "MONTHLY"
	FrequencyQuarterly         Frequency = "QUARTERLY"
	FrequencyEveryTwoMonths    Frequency = "EVERY_TWO_MONTHS"
	FrequencyEveryThreeMonths  Frequency = "EVERY_THREE_MONTHS"
	FrequencyEveryFourMonths   Frequency = "EVERY_FOUR_MONTHS"
	FrequencyEveryFiveMonths   Frequency = "EVERY_FIVE_MONTHS"
	FrequencyEverySixMonths    Frequency = "EVERY_SIX_MONTHS"
	FrequencyEverySevenMonths  Frequency = "EVERY_SEVEN_MONTHS"
	FrequencyEveryEightMonths  Frequency = "EVERY_EIGHT_MONTHS"
	FrequencyEveryNineMonths   Frequency = "EVERY_NINE_MONTHS"
	FrequencyEveryTenMonths    Frequency = "EVERY_TEN_MONTHS"
	FrequencyEveryElevenMonths Frequency = "EVERY_ELEVEN_MONTHS"
	FrequencyEveryTwelveMonths Frequency = "EVERY_TWELVE_MONTHS"
)

type MandateStatus string

const (
	MandateStatusActive    MandateStatus = "ACTIVE"
	MandateStatusSuspended MandateStatus = "SUSPENDED"
	MandateStatusDeleted   MandateStatus = "DELETED"
)

type UpdateMandateStatus string

const (
	UpadateMandateStatusActive    MandateStatus = "ACTIVE"
	UpadateMandateStatusSuspended MandateStatus = "SUSPEND"
	UpadateMandateStatusDeleted   MandateStatus = "DELETE"
)
