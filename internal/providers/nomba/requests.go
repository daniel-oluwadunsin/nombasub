package nomba

import "github.com/daniel-oluwadunsin/nombasub/internal/models"

type NombaOrder struct {
	CallbackUrl           string                  `json:"callbackUrl" binding:"required"`
	CustomerEmail         string                  `json:"customerEmail" binding:"required"`
	Amount                int64                   `json:"amount" binding:"required"`
	Currency              *string                 `json:"currency,omitempty" binding:"oneof=NGN"`
	OrderReference        *string                 `json:"orderReference,omitempty"`
	CustomerId            *string                 `json:"customerId,omitempty"`
	AccountId             *string                 `json:"accountId,omitempty"`
	AllowedPaymentMethods *[]PaymentMethod        `json:"allowedPaymentMethods,omitempty"`
	OrderMetaData         *map[string]interface{} `json:"orderMetaData,omitempty"`
}

type CreateCheckoutOrderRequest struct {
	Order        NombaOrder `json:"order"`
	TokenizeCard *bool      `json:"tokenizeCard,omitempty"`
}

type ChargeCardRequest struct {
	Order    NombaOrder `json:"order"`
	TokenKey string     `json:"tokenKey"`
}

type TransferToAccountRequest struct {
	Amount            float64 `json:"amount" binding:"required"`
	Narration         string  `json:"narration" binding:"required"`
	MerchantTxRef     string  `json:"merchantTxRef" binding:"required"`
	ReceiverAccountId string  `json:"receiverAccountId" binding:"required"`
	SenderName        string  `json:"senderName" binding:"required"`
}

type CreateDirectDebitManadateRequest struct {
	CustomerAccountNumber string    `json:"customerAccountNumber" binding:"required"`
	BankCode              string    `json:"bankCode" binding:"required"`
	CustomerName          string    `json:"customerName" binding:"required"`
	CustomerAddress       string    `json:"customerAddress" binding:"required"`
	CustomerAccountName   string    `json:"customerAccountName" binding:"required"`
	Frequency             Frequency `json:"frequency" binding:"required,oneof=VARIABLE WEEKLY MONTHLY QUARTERLY EVERY_TWO_MONTHS EVERY_THREE_MONTHS EVERY_FOUR_MONTHS EVERY_FIVE_MONTHS EVERY_SIX_MONTHS EVERY_SEVEN_MONTHS EVERY_EIGHT_MONTHS EVERY_NINE_MONTHS EVERY_TEN_MONTHS EVERY_ELEVEN_MONTHS EVERY_TWELVE_MONTHS"`
	Narration             string    `json:"narration" binding:"required"`
	CustomerPhoneNumber   string    `json:"customerPhoneNumber" binding:"required"`
	MerchantReference     string    `json:"merchantReference" binding:"required"`
	StartDate             string    `json:"startDate" binding:"required"`
	EndDate               string    `json:"endDate" binding:"required"`
	StartImmediately      bool      `json:"startImmediately" binding:"required"`
}

type DebitMandateRequest struct {
	MandateId string  `json:"mandateId" binding:"required"`
	Amount    float64 `json:"amount" binding:"required"`
}

type UpdateDirectDebitManadateRequest struct {
	MandateId     string        `json:"mandateId" binding:"required"`
	MandateStatus MandateStatus `json:"mandateStatus" binding:"required,oneof=ACTIVE SUSPENDED DELETED"`
}

type NombaWebhookRequest struct {
	EventType WebhookEventType `json:"event_type"`
	RequestID string           `json:"requestId"`
	Data      struct {
		Merchant struct {
			WalletID      string  `json:"walletId"`
			WalletBalance float64 `json:"walletBalance"`
			UserID        string  `json:"userId"`
		} `json:"merchant"`
		Terminal struct {
			TerminalID    string `json:"terminalId"`
			TerminalLabel string `json:"terminalLabel"`
		} `json:"terminal"`
		Transaction struct {
			AliasAccountNumber    string          `json:"aliasAccountNumber"`
			Fee                   float64         `json:"fee"`
			SessionID             string          `json:"sessionId"`
			Type                  TransactionType `json:"type"`
			TransactionID         string          `json:"transactionId"`
			AliasAccountName      string          `json:"aliasAccountName"`
			ResponseCode          string          `json:"responseCode"`
			ResponseCodeMessage   string          `json:"responseCodeMessage"`
			OriginatingFrom       string          `json:"originatingFrom"`
			TransactionAmount     float64         `json:"transactionAmount"`
			Narration             string          `json:"narration"`
			Time                  string          `json:"time"`
			AliasAccountReference string          `json:"aliasAccountReference"`
			AliasAccountType      string          `json:"aliasAccountType"`
			MerchantTxRef         string          `json:"merchantTxRef"`
			TokenizedCardPayment  string          `json:"tokenizedCardPayment"`
			IsSubscriptionPayment bool            `json:"isSubscriptionPayment"`
		} `json:"transaction"`
		Customer struct {
			BankCode      string `json:"bankCode"`
			SenderName    string `json:"senderName"`
			BankName      string `json:"bankName"`
			AccountNumber string `json:"accountNumber"`
		} `json:"customer"`
		Order struct {
			OrderReference       string  `json:"orderReference"`
			OrderId              string  `json:"orderId"`
			Amount               float64 `json:"amount"`
			Currency             string  `json:"currency"`
			PaymentMethod        string  `json:"paymentMethod"`
			CardType             string  `json:"cardType"`
			CardLast4Digits      string  `json:"cardLast4Digits"`
			AccountId            string  `json:"accountId"`
			CardCurrency         string  `json:"cardCurrency"`
			IsSubscription       bool    `json:"isSubscription"`
			SubscriptionPlanCode string  `json:"subscriptionPlanCode"`
		} `json:"order"`
		TokenizedCardData *struct {
			TokenKey         string `json:"tokenKey"`
			CardType         string `json:"cardType"`
			TokenExpiryYear  string `json:"tokenExpiryYear"`
			TokenExpiryMonth string `json:"tokenExpiryMonth"`
			CardPan          string `json:"cardPan"`
		} `json:"tokenizedCardData"`
		Subscription *models.Subscription `json:"subscription,omitempty"`
	} `json:"data"`
}
