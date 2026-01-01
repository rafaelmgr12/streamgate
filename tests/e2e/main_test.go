package e2e

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/rafaelmgr12/streamgate/proto/greeter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func getFreeAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not get free port: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("could not release free port: %v", err)
	}
	return addr
}

func waitForHTTP(t *testing.T, url string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server did not start within %s", timeout)
}

func TestMainE2E(t *testing.T) {
	// Resolve project root and binary path
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("could not get working directory: %v", err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	binPath := filepath.Join(root, "streamgate-e2e")

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/main.go")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("could not build binary: %v", err)
	}
	defer os.Remove(binPath)

	httpAddr := getFreeAddr(t)
	grpcAddr := getFreeAddr(t)

	// Run the binary
	cmd = exec.Command(binPath)
	cmd.Dir = root // Root directory of the project
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"STREAMGATE_HTTP_ADDR="+httpAddr,
		"STREAMGATE_GRPC_ADDR="+grpcAddr,
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("could not start binary: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process == nil {
			return
		}
		_ = cmd.Process.Signal(os.Interrupt)
		_, _ = cmd.Process.Wait()
	})

	// Give the server time to start
	waitForHTTP(t, "http://"+httpAddr+"/healthz", 5*time.Second)

	// Make a request to the server
	resp, err := http.Get("http://" + httpAddr)
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
	resp, err = http.Get("http://" + httpAddr + "/healthz")
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
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	// Cleanup handled by t.Cleanup to ensure the server stops on failures.
}
