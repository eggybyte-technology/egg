# Minimal Connect Service Example

This example demonstrates a comprehensive Connect service using the egg framework.

## Features

- **Connect Service**: Simple greeting service with Connect protocol
- **Configuration Management**: Uses `configx` for configuration with hot reloading
- **Observability**: OpenTelemetry integration for tracing and metrics
- **Runtime Management**: Uses `runtimex` for lifecycle management
- **Unified Interceptors**: Recovery, logging, tracing, identity injection, error mapping
- **Graceful Shutdown**: Proper signal handling and cleanup

## Configuration

The service supports the following environment variables:

### Base Configuration (from configx.BaseConfig)
- `SERVICE_NAME`: Service name (default: "app")
- `SERVICE_VERSION`: Service version (default: "0.0.0")
- `ENV`: Environment (default: "dev")
- `HTTP_PORT`: HTTP server port (default: ":8080")
- `HEALTH_PORT`: Health check port (default: ":8081")
- `METRICS_PORT`: Metrics port (default: ":9091")
- `OTEL_EXPORTER_OTLP_ENDPOINT`: OpenTelemetry endpoint
- `APP_CONFIGMAP_NAME`: ConfigMap name for dynamic configuration

### Application-Specific Configuration
- `SLOW_REQUEST_MILLIS`: Slow request threshold in milliseconds (default: 1000)
- `RATE_LIMIT_QPS`: Rate limit QPS (default: 100)
- `ENABLE_DEBUG_LOGS`: Enable debug logging (default: false)

## Running the Example

### Basic Usage

```bash
# Build the service
go build -o minimal-connect-service .

# Run with default configuration
./minimal-connect-service
```

### With Custom Configuration

```bash
# Set environment variables
export SERVICE_NAME="greeter-service"
export SERVICE_VERSION="1.0.0"
export ENV="production"
export HTTP_PORT=":8080"
export HEALTH_PORT=":8081"
export METRICS_PORT=":9091"
export SLOW_REQUEST_MILLIS="500"
export ENABLE_DEBUG_LOGS="true"

# Run the service
./minimal-connect-service
```

### With OpenTelemetry

```bash
# Set OpenTelemetry endpoint
export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317"

# Run the service
./minimal-connect-service
```

### With ConfigMap (Kubernetes)

```bash
# Set ConfigMap name for dynamic configuration
export APP_CONFIGMAP_NAME="greeter-config"
export NAMESPACE="default"

# Run the service
./minimal-connect-service
```

## API Endpoints

### Connect Service
- **Endpoint**: `POST /greet.v1.GreeterService/SayHello`
- **Protocol**: Connect (HTTP/2)
- **Request**: `{"name": "World"}`
- **Response**: `{"message": "Hello, World!"}`

### Health Check
- **Endpoint**: `GET /health`
- **Response**: `200 OK` with body `OK`

### Metrics
- **Endpoint**: `GET /metrics`
- **Response**: `200 OK` with metrics data

## Testing the Service

### Using curl

```bash
# Health check
curl http://localhost:8081/health

# Metrics
curl http://localhost:9091/metrics

# Connect service (requires Connect client)
curl -X POST http://localhost:8080/greet.v1.GreeterService/SayHello \
  -H "Content-Type: application/json" \
  -H "Connect-Protocol-Version: 1" \
  -d '{"name": "World"}'
```

### Using Connect client

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    
    "connectrpc.com/connect"
)

func main() {
    client := connect.NewHTTPClient(http.DefaultClient, "http://localhost:8080")
    
    req := connect.NewRequest(&HelloRequest{Name: "World"})
    resp, err := client.CallUnary(context.Background(), req)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Response: %s\n", resp.Msg.Message)
}
```

## Configuration Hot Reloading

The service supports configuration hot reloading through ConfigMaps:

1. Create a ConfigMap with dynamic configuration:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: greeter-config
  namespace: default
data:
  SLOW_REQUEST_MILLIS: "500"
  RATE_LIMIT_QPS: "200"
  ENABLE_DEBUG_LOGS: "true"
```

2. Set the ConfigMap name in environment:
```bash
export APP_CONFIGMAP_NAME="greeter-config"
```

3. Run the service - it will automatically reload configuration when the ConfigMap changes.

## Observability

The service provides comprehensive observability:

### Metrics
- Request duration histogram
- Request count counter
- Payload size tracking
- Runtime metrics (if enabled)

### Tracing
- Distributed tracing with OpenTelemetry
- Configurable sampling ratio
- Service name and version tags

### Logging
- Structured logging with key-value pairs
- Request/response logging (configurable)
- Slow request detection
- Error logging with context

## Architecture

The example follows the egg framework architecture:

1. **Configuration Layer**: `configx` for unified configuration management
2. **Observability Layer**: `obsx` for OpenTelemetry integration
3. **Transport Layer**: `connectx` for Connect protocol and interceptors
4. **Runtime Layer**: `runtimex` for lifecycle management
5. **Core Layer**: `core` for logging, errors, and identity

## Development

### Prerequisites
- Go 1.21+
- Docker (for OpenTelemetry collector)
- Kubernetes cluster (for ConfigMap testing)

### Building
```bash
go build -o minimal-connect-service .
```

### Testing
```bash
go test ./...
```

### Running Tests
```bash
# Unit tests
go test -v

# Integration tests (requires dependencies)
go test -v -tags=integration
```

## Production Deployment

### Docker
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o minimal-connect-service .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/minimal-connect-service .
CMD ["./minimal-connect-service"]
```

### Kubernetes
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: greeter-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: greeter-service
  template:
    metadata:
      labels:
        app: greeter-service
    spec:
      containers:
      - name: greeter-service
        image: greeter-service:latest
        ports:
        - containerPort: 8080
        - containerPort: 8081
        - containerPort: 9091
        env:
        - name: SERVICE_NAME
          value: "greeter-service"
        - name: SERVICE_VERSION
          value: "1.0.0"
        - name: ENV
          value: "production"
        - name: APP_CONFIGMAP_NAME
          value: "greeter-config"
        - name: OTEL_EXPORTER_OTLP_ENDPOINT
          value: "http://otel-collector:4317"
```

## Troubleshooting

### Common Issues

1. **Port conflicts**: Ensure ports 8080, 8081, and 9091 are available
2. **OpenTelemetry connection**: Check if OTLP endpoint is reachable
3. **ConfigMap access**: Verify RBAC permissions for ConfigMap access
4. **Memory usage**: Monitor memory usage with runtime metrics

### Debug Mode

Enable debug logging:
```bash
export ENABLE_DEBUG_LOGS="true"
./minimal-connect-service
```

### Health Checks

Check service health:
```bash
curl http://localhost:8081/health
```

### Metrics

View metrics:
```bash
curl http://localhost:9091/metrics
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This example is part of the egg framework and follows the same license.
