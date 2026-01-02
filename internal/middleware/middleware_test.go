package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	require.NoError(t, w.Close())
	os.Stdout = original

	out, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())

	return string(out)
}

func TestChain_Order(t *testing.T) {
	trace := make([]string, 0, 5)

	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			trace = append(trace, "m1-before")
			next.ServeHTTP(w, r)
			trace = append(trace, "m1-after")
		})
	}

	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			trace = append(trace, "m2-before")
			next.ServeHTTP(w, r)
			trace = append(trace, "m2-after")
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trace = append(trace, "handler")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	Chain(handler, m1, m2).ServeHTTP(rr, req)

	expected := []string{"m1-before", "m2-before", "handler", "m2-after", "m1-after"}
	assert.Equal(t, expected, trace)
}

func TestRequestID_PreservesExisting(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := GetRequestID(r.Context())
		assert.True(t, ok)
		assert.Equal(t, "req-123", id)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "req-123")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "req-123", rr.Header().Get("X-Request-ID"))
}

func TestRequestID_GeneratesID(t *testing.T) {
	var seen string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := GetRequestID(r.Context())
		assert.True(t, ok)
		seen = id
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	id := rr.Header().Get("X-Request-ID")
	assert.NotEmpty(t, id)
	assert.Equal(t, seen, id)
	assert.Regexp(t, regexp.MustCompile("^[0-9a-f]{32}$"), id)
}

func TestGetRequestID_NoValue(t *testing.T) {
	id, ok := GetRequestID(context.Background())
	assert.False(t, ok)
	assert.Empty(t, id)
}

func TestLogging_LogsRequestIDAndStatus(t *testing.T) {
	output := captureStdout(t, func() {
		handler := RequestID(Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("ok"))
		})))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "req-logging")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		assert.Equal(t, "ok", rr.Body.String())
	})

	assert.Contains(t, output, "\"request_id\":\"req-logging\"")
	assert.Contains(t, output, "\"status\":201")
}
