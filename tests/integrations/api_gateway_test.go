package integrations

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rafaelmgr12/streamgate/internal/balancer"
	"github.com/rafaelmgr12/streamgate/internal/config"
	"github.com/rafaelmgr12/streamgate/internal/handler"
	"github.com/stretchr/testify/require"
)

func TestAPIGatewayIntegration(t *testing.T) {
	h := handler.NewHTTPHandler()
	server := httptest.NewServer(h)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("could not send GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read response body: %v", err)
	}

	expected := "Hello from API Gateway"
	if string(body) != expected {
		t.Errorf("expected response body to be %q; got %q", expected, string(body))
	}
}

func TestIntegration_BackendFailover(t *testing.T) {
	var badHits int32
	var goodHits int32

	badBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&badHits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer badBackend.Close()

	goodBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&goodHits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer goodBackend.Close()

	cfg := &config.Config{
		Services: []config.ServiceConfig{
			{
				Name:       "test",
				PathPrefix: "/api/test/",
				Backends:   []string{badBackend.URL, goodBackend.URL},
			},
		},
	}

	tracker := balancer.NewHealthTracker(1, time.Minute)
	gateway := httptest.NewServer(handler.NewHTTPHandlerWithTracker(cfg, tracker))
	defer gateway.Close()

	resp1, err := http.Get(gateway.URL + "/api/test/one")
	require.NoError(t, err)
	resp1.Body.Close()
	require.Equal(t, http.StatusInternalServerError, resp1.StatusCode)

	resp2, err := http.Get(gateway.URL + "/api/test/two")
	require.NoError(t, err)
	resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	require.Equal(t, int32(1), badHits)
	require.Equal(t, int32(1), goodHits)
}

func TestIntegration_HealthCheck(t *testing.T) {
	var status int32 = http.StatusServiceUnavailable

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(atomic.LoadInt32(&status)))
	}))
	defer backend.Close()

	u, err := url.Parse(backend.URL)
	require.NoError(t, err)

	tracker := balancer.NewHealthTracker(1, time.Minute)
	checker := balancer.NewActiveHealthChecker(tracker, []*url.URL{u}, 50*time.Millisecond, 50*time.Millisecond, "/healthz")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	checker.Start(ctx)

	require.Eventually(t, func() bool {
		return !tracker.IsHealthy(u)
	}, 500*time.Millisecond, 20*time.Millisecond)

	atomic.StoreInt32(&status, http.StatusOK)

	require.Eventually(t, func() bool {
		return tracker.IsHealthy(u)
	}, 500*time.Millisecond, 20*time.Millisecond)
}
