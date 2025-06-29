package middleware

import (
	"log"
	"net/http"
)

// Logging logs all incoming HTTP requests.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("request received: method=%s host=%s path=%s remote_addr=%s", r.Method, r.Host, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
