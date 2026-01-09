package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/rafaelmgr12/streamgate/internal/middleware/utils"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	mu         sync.Mutex
	buckets    map[string]*utils.TokenBucket
	capacity   int
	refillRate float64
}

// NewRateLimiter creates a new RateLimiter
func NewRateLimiter(capacity int, refillRate float64) *RateLimiter {
	return &RateLimiter{
		buckets:    make(map[string]*utils.TokenBucket),
		capacity:   capacity,
		refillRate: refillRate,
	}
}

// Allow checks if a request with the given key is allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket := rl.buckets[key]
	if bucket == nil {
		bucket = utils.NewTokenBucket(rl.capacity, rl.refillRate)
		rl.buckets[key] = bucket
	}

	return bucket.Allow(1)

}

// RateLimitMiddleware returns a middleware that enforces rate limiting
// based on the provided RateLimiter.
func RateLimitMiddleware(limiter *RateLimiter) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := clientKey(r)
			if !limiter.Allow(key) {
				w.Header().Set("Retry-After", "1")
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientKey extracts the client key from the request for rate limiting purposes.
func clientKey(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
