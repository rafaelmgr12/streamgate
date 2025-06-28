package transport

import (
	"context"
	"net/http"
)

type HTTPTransport struct {
	server  *http.Server
	addr    string
	msgChan chan Message
}

func NewHTTPTransport(addr string, handler http.Handler) *HTTPTransport {
	return &HTTPTransport{
		addr:    addr,
		msgChan: make(chan Message, 100), // Buffered channel for efficiency
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
	// Not implemented for HTTP server; return error or panic
	panic("Not implemented")
}

func (h *HTTPTransport) Consume() <-chan Message {
	return h.msgChan
}

func (h *HTTPTransport) Close() error {
	return h.server.Shutdown(context.Background())
}
