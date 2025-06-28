package handler

import "net"

// ConnectionHandler defines the interface for handling raw network connections.
// This can be implemented for any protocol (TCP, etc.).
type ConnectionHandler interface {
	Handle(conn net.Conn)
}
