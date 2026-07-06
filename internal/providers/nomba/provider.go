package nomba

type Provider interface {
	CreateCheckoutOrder(CreateCheckoutOrderRequest) (*CreateCheckoutOrderResponse, error)
	TransferToNombaAccount(TransferToAccountRequest) (*TransferToAccountResponse, error)
	GenerateSignature(payload, timestamp string) (string, error)
	DeductFee(amount float64) float64
	CalculateFee(amount float64) float64
	ChargeCard(ChargeCardRequest) (*ChargeCardResponse, error)
	CreateDirectDebitManadate(CreateDirectDebitManadateRequest) (*CreateDirectDebitManadateResponse, error)
	GetDirectDebitManadateStatus(mandateId string) (*GetDirectDebitManadateResponse, error)
	UpdateDirectDebitStatus(UpdateDirectDebitManadateRequest) (*UpdateDirectDebitStatusResponse, error)
	DebitMandate(DebitMandateRequest) (*DebitMandateResponse, error)
	RequestRefund(RefundRequest) error
}
