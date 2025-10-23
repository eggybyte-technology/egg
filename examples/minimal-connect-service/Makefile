.PHONY: help build run test lint generate clean docker-build docker-run

# Default target
help:
	@echo "Available targets:"
	@echo "  make build         - Build the service binary"
	@echo "  make run           - Run the service"
	@echo "  make test          - Run tests"
	@echo "  make lint          - Run linter"
	@echo "  make generate      - Generate protobuf code"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-run    - Run Docker container"

# Build the service
build:
	@echo "Building greet-service..."
	@go build -o bin/greet-service main.go
	@echo "Build complete: bin/greet-service"

# Run the service
run:
	@echo "Running greet-service..."
	@go run main.go

# Run tests
test:
	@echo "Running tests..."
	@go test -v -race -cover ./...

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run || echo "golangci-lint not installed, skipping..."

# Generate protobuf code
generate:
	@echo "Generating protobuf code..."
	@cd api && buf generate
	@echo "Generation complete"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf gen/
	@echo "Clean complete"

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t greet-service:latest -f ../../build/Dockerfile.backend .
	@echo "Docker build complete"

# Run Docker container
docker-run:
	@echo "Running Docker container..."
	@docker run -p 8080:8080 -p 8081:8081 -p 9091:9091 --rm greet-service:latest

