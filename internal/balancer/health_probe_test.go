package balancer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

func TestActiveHealthChecker_CheckOnce_MarksFailureAndSuccess(t *testing.T) {
	var status int32 = http.StatusServiceUnavailable
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Fatalf("expected health path /healthz, got %q", r.URL.Path)
		}
		w.WriteHeader(int(atomic.LoadInt32(&status)))
	}))
	defer server.Close()

	target := mustParseURL(t, server.URL)
	tracker := NewHealthTracker(1, time.Minute)

	checker := NewActiveHealthChecker(tracker, []*url.URL{target}, time.Second, time.Second, "/healthz")
	checker.client = server.Client()

	checker.checkOnce(context.Background())
	if tracker.IsHealthy(target) {
		t.Fatalf("expected unhealthy after failure probe")
	}

	atomic.StoreInt32(&status, http.StatusOK)
	checker.checkOnce(context.Background())
	if !tracker.IsHealthy(target) {
		t.Fatalf("expected healthy after success probe")
	}
}
