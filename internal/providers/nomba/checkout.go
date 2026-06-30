package nomba

import "github.com/daniel-oluwadunsin/nombasub/internal/responses"

func (c *Client) CreateCheckoutOrder(body CreateCheckoutOrderRequest) (*CreateCheckoutOrderResponse, error) {
	res, err := c.HTTPClient.R().
		SetHeader("accountId", c.AccountID).
		SetHeader("Authorization", "Bearer "+*c.AccessToken).
		SetBody(body).
		SetResultError(&errorResponse{}).
		SetResult(&CreateCheckoutOrderResponse{}).
		Post("/checkout/order")

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if res.IsStatusFailure() {
		err := res.ResultError().(*errorResponse)
		return nil, &responses.AppError{
			StatusCode: res.StatusCode(),
			Message:    err.Description,
			Data:       err.Data,
		}
	}

	result := res.Result().(*CreateCheckoutOrderResponse)
	return result, nil
}
