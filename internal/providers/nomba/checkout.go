package nomba

import (
	"fmt"

	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"resty.dev/v3"
)

func (c *Client) CreateCheckoutOrder(body CreateCheckoutOrderRequest) (*CreateCheckoutOrderResponse, error) {
	res, err := c.authenticatedRequest(func() *resty.Request {
		return c.HTTPClient.R().
			SetBody(body).
			SetResultError(&errorResponse{}).
			SetResult(&CreateCheckoutOrderResponse{})
	}, resty.MethodPost, "/v1/checkout/order")

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if res.IsStatusFailure() {
		fmt.Println("Error response from Nomba:", res.String())
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
