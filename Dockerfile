FROM golang:1.23-alpine AS builder

WORKDIR /app

# Pre-fetch dependencies
COPY go.mod go.sum ./
RUN go mod download  

# Copy source
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o streamgate ./cmd/main.go

# Minimal runtime image
FROM gcr.io/distroless/static-debian12

WORKDIR /app

COPY --from=builder /app/streamgate /app/streamgate
COPY config.yaml /app/config.yaml

EXPOSE 8080 50051

ENTRYPOINT ["/app/streamgate"]