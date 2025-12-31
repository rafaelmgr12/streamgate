package handler

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/rafaelmgr12/streamgate/internal/balancer"
	"github.com/rafaelmgr12/streamgate/internal/config"
	"github.com/rafaelmgr12/streamgate/internal/middleware"
)

type serviceRoute struct {
	name       string
	pathPrefix string
	proxy      *httputil.ReverseProxy
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

		selector, err := balancer.NewSelector(targets, svc.LoadBalancing)
		if err != nil {
			log.Printf("%v, using %s", err, balancer.DefaultAlgorithm)
		}
		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				target := selector.Next()
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
// The config argument is optional; when omitted, the handler will only expose
// the default root and healthz endpoints.
func NewHTTPHandler(cfgs ...*config.Config) http.Handler {
	var cfg *config.Config
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
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
