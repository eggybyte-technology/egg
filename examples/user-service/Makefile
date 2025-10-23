.PHONY: help build run test lint generate clean docker-build docker-run db-up db-down

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
	@echo "  make db-up         - Start database (MySQL)"
	@echo "  make db-down       - Stop database"

# Build the service
build:
	@echo "Building user-service..."
	@go build -o bin/user-service cmd/server/main.go
	@echo "Build complete: bin/user-service"

# Run the service
run:
	@echo "Running user-service..."
	@go run cmd/server/main.go

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
	@docker build -t user-service:latest -f ../../build/Dockerfile.backend .
	@echo "Docker build complete"

# Run Docker container
docker-run:
	@echo "Running Docker container..."
	@docker run -p 8080:8080 -p 8081:8081 -p 9091:9091 --rm user-service:latest

# Start database (MySQL)
db-up:
	@echo "Starting MySQL database..."
	@docker run -d \
		--name user-service-db \
		-e MYSQL_ROOT_PASSWORD=password \
		-e MYSQL_DATABASE=userdb \
		-e MYSQL_USER=user \
		-e MYSQL_PASSWORD=password \
		-p 3306:3306 \
		mysql:8.0
	@echo "Database started. Connection: user:password@tcp(localhost:3306)/userdb"

# Stop database
db-down:
	@echo "Stopping database..."
	@docker stop user-service-db || true
	@docker rm user-service-db || true
	@echo "Database stopped"

