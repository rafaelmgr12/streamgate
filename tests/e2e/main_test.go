package e2e

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/rafaelmgr12/streamgate/proto/greeter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	cmd.Dir = "../../" // Root directory of the project
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

	// Make a request to the /healthz endpoint
	resp, err = http.Get("http://localhost:8080/healthz")
	if err != nil {
		t.Fatalf("could not send GET request to /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK for /healthz; got %v", resp.Status)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read response body for /healthz: %v", err)
	}

	expected = "ok"
	if string(body) != expected {
		t.Errorf("expected response body for /healthz to be %q; got %q", expected, string(body))
	}

	// Test gRPC connection
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := greeter.NewGreeterClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.SayHello(ctx, &greeter.HelloRequest{Name: "e2e"})
	if err != nil {
		t.Fatalf("could not greet: %v", err)
	}
	if r.GetMessage() != "Hello, e2e!" {
		t.Errorf("unexpected gRPC reply: got %q, want %q", r.GetMessage(), "Hello, e2e!")
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
