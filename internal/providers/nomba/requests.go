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
		} `json:"order"`
	} `json:"data"`
}
