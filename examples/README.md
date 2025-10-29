# Egg Framework Examples

This directory contains complete, production-ready examples demonstrating how to use the egg microservice framework.

## Available Examples

### 1. Minimal Connect Service

**Location**: `minimal-connect-service/`

A minimal Connect-based microservice showing the essentials:

- Single-call service initialization with `servicex`
- Unary and server-streaming RPC methods
- Multi-language greeting support
- Built-in observability (tracing, metrics, logging)
- Graceful shutdown

**Perfect for**: Learning the basics, starting a new microservice, reference implementation

**Documentation**: See [minimal-connect-service/README.md](minimal-connect-service/README.md)

**Quick Start**:

```bash
cd minimal-connect-service
go run main.go
```

Test:

```bash
curl -X POST http://localhost:8080/greet.v1.GreeterService/SayHello \
  -H "Content-Type: application/json" \
  -d '{"name":"World","language":"en"}'
```

---

### 2. User Service (CRUD Example)

**Location**: `user-service/`

A complete CRUD microservice with layered architecture:

- Clean architecture (handler → service → repository → model)
- Full user management (Create, Read, Update, Delete, List)
- Optional database integration (GORM with MySQL/PostgreSQL)
- In-memory mock repository for demo
- Comprehensive error handling and validation
- Pagination support

**Perfect for**: Production microservices, CRUD APIs, learning layered architecture

**Documentation**: See [user-service/README.md](user-service/README.md)

**Quick Start**:

```bash
cd user-service
go run cmd/server/main.go
```

Test:

```bash
# Create a user
curl -X POST http://localhost:8080/user.v1.UserService/CreateUser \
  -H "Content-Type: application/json" \
  -d '{"name":"John Doe","email":"john@example.com"}'

# List users
curl -X POST http://localhost:8080/user.v1.UserService/ListUsers \
  -H "Content-Type: application/json" \
  -d '{"page":1,"page_size":10}'
```

---

## Common Features

Both examples demonstrate:

✅ **servicex Integration**: One-call service initialization  
✅ **Connect Protocol**: HTTP/2-based RPC with efficient serialization  
✅ **Observability**: Prometheus metrics, structured logging (metrics-only focus)  
✅ **Health Checks**: Built-in `/health` and `/metrics` endpoints  
✅ **Graceful Shutdown**: Automatic signal handling and resource cleanup  
✅ **Production Ready**: Error handling, validation, context propagation  
✅ **Multi-Platform Support**: Automatic Docker image platform detection (arm64/amd64)  

## Project Structure

Each example follows the egg framework conventions:

```
example-service/
├── api/                    # Protocol Buffer definitions
│   ├── buf.gen.yaml       # Buf code generation config
│   ├── buf.yaml           # Buf module config
│   └── service/
│       └── v1/
│           └── service.proto
├── gen/                    # Generated code (by buf)
│   └── go/
│       └── service/
│           └── v1/
├── cmd/                    # Entrypoints (for larger services)
│   └── server/
│       └── main.go
├── internal/               # Internal packages (for larger services)
│   ├── handler/
│   ├── service/
│   ├── repository/
│   ├── model/
│   └── config/
├── main.go                 # Simple entrypoint (for minimal services)
├── go.mod
├── Makefile
└── README.md
```

## Building and Running

### Prerequisites

- Go 1.23 or later
- Docker and Docker Compose (for deployment)
- Buf CLI (for regenerating protobuf code): `go install github.com/bufbuild/buf/cmd/buf@latest`

### Quick Start with Make

From the `examples/` directory:

```bash
# Build & Deploy
make docker-build    # Build Docker images for all examples
make deploy-up       # Start infrastructure + services
make deploy-status   # Check service status

# Testing
make test            # Run full integration tests

# Cleanup
make services-down   # Stop application services only
make deploy-down     # Stop everything (including infrastructure)
```

### Development Workflow

**Build and run locally:**
```bash
cd minimal-connect-service  # or user-service
go run main.go              # Run directly
```

**Build Docker image:**
```bash
cd ..                       # Back to examples/
make docker-build           # Build all service images
```

**Deploy with Docker Compose:**
```bash
make infra-up              # Start infrastructure (MySQL)
make services-up           # Start application services
```

**View logs and status:**
```bash
make deploy-logs           # Follow all logs
make deploy-status         # Show container status
```

**Rebuild and restart:**
```bash
make docker-build          # Rebuild Docker images
make services-restart      # Restart services with latest images
```

## Configuration

Both examples support configuration via environment variables. See each example's README for details.

Common environment variables:

```bash
SERVICE_NAME=my-service
SERVICE_VERSION=0.1.0
HTTP_ADDR=:8080
HEALTH_ADDR=:8081
METRICS_ADDR=:9091
ENABLE_TRACING=true
ENABLE_METRICS=true
ENABLE_DEBUG_LOGS=false
SLOW_REQUEST_MILLIS=1000
```

## Deployment & Observability

### Infrastructure Services

The `examples/deploy/` directory contains Docker Compose configurations for:

- **MySQL 9.4** - Database (port 3306)

**Note**: Tracing infrastructure (Jaeger, OTEL Collector) has been removed. Examples now focus on **metrics-only** observability using Prometheus.

See [deploy/README.md](deploy/README.md) for detailed deployment documentation.

### Accessing Services

| Service | URL | Description |
|---------|-----|-------------|
| Minimal Service | http://localhost:8080 | Connect-RPC endpoints |
| Minimal Health | http://localhost:8081/health | Health check |
| Minimal Metrics | http://localhost:9091/metrics | Prometheus metrics |
| User Service | http://localhost:8082 | Connect-RPC endpoints |
| User Health | http://localhost:8083/health | Health check |
| User Metrics | http://localhost:9092/metrics | Prometheus metrics |
| MySQL | localhost:3306 | Database (user: egguser, pass: eggpassword) |

### Health Checks

```bash
curl http://localhost:8081/health  # Minimal service
curl http://localhost:8083/health  # User service
```

### Metrics

```bash
curl http://localhost:9091/metrics  # Minimal service metrics
curl http://localhost:9092/metrics  # User service metrics
```

### Metrics Collection

Services expose Prometheus metrics on dedicated ports. The `connect-tester` tool automatically validates metrics endpoints, checking for:

- **RPC Metrics**: Request counts, durations, sizes
- **Runtime Metrics**: Goroutines, GC, memory usage
- **Process Metrics**: CPU, RSS, uptime
- **Database Metrics**: Connection pool stats (for user-service)

Metrics are collected via pull model (Prometheus scrapes `/metrics` endpoints). No additional infrastructure required.

## Testing with Connect

### Using curl

```bash
curl -X POST http://localhost:8080/<package>.<Service>/<Method> \
  -H "Content-Type: application/json" \
  -d '{"field":"value"}'
```

### Using buf curl (recommended)

```bash
buf curl \
  --protocol connect \
  --http2-prior-knowledge \
  http://localhost:8080/<package>.<Service>/<Method> \
  -d '{"field":"value"}'
```

### Using Connect client libraries

See [connectrpc.com](https://connectrpc.com/) for client libraries in multiple languages.

## Development Workflow

1. **Define your API** in `.proto` files under `api/`
2. **Generate code** with `make generate` or `cd api && buf generate`
3. **Implement handlers** in `internal/handler/` (or directly in `main.go` for simple services)
4. **Add business logic** in `internal/service/`
5. **Implement data access** in `internal/repository/` (if needed)
6. **Register handlers** in the `servicex.WithRegister` function
7. **Test** with `make test` and manual testing
8. **Run** with `make run`

## Makefile Targets

The `examples/Makefile` provides comprehensive deployment and testing targets:

### Build & Test
- `make build` - Build all example services (Go)
- `make test` - Run full integration tests
- `make docker-build` - Build Docker images for all services
- `make docker-clean` - Clean Docker images

### Deployment (All Services)
- `make deploy-up` - Start infrastructure + application services
- `make deploy-down` - Stop all services
- `make deploy-restart` - Restart all services
- `make deploy-logs` - Follow service logs
- `make deploy-status` - Show service status

### Infrastructure Only
- `make infra-up` - Start MySQL database
- `make infra-down` - Stop infrastructure
- `make infra-restart` - Restart infrastructure
- `make infra-status` - Show infrastructure status
- `make infra-clean` - Clean infrastructure (including volumes)

### Application Services Only
- `make services-up` - Start minimal-service and user-service
- `make services-down` - Stop application services
- `make services-restart` - Restart application services

## Adding a New Example

To add a new example:

1. Create a new directory under `examples/`
2. Copy the structure from `minimal-connect-service` or `user-service`
3. Update `.proto` files for your service
4. Run `buf generate` to create Go code
5. Implement your service logic
6. Update README.md with your example's specifics
7. Add the service to `examples/deploy/docker-compose.services.yaml`
8. Update `scripts/build.sh` to include your service

## Troubleshooting

### Port already in use

```bash
# Find process using port
lsof -i :8080

# Kill process
kill -9 <PID>
```

### Cannot connect to service

1. Check if service is running: `curl http://localhost:8081/health`
2. Verify port configuration in environment variables
3. Check logs for startup errors

### Protobuf generation fails

1. Install buf: `go install github.com/bufbuild/buf/cmd/buf@latest`
2. Verify `buf.yaml` and `buf.gen.yaml` are correct
3. Run `buf mod update` in the `api/` directory

## Related Documentation

- [egg Framework Documentation](../docs/)
- [servicex Module](../servicex/README.md)
- [connectx Module](../connectx/README.md)
- [Connect Protocol](https://connectrpc.com/)
- [Protocol Buffers](https://protobuf.dev/)
- [Buf CLI](https://buf.build/)

## License

These examples are part of the EggyByte egg framework and are licensed under the MIT License. See the root LICENSE file for details.

