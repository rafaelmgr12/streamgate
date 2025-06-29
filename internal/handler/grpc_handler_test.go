package handler_test

import (
	"context"
	"testing"

	"github.com/rafaelmgr12/streamgate/internal/handler"
	"github.com/rafaelmgr12/streamgate/proto/greeter"
	"github.com/stretchr/testify/assert"
)

func TestGRPCHandler_SayHello(t *testing.T) {
	h := handler.NewGRPCHandler()
	req := &greeter.HelloRequest{Name: "World"}
	resp, err := h.SayHello(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, World!", resp.Message)
}
