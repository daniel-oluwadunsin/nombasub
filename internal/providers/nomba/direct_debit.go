package nomba

import (
	"errors"

	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"resty.dev/v3"
)

func (c *Client) CreateDirectDebitManadate(body CreateDirectDebitManadateRequest) (*CreateDirectDebitManadateResponse, error) {
	res, err := c.authenticatedRequest(func() *resty.Request {
		return c.HTTPClient.R().SetBody(body).SetResultError(&errorResponse{}).SetResult(&CreateDirectDebitManadateResponse{})
	}, resty.MethodPost, "/v1/direct-debits")

	if err != nil {
		return nil, err
	}
	if res.IsStatusFailure() {
		err := res.ResultError().(*errorResponse)
		return nil, &responses.AppError{
			StatusCode: res.StatusCode(),
			Message:    err.Description,
			Data:       err.Data,
			Err:        errors.New(err.Description),
		}
	}

	result := res.Result().(*CreateDirectDebitManadateResponse)
	return result, nil
}

func (c *Client) UpdateDirectDebitStatus(body UpdateDirectDebitManadateRequest) (*UpdateDirectDebitStatusResponse, error) {
	res, err := c.authenticatedRequest(func() *resty.Request {
		return c.HTTPClient.R().SetBody(body).SetResultError(&errorResponse{}).SetResult(&UpdateDirectDebitStatusResponse{})
	}, resty.MethodPut, "/v1/direct-debits/update-status")

	if err != nil {
		return nil, err
	}
	if res.IsStatusFailure() {
		err := res.ResultError().(*errorResponse)
		return nil, &responses.AppError{
			StatusCode: res.StatusCode(),
			Message:    err.Description,
			Data:       err.Data,
			Err:        errors.New(err.Description),
		}
	}

	result := res.Result().(*UpdateDirectDebitStatusResponse)
	return result, nil
}

func (c *Client) DebitMandate(body DebitMandateRequest) (*DebitMandateResponse, error) {
	res, err := c.authenticatedRequest(func() *resty.Request {
		return c.HTTPClient.R().SetBody(body).SetResultError(&errorResponse{}).SetResult(&DebitMandateResponse{})
	}, resty.MethodPost, "/v1/direct-debits/debit-mandate")

	if err != nil {
		return nil, err
	}
	if res.IsStatusFailure() {
		errBody := res.ResultError().(*errorResponse)
		return nil, &responses.AppError{
			StatusCode: res.StatusCode(),
			Message:    errBody.Description,
			Data:       errBody.Data,
			Err:        errors.New(errBody.Description),
		}
	}

	result := res.Result().(*DebitMandateResponse)
	return result, nil
}

func (c *Client) GetDirectDebitManadateStatus(mandateId string) (*GetDirectDebitManadateResponse, error) {
	res, err := c.authenticatedRequest(func() *resty.Request {
		return c.HTTPClient.R().SetPathParams(map[string]string{
			"mandateId": mandateId,
		}).SetResultError(&errorResponse{}).SetResult(&GetDirectDebitManadateResponse{})
	}, resty.MethodGet, "/v1/direct-debits/status?mandateId={mandateId}")

	if err != nil {
		return nil, err
	}
	if res.IsStatusFailure() {
		err := res.ResultError().(*errorResponse)
		return nil, &responses.AppError{
			StatusCode: res.StatusCode(),
			Message:    err.Description,
			Data:       err.Data,
			Err:        errors.New(err.Description),
		}
	}

	result := res.Result().(*GetDirectDebitManadateResponse)
	return result, nil
}
