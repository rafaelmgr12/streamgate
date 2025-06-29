package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"time"
)

// responseWriter is a wrapper for http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK} // Default to 200
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logging logs all incoming HTTP requests with structured data.
func Logging(next http.Handler) http.Handler {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := newResponseWriter(w)

		next.ServeHTTP(rw, r)

		logger.Info("incoming request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"status", rw.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
