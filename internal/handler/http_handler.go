package handler

import "net/http"

// NewHTTPHandler creates a new HTTP handler.
func NewHTTPHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello from API Gateway"))
	})
	return mux
}
