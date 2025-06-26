package transport

import "context"

// Message represents a generic inbound or outbound message/request.
// You can expand this struct or make it an interface.
type Message interface {
	Context() context.Context
	Payload() []byte
}

// Transport is a generic interface for network communication.
// It can represent servers or clients.
type Transport interface {
	Addr() string
	ListenAndServe() error     // For servers (HTTP/TCP/etc.)
	Dial(address string) error // For clients (optional, or return error if not supported)
	Consume() <-chan Message   // Channel of inbound messages
	Close() error
}
