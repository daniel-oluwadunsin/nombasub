package nomba

import (
	"fmt"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
)

func (c *Client) issueAccessToken(isRefreshing bool) (*Client, error) {
	client := c.HTTPClient.R()
	var url string

	if !isRefreshing {
		url = "/auth/token/issue"
		client = client.
			SetHeader("accountId", c.AccountID).
			SetBody(map[string]any{
				"grant_type":    "client_credentials",
				"client_id":     c.ClientID,
				"client_secret": c.ClientSecret,
			})
	} else {
		url = "/auth/token/refresh"
		client = client.
			SetHeader("accountId", c.AccountID).
			SetHeader("Authorization", fmt.Sprintf("Bearer %s", *c.AccessToken)).
			SetBody(map[string]any{
				"grant_type":    "refresh_token",
				"refresh_token": *c.RefreshToken,
			})
	}

	res, err := client.
		SetResultError(&errorResponse{}).
		SetResult(&GetAccessTokenResponse{}).
		Post(url)

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

	if res.IsStatusSuccess() {
		result := res.Result().(*GetAccessTokenResponse).Data
		c.AccessToken = &result.AccessToken
		c.RefreshToken = &result.RefreshToken

		expiresAt, err := time.Parse(time.RFC3339, result.ExpiresAt)
		if err != nil {
			return nil, err
		}
		c.AccessTokenExpiresAt = &expiresAt

		return c, nil
	}

	return c, nil
}
