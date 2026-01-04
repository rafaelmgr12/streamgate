package handler_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/rafaelmgr12/streamgate/internal/config"
	"github.com/rafaelmgr12/streamgate/internal/handler"
	"github.com/stretchr/testify/assert"
)

func TestHTTPHandler(t *testing.T) {
	cfg := &config.Config{}
	h := handler.NewHTTPHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "Hello from API Gateway", rr.Body.String())
}

func TestHTTPHandler_Healthz(t *testing.T) {
	cfg := &config.Config{}
	h := handler.NewHTTPHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "ok", rr.Body.String())
}

func TestHTTPHandler_NotFoundForUnknownPath(t *testing.T) {
	cfg := &config.Config{}
	h := handler.NewHTTPHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHTTPHandler_ServiceProxy_StripsPrefixAndForwards(t *testing.T) {
	var seenPath string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("from-backend"))
	}))
	defer backend.Close()

	cfg := &config.Config{
		Services: []config.ServiceConfig{
			{
				Name:       "test",
				PathPrefix: "/api/test/",
				Backends:   []string{backend.URL},
			},
		},
	}

	h := handler.NewHTTPHandler(cfg)
	server := httptest.NewServer(h)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/test/hello")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}

	assert.Equal(t, http.StatusTeapot, resp.StatusCode)
	assert.Equal(t, "from-backend", string(body))
	assert.Equal(t, "/hello", seenPath)
}

func TestHTTPHandler_ServiceProxy_MultipleBackendsUsed(t *testing.T) {
	var hits1 int32
	var hits2 int32

	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits1, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("backend-1"))
	}))
	defer backend1.Close()

	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits2, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("backend-2"))
	}))
	defer backend2.Close()

	cfg := &config.Config{
		Services: []config.ServiceConfig{
			{
				Name:       "test",
				PathPrefix: "/api/test/",
				Backends:   []string{backend1.URL, backend2.URL},
			},
		},
	}

	h := handler.NewHTTPHandler(cfg)
	server := httptest.NewServer(h)
	defer server.Close()

	for i := 0; i < 4; i++ {
		resp, err := http.Get(server.URL + "/api/test/any")
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		_, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
	}

	if atomic.LoadInt32(&hits1) == 0 || atomic.LoadInt32(&hits2) == 0 {
		t.Fatalf("expected both backends to receive at least one request, got hits1=%d hits2=%d", hits1, hits2)
	}
}

func TestHTTPHandler_ServiceProxy_SkipsUnhealthyBackend(t *testing.T) {
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

	h := handler.NewHTTPHandler(cfg)
	server := httptest.NewServer(h)
	defer server.Close()

	for i := 0; i < 7; i++ {
		resp, err := http.Get(server.URL + "/api/test/any")
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		_, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
	}

	if atomic.LoadInt32(&badHits) != 3 {
		t.Fatalf("expected bad backend to be hit 3 times, got %d", badHits)
	}
	if atomic.LoadInt32(&goodHits) != 4 {
		t.Fatalf("expected good backend to be hit 4 times, got %d", goodHits)
	}
}
