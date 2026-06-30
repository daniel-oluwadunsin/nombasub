package nomba

type Provider interface {
	CreateCheckoutOrder(CreateCheckoutOrderRequest) (*CreateCheckoutOrderResponse, error)
}
