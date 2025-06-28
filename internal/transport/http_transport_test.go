package transport_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/rafaelmgr12/streamgate/internal/transport"
	"github.com/stretchr/testify/assert"
)

func TestHTTPTransport(t *testing.T) {
	// Mock handler for the server
	mockHandler := http.NewServeMux()
	mockHandler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// 1. Test NewHTTPTransport
	addr := ":8081"
	httpTransport := transport.NewHTTPTransport(addr, mockHandler)
	assert.NotNil(t, httpTransport, "NewHTTPTransport should not return nil")

	// 2. Test Addr method
	assert.Equal(t, addr, httpTransport.Addr(), "Addr() should return the correct address")

	// 3. Test ListenAndServe and Close methods
	go func() {
		err := httpTransport.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			assert.Fail(t, "ListenAndServe() returned an unexpected error: %v", err)
		}
	}()

	// Allow time for the server to start
	time.Sleep(100 * time.Millisecond)

	// Make a request to verify the server is running
	resp, err := http.Get("http://localhost" + addr)
	assert.NoError(t, err, "Should be able to make a request to the server")
	if resp != nil {
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Server should return 200 OK")
		resp.Body.Close()
	}

	// Test Close method
	err = httpTransport.Close()
	assert.NoError(t, err, "Close() should not return an error")

	// Allow time for the server to shut down
	time.Sleep(100 * time.Millisecond)

	// Verify the server is no longer running
	_, err = http.Get("http://localhost" + addr)
	assert.Error(t, err, "Should not be able to make a request to a closed server")
}
