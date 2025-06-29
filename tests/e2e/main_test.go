package e2e

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestMainE2E(t *testing.T) {
	// Build the binary
	cmd := exec.Command("go", "build", "-o", "streamgate", "../../cmd/main.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("could not build binary: %v", err)
	}
	defer os.Remove("streamgate")

	// Run the binary
	cmd = exec.Command("./streamgate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("could not start binary: %v", err)
	}

	// Give the server time to start
	time.Sleep(1 * time.Second)

	// Make a request to the server
	resp, err := http.Get("http://localhost:8080")
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

	// Stop the server
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("could not send interrupt signal: %v", err)
	}

	// Wait for the process to exit
	if _, err := cmd.Process.Wait(); err != nil {
		t.Fatalf("error waiting for process to exit: %v", err)
	}
}
