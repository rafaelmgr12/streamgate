package integrations

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rafaelmgr12/streamgate/internal/handler"
)

func TestAPIGatewayIntegration(t *testing.T) {
	h := handler.NewHTTPHandler()
	server := httptest.NewServer(h)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("could not send GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read response body: %v", err)
	}

	expected := "Hello from API Gateway"
	if string(body) != expected {
		t.Errorf("expected response body to be %q; got %q", expected, string(body))
	}
}
