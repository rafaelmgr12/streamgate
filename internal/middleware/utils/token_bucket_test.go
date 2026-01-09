package utils

import (
	"testing"
	"time"
)

func TestTokenBucket_AlloweAndRefill(t *testing.T) {

	b := NewTokenBucket(2, 2) // 2 tokens, 2 tokens/second

	if !b.Allow(1) {
		t.Fatalf("expected first token be allowed")
	}

	if !b.Allow(1) {
		t.Fatalf("expected second token be allowed")
	}
	if b.Allow(1) {
		t.Fatalf("expected third token to be denied (empty bucket)")
	}

	now := time.Now()
	b.mu.Lock()
	b.lastRefill = now.Add(-600 * time.Millisecond) // ~1.2 tokens should be refilled
	b.mu.Unlock()
	b.refill(now)

	if !b.Allow(1) {
		t.Fatalf("expected token to be allowed after refill")
	}
	if b.Allow(1) {
		t.Fatalf("expected no second token after partial refill")
	}

}

func TestTokenBucket_NoOverfill(t *testing.T) {
	b := NewTokenBucket(2, 100) // fast refill rate
	now := time.Now()

	b.mu.Lock()
	b.lastRefill = now.Add(-10 * time.Second)
	b.mu.Unlock()
	b.refill(now)

	if b.Allow(3) {
		t.Fatalf("expected bucket to cap at capacity and deny 3 tokens")
	}
}

func TestTokenBucket_AllowZeroDoesNotConsume(t *testing.T) {
	b := NewTokenBucket(1, 1)

	if !b.Allow(0) {
		t.Fatalf("expected Allow(0) to return true")
	}
	if !b.Allow(1) {
		t.Fatalf("expected token to still be available after Allow(0)")
	}
	if b.Allow(1) {
		t.Fatalf("expected bucket to be empty after consuming 1 token")
	}
}
