package mcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

func AuthAndRateLimit(next http.Handler, limiter *RateLimiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := extractAPIKey(r)
		if key == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "missing X-Api-Key header — configure your MCP client with the merchant API key")
			return
		}
		if !limiter.Allow(key) {
			w.Header().Set("Retry-After", "10")
			writeJSONError(w, http.StatusTooManyRequests, "rate_limited", "too many requests for this API key; slow down and retry")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractAPIKey(r *http.Request) string {
	if k := r.Header.Get("X-Api-Key"); k != "" {
		return k
	}
	if auth := r.Header.Get("Authorization"); auth != "" {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	return ""
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
