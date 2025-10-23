# Minimal Connect Service Example

## Overview

This example demonstrates a minimal Connect-based microservice using the `servicex` library from the egg framework. It shows how to set up a production-ready service with minimal code, including observability, health checks, metrics, and graceful shutdown.

## Features

- **Minimal Setup**: One-call service initialization using `servicex.Run`
- **Connect Protocol**: Uses Connect RPC for efficient HTTP/2 communication
- **Observability**: Integrated OpenTelemetry tracing and metrics
- **Health Checks**: Built-in health and readiness endpoints
- **Graceful Shutdown**: Automatic signal handling and graceful shutdown
- **Streaming Support**: Example of server-side streaming RPC

## Project Structure

```
minimal-connect-service/
├── api/                        # Protocol Buffer definitions
│   ├── buf.gen.yaml           # Buf code generation config
│   ├── buf.yaml               # Buf module config
│   └── greet/
│       └── v1/
│           └── greet.proto    # GreeterService definition
├── gen/                        # Generated code (by buf)
│   └── go/
│       └── greet/
│           └── v1/
│               ├── greet.pb.go
│               └── greetv1connect/
│                   └── greet.connect.go
├── main.go                     # Service entrypoint
├── go.mod                      # Go module definition
├── go.sum                      # Go module checksums
├── Makefile                    # Build and development tasks
├── .env.example                # Example environment variables
└── README.md                   # This file
```

## Prerequisites

- Go 1.23 or later
- Buf CLI (for regenerating protobuf code)

## Installation

1. Clone the repository:

```bash
git clone https://github.com/eggybyte-technology/egg.git
cd egg/examples/minimal-connect-service
```

2. Install dependencies:

```bash
go mod download
```

## Configuration

The service can be configured via environment variables. Copy `.env.example` to `.env` and adjust as needed:

```bash
cp .env.example .env
```

### Environment Variables

| Variable              | Default | Description                           |
| --------------------- | ------- | ------------------------------------- |
| `SERVICE_NAME`        | `greet-service` | Service name for observability |
| `SERVICE_VERSION`     | `0.1.0` | Service version                      |
| `HTTP_ADDR`           | `:8080` | HTTP server address                  |
| `HEALTH_ADDR`         | `:8081` | Health check endpoint address        |
| `METRICS_ADDR`        | `:9091` | Metrics endpoint address             |
| `ENABLE_TRACING`      | `true`  | Enable OpenTelemetry tracing         |
| `ENABLE_HEALTH_CHECK` | `true`  | Enable health check endpoint         |
| `ENABLE_METRICS`      | `true`  | Enable Prometheus metrics            |
| `ENABLE_DEBUG_LOGS`   | `false` | Enable debug-level logging           |
| `SLOW_REQUEST_MILLIS` | `1000`  | Threshold for slow request logging   |
| `PAYLOAD_ACCOUNTING`  | `true`  | Enable payload size tracking         |
| `SHUTDOWN_TIMEOUT`    | `15s`   | Graceful shutdown timeout            |

## Running the Service

### Using Go

```bash
go run main.go
```

### Using Make

```bash
make run
```

### Using Docker

```bash
make docker-build
make docker-run
```

## Testing the Service

### Using curl (unary RPC)

```bash
curl -X POST http://localhost:8080/greet.v1.GreeterService/SayHello \
  -H "Content-Type: application/json" \
  -d '{"name":"World","language":"en"}'
```

Expected response:

```json
{
  "message": "Hello, World!",
  "timestamp": "2025-10-23T12:34:56Z"
}
```

### Using curl (server streaming)

```bash
curl -X POST http://localhost:8080/greet.v1.GreeterService/SayHelloStream \
  -H "Content-Type: application/json" \
  -d '{"name":"World","count":5}'
```

### Testing different languages

```bash
# Spanish
curl -X POST http://localhost:8080/greet.v1.GreeterService/SayHello \
  -H "Content-Type: application/json" \
  -d '{"name":"Mundo","language":"es"}'

# French
curl -X POST http://localhost:8080/greet.v1.GreeterService/SayHello \
  -H "Content-Type: application/json" \
  -d '{"name":"Monde","language":"fr"}'

# Chinese
curl -X POST http://localhost:8080/greet.v1.GreeterService/SayHello \
  -H "Content-Type: application/json" \
  -d '{"name":"世界","language":"zh"}'
```

## Observability

### Health Check

```bash
curl http://localhost:8081/health
```

Response:

```json
{
  "status": "ok",
  "timestamp": "2025-10-23T12:34:56Z"
}
```

### Metrics

```bash
curl http://localhost:9091/metrics
```

Key metrics:

- `rpc_server_requests_total`: Total number of RPC requests
- `rpc_server_request_duration_seconds`: RPC request duration histogram
- `rpc_server_request_size_bytes`: RPC request size histogram
- `rpc_server_response_size_bytes`: RPC response size histogram

### Tracing

Configure OpenTelemetry collector endpoint via environment variables or config file.

## Development

### Regenerating Protocol Buffers

```bash
make generate
```

Or manually:

```bash
cd api && buf generate
```

### Running Tests

```bash
make test
```

### Linting

```bash
make lint
```

### Building

```bash
make build
```

## API Reference

### GreeterService

#### SayHello (Unary)

Sends a greeting to the specified name in the requested language.

**Request:**

```protobuf
message SayHelloRequest {
  string name = 1;      // Name to greet (default: "World")
  string language = 2;  // Language code (en, es, fr, de, zh)
}
```

**Response:**

```protobuf
message SayHelloResponse {
  string message = 1;   // Greeting message
  string timestamp = 2; // Response timestamp
}
```

#### SayHelloStream (Server Streaming)

Sends multiple greetings as a server stream.

**Request:**

```protobuf
message SayHelloStreamRequest {
  string name = 1;  // Name to greet (default: "World")
  int32 count = 2;  // Number of greetings (default: 5)
}
```

**Response:**

```protobuf
message SayHelloStreamResponse {
  string message = 1;  // Greeting message
  int32 sequence = 2;  // Sequence number
}
```

## Key Concepts

### servicex Integration

This example uses `servicex.Run()` for unified service initialization:

```go
err := servicex.Run(ctx, servicex.Options{
    ServiceName: "greet-service",
    Config:      &cfg,
    Register: func(app *servicex.App) error {
        // Register Connect handlers
        greeterService := &GreeterService{}
        path, handler := greetv1connect.NewGreeterServiceHandler(
            greeterService,
            connect.WithInterceptors(app.Interceptors()...),
        )
        app.Mux().Handle(path, handler)
        return nil
    },
    EnableTracing:     true,
    EnableHealthCheck: true,
    EnableMetrics:     true,
})
```

### Connect Interceptors

The service automatically includes the following interceptors from `connectx`:

- **Recovery**: Panic recovery with structured logging
- **Timeout**: Request timeout enforcement
- **Logging**: Request/response logging with correlation IDs
- **Metrics**: Prometheus metrics collection
- **Identity**: User identity injection (if configured)
- **Error Mapping**: Structured error responses

## Troubleshooting

### Service doesn't start

1. Check if ports are already in use:

```bash
lsof -i :8080
lsof -i :8081
lsof -i :9091
```

2. Check logs for initialization errors

### Connection refused

Ensure the service is running and listening on the correct address:

```bash
curl http://localhost:8081/health
```

### Slow requests

Check the `SLOW_REQUEST_MILLIS` threshold and adjust as needed. Slow requests are automatically logged.

## License

This example is part of the EggyByte egg framework and is licensed under the MIT License. See the root LICENSE file for details.

## Related Documentation

- [egg Framework Documentation](../../docs/)
- [servicex Module](../../servicex/README.md)
- [connectx Module](../../connectx/README.md)
- [Connect Protocol](https://connectrpc.com/)

