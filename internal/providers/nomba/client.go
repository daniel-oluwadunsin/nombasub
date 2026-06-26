package nomba

import (
	"fmt"
	"net/http"
	"sync"
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
	tokenMu              sync.Mutex
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
	c.HTTPClient = resty.New().
		SetBaseURL(nombaBaseUrl).
		AddRequestMiddleware(func(_ *resty.Client, r *resty.Request) error {
			accessToken, err := c.ensureAccessToken()
			if err != nil {
				return err
			}

			r.SetHeader("accountId", c.AccountID)
			r.SetHeader("Authorization", fmt.Sprintf("Bearer %s", accessToken))
			return nil
		})

	return nil
}

func (c *Client) authenticatedRequest(build func() *resty.Request, method string, url string) (*resty.Response, error) {
	res, err := build().Execute(method, url)
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != http.StatusUnauthorized {
		return res, nil
	}

	if _, err := c.refreshAccessToken(); err != nil {
		return nil, err
	}

	return build().Execute(method, url)
}
