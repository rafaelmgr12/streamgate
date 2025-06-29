package transport

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HTTPTransport is an implementation of the Transport interface for HTTP.
// It wraps a standard Go http.Server to handle HTTP requests.
type HTTPTransport struct {
	server *http.Server
	addr   string
}

// NewHTTPTransport creates and configures a new HTTPTransport.
// It takes a listening address and an http.Handler to process requests.
func NewHTTPTransport(addr string, handler http.Handler) *HTTPTransport {
	return &HTTPTransport{
		addr: addr,
		server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
	}
}

// Addr returns the configured listening address of the HTTP server.
func (h *HTTPTransport) Addr() string {
	return h.addr
}

// ListenAndServe starts the HTTP server and blocks until the server stops.
// Any error during startup or runtime, except for http.ErrServerClosed, is returned.
func (h *HTTPTransport) ListenAndServe() error {
	return h.server.ListenAndServe()
}

// Dial is not supported by the HTTPTransport as it is a server-side transport.
// It will always return an error.
func (h *HTTPTransport) Dial(address string) error {
	return fmt.Errorf("dial is not supported for HTTP transport")
}

// Close gracefully shuts down the HTTP server.
// It allows active connections to finish within a 5-second timeout.
func (h *HTTPTransport) Close() error {
	// Create a context with a timeout to allow for graceful shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return h.server.Shutdown(ctx)
}
