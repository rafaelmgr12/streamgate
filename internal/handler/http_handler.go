package handler

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/rafaelmgr12/streamgate/internal/config"
	"github.com/rafaelmgr12/streamgate/internal/middleware"
)

type serviceRoute struct {
	name       string
	pathPrefix string
	proxy      *httputil.ReverseProxy
}

type backendPool struct {
	targets []*url.URL
	idx     uint64
}

func (p *backendPool) next() *url.URL {
	n := atomic.AddUint64(&p.idx, 1)
	return p.targets[(n-1)%uint64(len(p.targets))]
}

func parseBackendURL(raw string) (*url.URL, error) {
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	return url.Parse(raw)
}

func buildServiceRoutes(cfg *config.Config) []serviceRoute {
	if cfg == nil {
		return nil
	}

	routes := make([]serviceRoute, 0, len(cfg.Services))

	for _, svc := range cfg.Services {
		if svc.PathPrefix == "" || len(svc.Backends) == 0 {
			continue
		}

		pool := &backendPool{}
		for _, raw := range svc.Backends {
			u, err := parseBackendURL(raw)
			if err != nil {
				log.Printf("invalid backend for service %q: %v", svc.Name, err)
				continue
			}
			pool.targets = append(pool.targets, u)
		}
		if len(pool.targets) == 0 {
			continue
		}

		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				target := pool.next()
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
				req.Host = target.Host
				// path is already stripped by http.StripPrefix below
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

// NewHTTPHandler creates a new HTTP handler with routing/proxy behavior.
func NewHTTPHandler(cfg *config.Config) http.Handler {
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

	// Register config-driven routes
	for _, rt := range buildServiceRoutes(cfg) {
		prefix := rt.pathPrefix
		// Ensure prefix semantics; config should typically end with "/"
		stripped := strings.TrimRight(prefix, "/")
		if stripped == "" {
			stripped = "/"
		}

		// Strip the prefix before proxying so backend sees shorter path
		mux.Handle(prefix, http.StripPrefix(stripped, rt.proxy))
	}

	// Compose middleware chain (outermost last in the list)
	return middleware.Chain(
		mux,
		middleware.Logging,
	)
}
