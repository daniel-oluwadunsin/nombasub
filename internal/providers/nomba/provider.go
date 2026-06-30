package nomba

type Provider interface {
	CreateCheckoutOrder(CreateCheckoutOrderRequest) (*CreateCheckoutOrderResponse, error)
	TransferToNombaAccount(TransferToAccountRequest) (*TransferToAccountResponse, error)
	GenerateSignature(payload, timestamp string) (string, error)
}
