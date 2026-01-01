package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type ctxKey int

const requestIDkey ctxKey = iota

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = newRequestID()
			if id != "" {
				r.Header.Set("X-Request-ID", id)
			}
		}
		if id != "" {
			w.Header().Set("X-Request-ID", id)
		}

		ctx := context.WithValue(r.Context(), requestIDkey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetRequestID(ctx context.Context) (string, bool) {
	v := ctx.Value(requestIDkey)
	if v == nil {
		return "", false
	}
	id, ok := v.(string)
	return id, ok
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}

	return hex.EncodeToString(b[:])
}
