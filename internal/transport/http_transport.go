package transport

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type HTTPTransport struct {
	server *http.Server
	addr   string
}

func NewHTTPTransport(addr string, handler http.Handler) *HTTPTransport {
	return &HTTPTransport{
		addr: addr,
		server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
	}
}

func (h *HTTPTransport) Addr() string {
	return h.addr
}

func (h *HTTPTransport) ListenAndServe() error {
	return h.server.ListenAndServe()
}

func (h *HTTPTransport) Dial(address string) error {
	return fmt.Errorf("dial is not supported for HTTP transport")
}

func (h *HTTPTransport) Close() error {
	// Create a context with a timeout to allow for graceful shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return h.server.Shutdown(ctx)
}
