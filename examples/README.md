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
✅ **Observability**: OpenTelemetry tracing, Prometheus metrics, structured logging  
✅ **Health Checks**: Built-in `/health` and `/metrics` endpoints  
✅ **Graceful Shutdown**: Automatic signal handling and resource cleanup  
✅ **Production Ready**: Error handling, validation, context propagation  

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
- Buf CLI (for regenerating protobuf code): `go install github.com/bufbuild/buf/cmd/buf@latest`

### Using Make

Each example includes a Makefile with common targets:

```bash
make build      # Build the service binary
make run        # Run the service
make test       # Run tests
make lint       # Run linter
make generate   # Regenerate protobuf code
make clean      # Clean build artifacts
```

### Manual Build

```bash
# Build
go build -o bin/service main.go

# Run
./bin/service

# Or run directly
go run main.go
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

## Observability

### Health Check

```bash
curl http://localhost:8081/health
```

### Metrics

```bash
curl http://localhost:9091/metrics
```

### Tracing

Configure OpenTelemetry collector:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

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

## Adding a New Example

To add a new example:

1. Create a new directory under `examples/`
2. Copy the structure from `minimal-connect-service` or `user-service`
3. Update `.proto` files for your service
4. Run `buf generate` to create Go code
5. Implement your service logic
6. Update README.md with your example's specifics
7. Add Makefile targets for building and testing

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

