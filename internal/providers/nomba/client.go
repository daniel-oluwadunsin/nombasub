package nomba

import (
	"fmt"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"resty.dev/v3"
)

const nombaBaseUrl = "https://api.nomba.io/v1"

type Client struct {
	BaseURL                 string
	HTTPClient              *resty.Client
	ClientID                string
	ClientSecret            string
	AccountID               string
	AccessToken             *string
	RefreshToken            *string
	AccessTokenExpiresAt    *time.Time
	TenantConnectionService TenantConnectionService
}

func NewClient(
	clientID,
	clientSecret,
	accountID string,
	accessToken,
	refreshToken *string,
	accessTokenExpiresAt *time.Time,
	tenantConnectionService TenantConnectionService,
) (*Client, error) {
	client := &Client{
		BaseURL:                 nombaBaseUrl,
		HTTPClient:              resty.New().SetBaseURL(nombaBaseUrl),
		ClientID:                clientID,
		ClientSecret:            clientSecret,
		AccountID:               accountID,
		AccessToken:             accessToken,
		RefreshToken:            refreshToken,
		AccessTokenExpiresAt:    accessTokenExpiresAt,
		TenantConnectionService: tenantConnectionService,
	}
	err := client.setNewHTTPClient()
	if err != nil {
		return nil, err
	}
	return client, nil
}

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
		SetResultError(&Response[any]{}).
		SetResult(&GetAccessTokenResponse{}).
		Post(url)

	if err != nil {
		return nil, err
	}

	if res.IsStatusFailure() {
		err := res.ResultError().(*Response[any])
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

		err = c.TenantConnectionService.SaveTenantNombaConnection(c.AccountID, &TenantConnection{
			ClientID:             c.ClientID,
			ClientSecret:         c.ClientSecret,
			AccountID:            c.AccountID,
			AccessToken:          c.AccessToken,
			RefreshToken:         c.RefreshToken,
			AccessTokenExpiresAt: c.AccessTokenExpiresAt,
		})
		if err != nil {
			return nil, err
		}

		return c, nil
	}

	return c, nil
}

func (c *Client) setNewHTTPClient() error {
	var err error
	if c.AccessToken == nil || c.AccessTokenExpiresAt == nil {
		c, err = c.issueAccessToken(true)
		if err != nil {
			return FailedToIssueAccessTokenForTenant
		}
	} else {
		if c.AccessTokenExpiresAt.Before(time.Now()) {
			c, err = c.issueAccessToken(true)
			if err != nil {
				return FailedToRefreshAccessTokenForTenant
			}
		}
	}

	c.HTTPClient = resty.New().
		SetBaseURL(nombaBaseUrl).
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", *c.AccessToken))

	return nil
}
