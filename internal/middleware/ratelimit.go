package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/gin-gonic/gin"
)

// rateLimiter is a per-key token-bucket limiter used to throttle abuse-prone
// endpoints (login, portal code issue/verify) by client IP. Buckets refill
// continuously and idle buckets are evicted by a background janitor so memory
// does not grow unbounded with the number of distinct callers.
type rateLimiter struct {
	mu         sync.Mutex
	buckets    map[string]*tokenBucket
	max        float64
	windowSecs float64
}

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

func newRateLimiter(requestsPerMinute int) *rateLimiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 60
	}
	rl := &rateLimiter{
		buckets:    map[string]*tokenBucket{},
		max:        float64(requestsPerMinute),
		windowSecs: 60,
	}
	go rl.janitor()
	return rl
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		rl.buckets[key] = &tokenBucket{tokens: rl.max - 1, lastRefill: now}
		return true
	}

	refillRate := rl.max / rl.windowSecs
	b.tokens += now.Sub(b.lastRefill).Seconds() * refillRate
	if b.tokens > rl.max {
		b.tokens = rl.max
	}
	b.lastRefill = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (rl *rateLimiter) janitor() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for key, b := range rl.buckets {
			if time.Since(b.lastRefill) > 10*time.Minute {
				delete(rl.buckets, key)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitBy returns middleware that allows at most requestsPerMinute requests
// per key, where the key is derived from the request by keyFn. Exceeding the
// limit responds with 429 and a Retry-After hint.
func RateLimitBy(requestsPerMinute int, keyFn func(*gin.Context) string) gin.HandlerFunc {
	rl := newRateLimiter(requestsPerMinute)
	return func(c *gin.Context) {
		if !rl.allow(keyFn(c)) {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, responses.Format(&responses.Response{
				Success: false,
				Message: "Too many requests; please slow down and retry",
			}))
			return
		}
		c.Next()
	}
}

// RateLimit throttles per client IP.
func RateLimit(requestsPerMinute int) gin.HandlerFunc {
	return RateLimitBy(requestsPerMinute, func(c *gin.Context) string {
		return c.ClientIP()
	})
}

// RateLimitByTenant throttles per authenticated tenant, falling back to client
// IP before authentication runs. Use on authenticated routes so a single busy
// merchant IP is not throttled by another tenant's traffic.
func RateLimitByTenant(requestsPerMinute int) gin.HandlerFunc {
	return RateLimitBy(requestsPerMinute, func(c *gin.Context) string {
		if tenantID := c.GetString(TenantIdCtxKey); tenantID != "" {
			return "tenant:" + tenantID
		}
		return "ip:" + c.ClientIP()
	})
}
