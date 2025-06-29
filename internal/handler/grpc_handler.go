package handler

import (
	"context"
	"fmt"

	"github.com/rafaelmgr12/streamgate/proto/greeter"
)

// GRPCHandler implements the gRPC GreeterServer interface.
type GRPCHandler struct {
	greeter.UnimplementedGreeterServer
}

// NewGRPCHandler creates a new GRPCHandler.
func NewGRPCHandler() *GRPCHandler {
	return &GRPCHandler{}
}

// SayHello handles the SayHello gRPC request.
func (h *GRPCHandler) SayHello(ctx context.Context, req *greeter.HelloRequest) (*greeter.HelloReply, error) {
	return &greeter.HelloReply{Message: fmt.Sprintf("Hello, %s!", req.Name)}, nil
}
