package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type EngineClient struct {
	baseURL string
	http    *http.Client
}

func NewEngineClient(baseURL string) *EngineClient {
	return &EngineClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

type envelope struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *EngineClient) Get(ctx context.Context, path string, query map[string]string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, path, query, nil)
}

func (c *EngineClient) Post(ctx context.Context, path string, query map[string]string, body any) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPost, path, query, body)
}

func (c *EngineClient) Put(ctx context.Context, path string, query map[string]string, body any) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPut, path, query, body)
}

func (c *EngineClient) do(ctx context.Context, method, path string, query map[string]string, body any) (json.RawMessage, error) {
	apiKey := APIKey(ctx)
	if apiKey == "" {
		return nil, fmt.Errorf("missing X-Api-Key: configure your MCP client with an API key header")
	}

	target, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("invalid engine URL: %w", err)
	}
	if len(query) > 0 {
		q := target.Query()
		for k, v := range query {
			if v != "" {
				q.Set(k, v)
			}
		}
		target.RawQuery = q.Encode()
	}

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, target.String(), reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-Api-Key", apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("engine request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read engine response: %w", err)
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("engine returned non-JSON (%d): %s", resp.StatusCode, string(raw))
	}

	if resp.StatusCode >= 400 || env.Status == "error" {
		msg := env.Message
		if msg == "" {
			msg = fmt.Sprintf("engine returned %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("%s", msg)
	}

	return env.Data, nil
}
