package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRateLimiter_AllowPerKey(t *testing.T) {

	rl := NewRateLimiter(1, 1) // 1 token, 1 token/second

	require.True(t, rl.Allow("client-a"), "first token should be allowed")
	require.True(t, rl.Allow("client-b"), "different key should also be allowed")
	require.False(t, rl.Allow("client-a"), "second token should be denied for same key")
}

func TestRateLimiter_BlocksAfterLimit(t *testing.T) {

	rl := NewRateLimiter(1, 1) // 1 token, 1 token/second

	h := RateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusTooManyRequests, rr.Code)
	require.Equal(t, "1", rr.Header().Get("Retry-After"))
}

func TestRateLimitMiddleware_UsesXForwardedFor(t *testing.T) {
	rl := NewRateLimiter(1, 1)

	h := RateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.RemoteAddr = "9.9.9.9:1111"
	req.Header.Set("X-Forwarded-For", "1.1.1.1, 2.2.2.2")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	rr = httptest.NewRecorder()
	req.RemoteAddr = "8.8.8.8:2222"
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusTooManyRequests, rr.Code)

	rr = httptest.NewRecorder()
	req.Header.Del("X-Forwarded-For")
	req.RemoteAddr = "3.3.3.3:3333"
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}
