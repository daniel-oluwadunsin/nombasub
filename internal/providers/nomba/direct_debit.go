package nomba

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"resty.dev/v3"
)

func (c *Client) CreateDirectDebitManadate(body CreateDirectDebitManadateRequest) (*CreateDirectDebitManadateResponse, error) {
	res, err := c.authenticatedRequest(func() *resty.Request {
		return c.HTTPClient.R().SetBody(body).SetResultError(errorResponse{}).SetResult(&CreateDirectDebitManadateResponse{})
	}, resty.MethodPost, "/v1/direct-debits")

	if err != nil {
		return nil, err
	}
	if res.IsStatusFailure() {
		err := res.Result().(*errorResponse)
		return nil, &responses.AppError{
			StatusCode: res.StatusCode(),
			Message:    err.Description,
			Data:       err.Data,
		}
	}

	result := res.Result().(*CreateDirectDebitManadateResponse)
	return result, nil
}
