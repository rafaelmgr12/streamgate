package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rafaelmgr12/streamgate/internal/handler"
	"github.com/stretchr/testify/assert"
)

func TestHTTPHandler(t *testing.T) {
	h := handler.NewHTTPHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "Hello from API Gateway", rr.Body.String())
}

func TestHTTPHandler_Healthz(t *testing.T) {
	h := handler.NewHTTPHandler()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "ok", rr.Body.String())
}
