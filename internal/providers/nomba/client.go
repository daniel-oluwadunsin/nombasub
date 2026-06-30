package nomba

import (
	"fmt"
	"net/http"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/config"
	"resty.dev/v3"
)

const nombaBaseUrl = "https://sandbox.api.nomba.com/v1"

type Client struct {
	BaseURL              string
	HTTPClient           *resty.Client
	ClientID             string
	ClientSecret         string
	AccountID            string
	AccessToken          *string
	RefreshToken         *string
	AccessTokenExpiresAt *time.Time
}

func New(env *config.Config) (*Client, error) {
	client := &Client{
		BaseURL:      nombaBaseUrl,
		HTTPClient:   resty.New().SetBaseURL(nombaBaseUrl),
		ClientID:     env.NombaClientID,
		ClientSecret: env.NombaClientSecret,
		AccountID:    env.NombaAccountID,
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
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", *c.AccessToken)).
		SetRetryCount(1).
		AddRetryConditions(func(r *resty.Response, err error) bool {
			if r == nil {
				return false
			}
			return false
		}).
		AddRetryHooks(func(r *resty.Response, err error) {
			if r != nil {
				if r.StatusCode() == http.StatusUnauthorized {
					_, err := c.issueAccessToken(true)
					if err != nil {
						return
					}
					r.Request.SetHeader("Authorization", fmt.Sprintf("Bearer %s", *c.AccessToken))
				}
			}
		})

	return nil
}
