package handler

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/rafaelmgr12/streamgate/internal/balancer"
	"github.com/rafaelmgr12/streamgate/internal/config"
	"github.com/rafaelmgr12/streamgate/internal/middleware"
)

type serviceRoute struct {
	name       string
	pathPrefix string
	proxy      *httputil.ReverseProxy
}

type serviceHealthResponse struct {
	Services []serviceHealth `json:"services"`
}

type serviceHealth struct {
	Name       string          `json:"name"`
	PathPrefix string          `json:"path_prefix"`
	Backends   []backendHealth `json:"backends"`
}

type backendHealth struct {
	URL                 string     `json:"url"`
	Healthy             bool       `json:"healthy"`
	ConsecutiveFailures int        `json:"consecutive_failures,omitempty"`
	UnhealthyUntil      *time.Time `json:"unhealthy_until,omitempty"`
}

type retryRoundTripper struct {
	base       http.RoundTripper
	maxRetries int
}

type targetContextKey struct{}

var selectedTargetKey = targetContextKey{}

// NewHTTPHandler creates a new HTTP handler with routing/proxy behavior.
// The config argument is optional; when omitted, the handler will only expose
// the default root and healthz endpoints.
func NewHTTPHandler(cfgs ...*config.Config) http.Handler {
	var cfg *config.Config
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
	return NewHTTPHandlerWithTracker(cfg, nil)
}

// NewHTTPHandlerWithTracker builds the handler with a provided health tracker.
func NewHTTPHandlerWithTracker(cfg *config.Config, tracker *balancer.HealthTracker) http.Handler {
	if tracker == nil {
		tracker = balancer.NewHealthTracker(0, 0)
	}

	return newHTTPHandler(cfg, tracker)
}

func newHTTPHandler(cfg *config.Config, tracker *balancer.HealthTracker) http.Handler {

	mux := http.NewServeMux()

	// Root + healthz preserved for existing tests and e2e
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte("Hello from API Gateway"))
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("/healthz/services", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		resp := buildServiceHealthResponse(cfg, tracker)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	})

	// Register config-driven routes
	for _, rt := range buildServiceRoutes(cfg, tracker) {
		prefix := rt.pathPrefix
		// Ensure prefix semantics; config should typically end with "/"
		stripped := strings.TrimRight(prefix, "/")
		if stripped == "" {
			stripped = "/"
		}

		// Strip the prefix before proxying so backend sees shorter path
		mux.Handle(prefix, http.StripPrefix(stripped, rt.proxy))
	}

	limiter := middleware.NewRateLimiter(10, 5) // 10 tokens, refill 5 tokens/sec

	// Compose middleware chain (outermost last in the list)
	return middleware.Chain(
		mux,
		middleware.RequestID,
		middleware.Logging,
		middleware.RateLimitMiddleware(limiter),
	)
}
func (rt *retryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.maxRetries <= 0 || !isIdempotent(req.Method) {
		return rt.base.RoundTrip(req)
	}

	if req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
		return rt.base.RoundTrip(req)
	}

	var lastErr error
	for attempt := 0; attempt <= rt.maxRetries; attempt++ {
		r := req
		if attempt > 0 {
			r = req.Clone(req.Context())
			if req.GetBody != nil {
				body, err := req.GetBody()
				if err != nil {
					return nil, err
				}
				r.Body = body
			}
		}

		resp, err := rt.base.RoundTrip(r)
		if err == nil {
			if shouldRetryStatus(resp.StatusCode) && attempt < rt.maxRetries {
				resp.Body.Close()
				continue
			}
			return resp, nil
		}
		lastErr = err
	}

	return nil, lastErr
}

func parseBackendURL(raw string) (*url.URL, error) {
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	return url.Parse(raw)
}

func buildServiceRoutes(cfg *config.Config, tracker *balancer.HealthTracker) []serviceRoute {
	if cfg == nil {
		return nil
	}

	routes := make([]serviceRoute, 0, len(cfg.Services))

	for _, svc := range cfg.Services {
		if svc.PathPrefix == "" || len(svc.Backends) == 0 {
			continue
		}

		targets := make([]*url.URL, 0, len(svc.Backends))
		for _, raw := range svc.Backends {
			u, err := parseBackendURL(raw)
			if err != nil {
				log.Printf("invalid backend for service %q: %v", svc.Name, err)
				continue
			}
			targets = append(targets, u)
		}
		if len(targets) == 0 {
			continue
		}

		selector, err := balancer.NewHealthAwareSelector(targets, svc.LoadBalancing, tracker)
		if err != nil {
			log.Printf("%v, using %s", err, balancer.DefaultAlgorithm)
		}

		proxyTransport := newProxyTransport()

		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				target := selector.Next()
				if target != nil {
					ctx := context.WithValue(req.Context(), selectedTargetKey, target)
					*req = *req.WithContext(ctx)
				}
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
				req.Host = target.Host
				// path is already stripped by http.StripPrefix below
			},
			Transport: &retryRoundTripper{
				base:       proxyTransport,
				maxRetries: 1,
			},
			ModifyResponse: func(resp *http.Response) error {
				if tracker == nil {
					return nil
				}
				if target := selectedTarget(resp.Request.Context()); target != nil {
					if resp.StatusCode >= 500 {
						tracker.MarkFailure(target)
					} else {
						tracker.MarkSuccess(target)
					}
				}
				return nil
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				if tracker != nil {
					if target := selectedTarget(r.Context()); target != nil {
						tracker.MarkFailure(target)
					}
				}
				http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			},
		}

		routes = append(routes, serviceRoute{
			name:       svc.Name,
			pathPrefix: svc.PathPrefix,
			proxy:      proxy,
		})
	}

	return routes
}

func newProxyTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   3 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	}
}

func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

func shouldRetryStatus(status int) bool {
	switch status {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func selectedTarget(ctx context.Context) *url.URL {
	target, _ := ctx.Value(selectedTargetKey).(*url.URL)
	return target
}

func buildServiceHealthResponse(cfg *config.Config, tracker *balancer.HealthTracker) serviceHealthResponse {
	resp := serviceHealthResponse{}
	if cfg == nil {
		resp.Services = []serviceHealth{}
		return resp
	}

	snapshot := map[string]balancer.BackendStatus{}
	if tracker != nil {
		snapshot = tracker.Snapshot()
	}
	now := time.Now()

	resp.Services = make([]serviceHealth, 0, len(cfg.Services))
	for _, svc := range cfg.Services {
		svcHealth := serviceHealth{
			Name:       svc.Name,
			PathPrefix: svc.PathPrefix,
			Backends:   make([]backendHealth, 0, len(svc.Backends)),
		}

		for _, raw := range svc.Backends {
			u, err := parseBackendURL(raw)
			if err != nil {
				continue
			}
			key := u.String()
			state, ok := snapshot[key]
			healthy := true
			if ok && !state.UnhealthyUntil.IsZero() && now.Before(state.UnhealthyUntil) {
				healthy = false
			}

			backend := backendHealth{
				URL:     key,
				Healthy: healthy,
			}
			if ok {
				backend.ConsecutiveFailures = state.ConsecutiveFailures
				if !state.UnhealthyUntil.IsZero() && now.Before(state.UnhealthyUntil) {
					until := state.UnhealthyUntil
					backend.UnhealthyUntil = &until
				}
			}

			svcHealth.Backends = append(svcHealth.Backends, backend)
		}

		resp.Services = append(resp.Services, svcHealth)
	}

	return resp
}
