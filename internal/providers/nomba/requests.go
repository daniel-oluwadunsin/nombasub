package nomba

type CreateCheckoutOrderRequest struct {
	Order struct {
		CallbackUrl           string                  `json:"callbackUrl" binding:"required"`
		CustomerEmail         string                  `json:"customerEmail" binding:"required"`
		Amount                int64                   `json:"amount" binding:"required"`
		Currency              *string                 `json:"currency" binding:"oneof=NGN"`
		OrderReference        *string                 `json:"orderReference"`
		CustomerId            *string                 `json:"customerId"`
		AccountId             *string                 `json:"accountId"`
		AllowedPaymentMethods *[]PaymentMethod        `json:"allowedPaymentMethods"`
		OrderMetaData         *map[string]interface{} `json:"orderMetaData"`
	} `json:"order"`
	TokenizeCard *bool `json:"tokenizeCard"`
}

type TransferToAccountRequest struct {
	Amount            float64 `json:"amount" binding:"required"`
	Narration         string  `json:"narration" binding:"required"`
	MerchantTxRef     string  `json:"merchantTxRef" binding:"required"`
	ReceiverAccountId string  `json:"receiverAccountId" binding:"required"`
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
			OriginatingFrom       string          `json:"originatingFrom"`
			TransactionAmount     float64         `json:"transactionAmount"`
			Narration             string          `json:"narration"`
			Time                  string          `json:"time"`
			AliasAccountReference string          `json:"aliasAccountReference"`
			AliasAccountType      string          `json:"aliasAccountType"`
			MerchantTxRef         string          `json:"merchantTxRef"`
			TokenizedCardPayment  string          `json:"tokenizedCardPayment"`
		} `json:"transaction"`
		Customer struct {
			BankCode      string `json:"bankCode"`
			SenderName    string `json:"senderName"`
			BankName      string `json:"bankName"`
			AccountNumber string `json:"accountNumber"`
		} `json:"customer"`
		Order struct {
			OrderReference  string  `json:"orderReference"`
			OrderId         string  `json:"orderId"`
			Amount          float64 `json:"amount"`
			Currency        string  `json:"currency"`
			PaymentMethod   string  `json:"paymentMethod"`
			CardType        string  `json:"cardType"`
			CardLast4Digits string  `json:"cardLast4Digits"`
			AccountId       string  `json:"accountId"`
			CardCurrency    string  `json:"cardCurrency"`
		} `json:"order"`
		TokenizedCardData *struct {
			TokenKey         string `json:"tokenKey"`
			CardType         string `json:"cardType"`
			TokenExpiryYear  string `json:"tokenExpiryYear"`
			TokenExpiryMonth string `json:"tokenExpiryMonth"`
			CardPan          string `json:"cardPan"`
		} `json:"tokenizedCardData"`
	} `json:"data"`
}
