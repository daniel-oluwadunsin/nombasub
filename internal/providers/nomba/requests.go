package nomba

type CreateCheckoutOrderRequest struct {
	Order struct {
		CallbackUrl           string           `json:"callbackUrl" binding:"required"`
		CustomerEmail         string           `json:"customerEmail" binding:"required"`
		Amount                int64            `json:"amount" binding:"required"`
		Currency              *string          `json:"currency" binding:"oneof=NGN"`
		OrderReference        *string          `json:"orderReference"`
		CustomerId            *string          `json:"customerId"`
		AccountId             *string          `json:"accountId"`
		AllowedPaymentMethods *[]PaymentMethod `json:"allowedPaymentMethods"`
		OrderMetaData         *interface{}     `json:"orderMetaData"`
		SplitRequest          *struct {
			SplitList []struct {
				AccountId string `json:"accountId"`
				Value     string `json:"value"`
			} `json:"splitList"`
		} `json:"splitRequest"`
	} `json:"order"`
	TokenizeCard *bool `json:"tokenizeCard"`
}
