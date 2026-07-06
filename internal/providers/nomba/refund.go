package nomba

import (
	"errors"

	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"resty.dev/v3"
)

func (c *Client) RequestRefund(body RefundRequest) error {
	res, err := c.authenticatedRequest(func() *resty.Request {
		return c.HTTPClient.R().
			SetBody(body).
			SetResultError(&errorResponse{})
	}, resty.MethodPost, "/v1/checkout/refund")

	if err != nil {
		return responses.InternalServerError(err)
	}

	if res.IsStatusFailure() {
		err := res.ResultError().(*errorResponse)
		return &responses.AppError{
			StatusCode: res.StatusCode(),
			Message:    err.Description,
			Data:       err.Data,
			Err:        errors.New(err.Description),
		}
	}

	return nil
}
