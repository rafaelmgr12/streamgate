package handler

import (
	"net/http"

	"github.com/rafaelmgr12/streamgate/internal/middleware"
)

// NewHTTPHandler creates a new HTTP handler.
func NewHTTPHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello from API Gateway"))
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	return middleware.Logging(mux)
}
