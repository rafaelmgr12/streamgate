package transport

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/rafaelmgr12/streamgate/proto/greeter"
)

// GRPCTransport manages a gRPC server.
type GRPCTransport struct {
	addr   string
	server *grpc.Server
}

// NewGRPCTransport creates a new GRPCTransport.
func NewGRPCTransport(addr string, handler greeter.GreeterServer) *GRPCTransport {
	server := grpc.NewServer()
	greeter.RegisterGreeterServer(server, handler)
	return &GRPCTransport{
		addr:   addr,
		server: server,
	}
}

// ListenAndServe starts the gRPC server.
func (t *GRPCTransport) ListenAndServe() error {
	lis, err := net.Listen("tcp", t.addr)
	if err != nil {
		return err
	}
	return t.server.Serve(lis)
}

// Addr returns the address the transport is listening on.
func (t *GRPCTransport) Addr() string {
	return t.addr
}

// Close gracefully shuts down the gRPC server.
func (t *GRPCTransport) Close() error {
	t.server.GracefulStop()
	return nil
}

// Dial is not implemented for gRPC transport.
func (t *GRPCTransport) Dial(address string) error {
	return fmt.Errorf("Dial not supported for gRPC transport")
}

func (t *GRPCTransport) Shutdown(ctx context.Context) error {
	t.server.GracefulStop()
	return nil
}
