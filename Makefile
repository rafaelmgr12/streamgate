# Makefile for the Streamgate project

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GORUN=$(GOCMD) run
GOMOD=$(GOCMD) mod

# Binary name
BINARY_NAME=streamgate

# Main package path
MAIN_PACKAGE=./cmd/main.go

# Default target
all: build

# Build the application binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PACKAGE)

# Run the application
run:
	@echo "Running $(BINARY_NAME)..."
	$(GORUN) $(MAIN_PACKAGE)

# Run unit tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Clean up build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

# Install/update dependencies
deps:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Lint the code (requires golangci-lint)
# Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
lint:
	@echo "Linting..."
	golangci-lint run ./...

.PHONY: all build run test clean deps lint
