package responses

type InitializeDirectDebitResponse struct {
	MandateID           string `json:"mandateId"`
	MerchantReference   string `json:"merchantReference"`
	CustomerPhoneNumber string `json:"customerPhoneNumber"`
	Description         string `json:"description"`
}
