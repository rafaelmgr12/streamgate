package utils

import (
	"sync"
	"time"
)

// TokenBucket implements a simple token bucket algorithm for rate limiting.
type TokenBucket struct {
	mu         sync.Mutex
	capacity   float64
	tokens     float64
	refillRate float64
	lastRefill time.Time
}

// NewTokenBucket creates a new TokenBucket with the given capacity and refill rate.
func NewTokenBucket(capacity int, refillRate float64) *TokenBucket {
	if capacity <= 0 {
		capacity = 1
	}

	if refillRate <= 0 {
		refillRate = 1.0
	}

	now := time.Now()
	return &TokenBucket{
		capacity:   float64(capacity),
		tokens:     float64(capacity),
		refillRate: refillRate,
		lastRefill: now,
	}
}

// Allow checks if n tokens can be consumed from the bucket.
func (b *TokenBucket) Allow(n int) bool {
	if n <= 0 {
		return true
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill(time.Now())

	if b.tokens < float64(n) {
		return false
	}

	b.tokens -= float64(n)
	return true
}

// refill adds tokens to the bucket based on the elapsed time since the last refill.
func (b *TokenBucket) refill(now time.Time) {
	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed <= 0 {
		return
	}

	b.tokens += elapsed * b.refillRate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.lastRefill = now
}
