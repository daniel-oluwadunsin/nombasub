package nomba

type Response[T any] struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Data        T      `json:"data"`
}

type errorResponse = Response[struct{}]

type GetAccessTokenResponse = Response[struct {
	BusinessID   string `json:"businessId"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expiresAt"`
}]

type CreateCheckoutOrderResponse = Response[struct {
	CheckoutLink   string `json:"checkoutLink"`
	OrderReference string `json:"orderReference"`
}]

type ChargeCardResponse = Response[struct{}]

type CreateDirectDebitManadateResponse = Response[struct {
	MandateID           string `json:"mandateId"`
	MerchantReference   string `json:"merchantReference"`
	CustomerPhoneNumber string `json:"customerPhoneNumber"`
	Description         string `json:"description"`
}]

type GetDirectDebitManadateResponse = Response[struct {
	MandateId             string        `json:"mandateId"`
	CustomerAccountName   string        `json:"customerAccountName"`
	CustomerAccountNumber string        `json:"customerAccountNumber"`
	MandateStatus         MandateStatus `json:"mandateStatus"`
	RejectionReason       string        `json:"rejectionReason"`
	MandateAdviceStatus   string        `json:"mandateAdviceStatus"`
}]

type TransferToAccountResponse = Response[struct {
	ID               *string        `json:"id"`
	Status           TransferStatus `json:"status"`
	Type             *string        `json:"type"`
	Amount           *float64       `json:"amount"`
	Source           *string        `json:"source"`
	SourceUserId     *string        `json:"sourceUserId"`
	CustomerBillerId *string        `json:"customerBillerId"`
	ProductId        *string        `json:"productId"`
	Meta             *struct {
		BankCode            *string `json:"bankCode"`
		ApiClientId         *string `json:"api_client_id"`
		ApiRrn              *string `json:"api_rrn"`
		ApiAccountId        *string `json:"api_account_id"`
		SenderName          *string `json:"sender_name"`
		BankName            *string `json:"bankName"`
		SessionId           *string `json:"sessionId"`
		UserName            *string `json:"userName"`
		AccountNumber       *string `json:"accountNumber"`
		Rrn                 *string `json:"rrn"`
		HooksEligible       *string `json:"hooksEligible"`
		MerchantTxRef       *string `json:"merchantTxRef"`
		BankingEntityType   *string `json:"banking_entity_type"`
		UserId              *string `json:"user_id"`
		IsCorporate         *string `json:"isCorporate"`
		Narration           *string `json:"narration"`
		TransactionCategory *string `json:"transactionCategory"`
		RecipientName       *string `json:"recipientName"`
		Currency            *string `json:"currency"`
	} `json:"meta"`
	UserId      *string `json:"userId"`
	TimeCreated *string `json:"timeCreated"`
}]

type VerifyCheckoutOrderResponse = Response[struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Order   struct {
		OrderID        string  `json:"orderId"`
		OrderReference string  `json:"orderReference"`
		Amount         float64 `json:"amount,string"`
		Currency       string  `json:"currency"`
		CustomerEmail  string  `json:"customerEmail"`
	} `json:"order"`
	TransactionDetails struct {
		TransactionDate      string `json:"transactionDate"`
		PaymentReference     string `json:"paymentReference"`
		StatusCode           string `json:"statusCode"`
		TokenizedCardPayment string `json:"tokenizedCardPayment"`
	} `json:"transactionDetails"`
	CardDetails struct {
		CardPan      string `json:"cardPan"`
		CardType     string `json:"cardType"`
		CardCurrency string `json:"cardCurrency"`
	} `json:"cardDetails"`
}]
