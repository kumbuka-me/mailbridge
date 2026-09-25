package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/containeroo/mailbridge/internal/response"
)

// RateLimit limits requests to the configured number per second.
// A nonpositive limit disables rate limiting.
func RateLimit(requestsPerSecond int) Middleware {
	if requestsPerSecond <= 0 {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	limiter := newTokenBucket(requestsPerSecond)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.allow(time.Now()) {
				w.Header().Set("Retry-After", "1")
				response.Problem(w, http.StatusTooManyRequests, "Too many requests.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// tokenBucket implements a concurrency-safe token bucket rate limiter.
type tokenBucket struct {
	mu       sync.Mutex
	rate     float64
	capacity float64
	tokens   float64
	last     time.Time
}

// newTokenBucket creates a full token bucket for the configured requests-per-second rate.
func newTokenBucket(requestsPerSecond int) *tokenBucket {
	rate := float64(requestsPerSecond)
	return &tokenBucket{
		rate:     rate,
		capacity: rate,
		tokens:   rate,
		last:     time.Now(),
	}
}

// allow reports whether one request may consume a token at now.
func (b *tokenBucket) allow(now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens = min(b.capacity, b.tokens+elapsed*b.rate)
		b.last = now
	}
	if b.tokens < 1 {
		return false
	}

	b.tokens--
	return true
}
