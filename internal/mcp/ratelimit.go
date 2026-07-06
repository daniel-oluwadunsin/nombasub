package mcp

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu         sync.Mutex
	buckets    map[string]*bucket
	max        int
	windowSecs int64
}

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 60
	}
	return &RateLimiter{
		buckets:    map[string]*bucket{},
		max:        requestsPerMinute,
		windowSecs: 60,
	}
}

func (r *RateLimiter) Allow(key string) bool {
	if key == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	b, ok := r.buckets[key]
	now := time.Now()
	if !ok {
		r.buckets[key] = &bucket{tokens: float64(r.max) - 1, lastRefill: now}
		return true
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	refillRate := float64(r.max) / float64(r.windowSecs)
	b.tokens += elapsed * refillRate
	if b.tokens > float64(r.max) {
		b.tokens = float64(r.max)
	}
	b.lastRefill = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
