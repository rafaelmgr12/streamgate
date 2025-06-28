# Go API Gateway + L4 Load Balancer

A modern, extensible API Gateway and Layer 4 Load Balancer implemented in Go.  
This project serves as a unified entrypoint for HTTP (and, in the future, any TCP-based protocols), enabling scalable microservice architectures and robust traffic management.

---

## Features

- **Generic Transport Layer:** Pluggable transport abstracation for HTTP, TCP, and future protocols (gRPC, WebSocket, etc.)
- **API Gateway Logic:** Centralized routing, authentication, rate limiting, and observability for all services.
- **L4 Load Balancing:** Efficient, protocol-agnostic TCP load balancing with support for backend health checks.
- **Config-Driven:** Easily map routes, services, and backends via configuration.
- **Designed for Extensibility:** Add new protocols or features (e.g., gRPC, raw TCP) by implementing the `Transport` interface.
- **Cloud Native:** Container-ready (Docker), stateless, and scalable.

---

## Architecture Overview

```bash
            +------------+        +-----------------+         +------------------+
Clients --> | Transport  |  --->  | Gateway Routing |  --->   | Service Backends |
            | (HTTP/TCP) |        +-----------------+         +------------------+
            +------------+               |        |
                                 +-------+        +-------+
                                 |   Auth / Rate Limiting  |
                                 +------------------------+
```

- **Transport Layer:** Abstracts incoming protocol handling (e.g., HTTP, TCP).
- **Gateway Layer:** Performs routing, filtering, and forwarding to backend services.
- **Load Balancer:** Distributes connections or requests across backend service instances.

---

## Getting Started

### Prerequisites

- [Go 1.22+](https://golang.org/doc/go1.22)
- [Docker](https://docs.docker.com/get-docker/) (optional, for containerization)

### Clone the repository

```bash
git clone https://github.com/yourname/apigw-lb.git
cd apigw-lb
go mod tidy
```

### Directory Structure

```bash
.
├── main.go             # Main application entrypoint
├── go.mod
├── internal/
│   ├── config/         # Configuration loading
│   ├── middleware/     # HTTP middleware
│   ├── proxy/          # Reverse proxy logic
│   ├── routes/         # Route definitions
│   ├── server/         # Server setup
│   └── transport/      # Transport abstraction (e.g., HTTP)
└── Makefile
```

---

## Usage

### Run Locally

```bash
go run main.go
```

- By default, the gateway listens on `:8080`.
- You can configure ports, services, and routing rules in `internal/config/config.go` or via environment variables.

### Docker

```bash
docker build -t apigw-lb .
docker run -p 8080:8080 apigw-lb
```

---

## Configuration

The service is configured via `internal/config/config.go` and can be extended to use YAML or JSON files.

- **Transports:** Which protocols/ports to enable (HTTP, TCP, etc.).
- **Services:** Logical service names and route patterns.
- **Backends:** List of backend instances (host:port) per service.

*Example (conceptual):*

```yaml
transports:
  - type: http
    addr: ":8080"

services:
  - name: users
    path_prefix: "/api/users/"
    backends:
      - "usersvc1:9001"
      - "usersvc2:9001"
```

---

## Extending

- To support a new protocol, implement the `Transport` interface in `internal/transport/`.
- To add new routes or proxy logic, see the `internal/routes/` and `internal/proxy/` packages.
- Middleware (auth, rate limiting, logging) can be added in the `internal/middleware/` package.

---

## Contributing

Pull requests and issues are welcome!  
Please ensure new code is tested and documented.

---

## License

MIT License

---

## Roadmap

- [x] Transport abstraction (`Transport` interface)
- [x] HTTP transport implementation
- [ ] TCP/L4 transport
- [ ] API Gateway features (routing, middleware)
- [ ] Reverse Proxy implementation
- [ ] Service discovery
- [ ] Dynamic configuration reload
- [ ] Observability (metrics, tracing)
- [ ] gRPC/WebSocket support

---

## Credits

Inspired by:

- [Tyk API Gateway](https://github.com/TykTechnologies/tyk)
- [KrakenD](https://github.com/krakendio/krakend)
- [Caddy](https://github.com/caddyserver/caddy)
- [Go net/http](https://golang.org/pkg/net/http/)
