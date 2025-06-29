package transport_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/rafaelmgr12/streamgate/internal/handler"
	"github.com/rafaelmgr12/streamgate/internal/transport"
	"github.com/rafaelmgr12/streamgate/proto/greeter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestGRPCTransport(t *testing.T) {
	h := handler.NewGRPCHandler()

	// Use port 0 to get a random free port
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := lis.Addr().String()
	lis.Close()

	trans := transport.NewGRPCTransport(addr, h)

	go func() {
		if err := trans.ListenAndServe(); err != nil {
			t.Logf("ListenAndServe failed: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond) // Give server time to start

	// Set up a connection to the server.
	conn, err := grpc.Dial(trans.Addr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	c := greeter.NewGreeterClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.SayHello(ctx, &greeter.HelloRequest{Name: "World"})
	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", r.GetMessage())

	err = trans.Close()
	assert.NoError(t, err)
}
