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

func (c *Client) GetDirectDebitManadateStatus(mandateId string) (*GetDirectDebitManadateResponse, error) {
	res, err := c.authenticatedRequest(func() *resty.Request {
		return c.HTTPClient.R().SetPathParams(map[string]string{
			"mandateId": mandateId,
		}).SetResultError(errorResponse{}).SetResult(&GetDirectDebitManadateResponse{})
	}, resty.MethodGet, "/v1/direct-debits/status?mandateId={mandateId}")

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

	result := res.Result().(*GetDirectDebitManadateResponse)
	return result, nil
}

func (c *Client) DebitMandate(body DebitMandateRequest) (*DebitMandateResponse, error) {
	res, err := c.authenticatedRequest(func() *resty.Request {
		return c.HTTPClient.R().SetBody(body).SetResultError(errorResponse{}).SetResult(&CreateDirectDebitManadateResponse{})
	}, resty.MethodPost, "/v1/direct-debits/debit-mandate")

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

	result := res.Result().(*DebitMandateResponse)
	return result, nil
}

func (c *Client) GetMandate(mandateId string) (*GetMandateResponse, error) {
	res, err := c.authenticatedRequest(func() *resty.Request {
		return c.HTTPClient.R().SetPathParams(map[string]string{
			"mandateId": mandateId,
		}).SetResultError(errorResponse{}).SetResult(&GetDirectDebitManadateResponse{})
	}, resty.MethodGet, "/v1/direct-debits/{mandateId}")

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

	result := res.Result().(*GetMandateResponse)
	return result, nil
}

func (c *Client) UpdateDirectDebitStatus(mandateId string, status string) error {}
