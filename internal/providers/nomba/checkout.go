package nomba

import (
	"errors"
	"net/http"

	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"resty.dev/v3"
)

func (c *Client) CreateCheckoutOrder(body CreateCheckoutOrderRequest) (*CreateCheckoutOrderResponse, error) {
	body.Order.AccountId = &c.SubAccountID
	body.Order.Amount = utils.ToPtr(*body.Order.Amount / 100)

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
		err := res.ResultError().(*errorResponse)
		return nil, &responses.AppError{
			StatusCode: res.StatusCode(),
			Message:    err.Description,
			Data:       err.Data,
			Err:        errors.New(err.Description),
		}
	}

	result := res.Result().(*CreateCheckoutOrderResponse)
	if result.Data.CheckoutLink == "" || result.Data.OrderReference == "" {
		return nil, responses.NewAppError(http.StatusBadGateway, utils.OrStrings(result.Description, "Nomba returned an empty checkout response"))
	}

	return result, nil
}

func (c *Client) ChargeCard(body ChargeCardRequest) (*ChargeCardResponse, error) {
	body.Order.Amount = utils.ToPtr(*body.Order.Amount / 100)

	res, err := c.authenticatedRequest(func() *resty.Request {
		return c.HTTPClient.R().
			SetHeader("accountId", c.SubAccountID).
			SetBody(body).
			SetResultError(&errorResponse{}).
			SetResult(&ChargeCardResponse{})
	}, resty.MethodPost, "/v1/checkout/tokenized-card-payment")

	if err != nil {
		return nil, responses.InternalServerError(err)
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

	result := res.Result().(*ChargeCardResponse)

	return result, nil
}
