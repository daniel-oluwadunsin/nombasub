package nomba

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"resty.dev/v3"
)

func (c *Client) TransferToNombaAccount(body TransferToAccountRequest) (*TransferToAccountResponse, error) {
	res, err := c.authenticatedRequest(func() *resty.Request {
		return c.HTTPClient.R().
			SetBody(body).
			SetResultError(&errorResponse{}).
			SetResult(&TransferToAccountResponse{})
	}, resty.MethodPost, "/v2/transfers/wallet")

	if err != nil {
		return nil, err
	}

	if res.IsStatusFailure() {
		err := res.ResultError().(*errorResponse)
		return nil, &responses.AppError{
			StatusCode: res.StatusCode(),
			Message:    err.Description,
			Data:       err.Data,
		}
	}

	result := res.Result().(*TransferToAccountResponse)
	return result, nil
}
