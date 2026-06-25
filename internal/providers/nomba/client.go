package nomba

import (
	"fmt"
	"time"

	"resty.dev/v3"
)

const nombaBaseUrl = "https://sandbox.api.nomba.com/v1"

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
