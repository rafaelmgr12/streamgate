package transport

import "context"

// Message represents a generic data unit transferred over a transport.
// It provides a common abstraction for different message types, such as a WebSocket
// frame or a custom TCP packet.
type Message interface {
	// Context returns the context associated with the message, allowing for cancellation
	// and metadata propagation.
	Context() context.Context
	// Payload returns the raw byte content of the message.
	Payload() []byte
}

// Transport is the foundational interface for network protocols. It defines the essential
// behaviors for network listeners and clients, such as starting, stopping, and connecting.
// This interface is designed to be protocol-agnostic.
type Transport interface {
	// Addr returns the network address the transport is configured to listen on or connect to.
	Addr() string
	// ListenAndServe starts the transport's server component, causing it to listen for
	// incoming connections or requests. It blocks until a critical error occurs.
	ListenAndServe() error
	// Dial attempts to establish a connection to a remote address. It is primarily for
	// client-side transports. Implementations for server-only transports should return
	// an error.
	Dial(address string) error
	// Close gracefully shuts down the transport, releasing any resources.
	Close() error
}

// StreamingTransport extends the base Transport interface for protocols that handle
// a continuous stream of messages, such as WebSockets or raw TCP.
type StreamingTransport interface {
	Transport
	// Consume returns a read-only channel that emits incoming messages. This allows
	// consumers to process messages asynchronously as they arrive.
	Consume() <-chan Message
}
