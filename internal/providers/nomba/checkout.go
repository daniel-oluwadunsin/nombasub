package nomba

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"resty.dev/v3"
)

func (c *Client) CreateCheckoutOrder(body CreateCheckoutOrderRequest) (*CreateCheckoutOrderResponse, error) {
	res, err := c.authenticatedRequest(func() *resty.Request {
		return c.HTTPClient.R().
			SetBody(body).
			SetResultError(&errorResponse{}).
			SetResult(&CreateCheckoutOrderResponse{})
	}, resty.MethodPost, "/checkout/order")

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
