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
