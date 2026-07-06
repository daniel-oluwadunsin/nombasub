package mcp

import (
	"context"
	"net/http"
)

type ctxKey string

const apiKeyCtxKey ctxKey = "nombasub-api-key"

func WithAPIKeyFromHTTP(ctx context.Context, r *http.Request) context.Context {
	if key := r.Header.Get("X-Api-Key"); key != "" {
		return context.WithValue(ctx, apiKeyCtxKey, key)
	}
	if key := r.Header.Get("Authorization"); key != "" {
		return context.WithValue(ctx, apiKeyCtxKey, key)
	}
	return ctx
}

func APIKey(ctx context.Context) string {
	if v, ok := ctx.Value(apiKeyCtxKey).(string); ok {
		return v
	}
	return ""
}
