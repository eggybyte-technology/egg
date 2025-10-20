# ConnectX Module

<div align="center">

**Connect protocol binding with unified interceptors for Egg services**

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg)](LICENSE)

</div>

## 📦 Overview

The `connectx` module provides Connect protocol binding with unified interceptors for Egg services. It offers a complete interceptor stack including recovery, logging, tracing, metrics, identity injection, and error mapping.

## ✨ Features

- 🚀 **Connect Protocol** - Full Connect protocol support
- 🛡️ **Unified Interceptors** - Complete interceptor stack
- 📊 **Automatic Tracing** - OpenTelemetry integration
- 📈 **Metrics Collection** - Prometheus metrics
- 🔐 **Identity Injection** - Automatic user identity extraction
- 🚨 **Error Mapping** - Structured error handling
- 🔄 **Recovery** - Panic recovery and error handling
- 📝 **Structured Logging** - Context-aware logging

## 🏗️ Architecture

```
connectx/
├── connectx.go           # Main Connect binding
├── internal/
│   └── interceptors.go   # Interceptor implementations
└── connectx_test.go      # Tests
```

## 🚀 Quick Start

### Installation

```bash
go get github.com/eggybyte-technology/egg/connectx@latest
```

### Basic Usage

```go
package main

import (
    "context"
    "net/http"

    "github.com/eggybyte-technology/egg/connectx"
    "github.com/eggybyte-technology/egg/core/log"
    "github.com/eggybyte-technology/egg/obsx"
    "connectrpc.com/connect"
)

func main() {
    // Create logger
    logger := &YourLogger{} // Implement log.Logger interface

    // Create OpenTelemetry provider
    otel, _ := obsx.NewProvider(context.Background(), obsx.Options{
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
    })
    defer otel.Shutdown(context.Background())

    // Create interceptor options
    opts := connectx.Options{
        Logger: logger,
        Otel:   otel,
    }

    // Get default interceptors
    interceptors := connectx.DefaultInterceptors(opts)

    // Create Connect service
    service := &MyService{}
    handler := NewMyServiceHandler(service, connect.WithInterceptors(interceptors...))

    // Set up HTTP mux
    mux := http.NewServeMux()
    mux.Handle(handler)

    // Start server
    http.ListenAndServe(":8080", mux)
}
```

## 📖 API Reference

### Interceptor Options

```go
type Options struct {
    Logger log.Logger
    Otel    obsx.Provider
}

type InterceptorOptions struct {
    Logger log.Logger
    Otel    obsx.Provider
    // Additional options
}
```

### Main Functions

```go
// DefaultInterceptors returns the default interceptor stack
func DefaultInterceptors(opts Options) []connect.Interceptor

// NewRecoveryInterceptor creates a recovery interceptor
func NewRecoveryInterceptor(opts InterceptorOptions) connect.Interceptor

// NewLoggingInterceptor creates a logging interceptor
func NewLoggingInterceptor(opts InterceptorOptions) connect.Interceptor

// NewTracingInterceptor creates a tracing interceptor
func NewTracingInterceptor(opts InterceptorOptions) connect.Interceptor

// NewMetricsInterceptor creates a metrics interceptor
func NewMetricsInterceptor(opts InterceptorOptions) connect.Interceptor

// NewIdentityInterceptor creates an identity injection interceptor
func NewIdentityInterceptor(opts InterceptorOptions) connect.Interceptor

// NewErrorInterceptor creates an error mapping interceptor
func NewErrorInterceptor(opts InterceptorOptions) connect.Interceptor
```

## 🔧 Interceptors

### Recovery Interceptor

Handles panics and converts them to Connect errors:

```go
interceptor := connectx.NewRecoveryInterceptor(connectx.InterceptorOptions{
    Logger: logger,
})
```

### Logging Interceptor

Provides structured logging for all Connect requests:

```go
interceptor := connectx.NewLoggingInterceptor(connectx.InterceptorOptions{
    Logger: logger,
})
```

**Log Fields:**
- `trace_id` - OpenTelemetry trace ID
- `span_id` - OpenTelemetry span ID
- `req_id` - Request ID
- `rpc_system` - RPC system (connect)
- `rpc_service` - RPC service name
- `rpc_method` - RPC method name
- `status` - Response status
- `latency_ms` - Request latency
- `remote_ip` - Client IP address
- `user_agent` - Client user agent

### Tracing Interceptor

Provides OpenTelemetry tracing for Connect requests:

```go
interceptor := connectx.NewTracingInterceptor(connectx.InterceptorOptions{
    Otel: otel,
})
```

**Trace Attributes:**
- `rpc.system` - RPC system
- `rpc.service` - RPC service
- `rpc.method` - RPC method
- `rpc.status_code` - RPC status code
- `user.id` - User ID (if available)

### Metrics Interceptor

Collects Prometheus metrics for Connect requests:

```go
interceptor := connectx.NewMetricsInterceptor(connectx.InterceptorOptions{
    Otel: otel,
})
```

**Metrics:**
- `rpc_server_duration_seconds` - Request duration histogram
- `rpc_server_requests_total` - Request counter
- `rpc_server_payload_bytes` - Payload size counter

### Identity Interceptor

Extracts user identity from request headers and injects into context:

```go
interceptor := connectx.NewIdentityInterceptor(connectx.InterceptorOptions{
    Logger: logger,
})
```

**Supported Headers:**
- `X-User-ID` - User ID
- `X-Request-ID` - Request ID
- `X-Trace-ID` - Trace ID
- `Authorization` - Bearer token

### Error Interceptor

Maps internal errors to Connect errors:

```go
interceptor := connectx.NewErrorInterceptor(connectx.InterceptorOptions{
    Logger: logger,
})
```

## 🛠️ Advanced Usage

### Custom Interceptor

```go
func customInterceptor() connect.Interceptor {
    return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
        return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
            // Pre-processing
            log.Info("Processing request", "method", req.Spec().Procedure)
            
            // Call next interceptor
            resp, err := next(ctx, req)
            
            // Post-processing
            if err != nil {
                log.Error("Request failed", "error", err)
            } else {
                log.Info("Request completed")
            }
            
            return resp, err
        }
    })
}
```

### Service Implementation

```go
type MyService struct{}

func (s *MyService) GetUser(ctx context.Context, req *connect.Request[GetUserRequest]) (*connect.Response[GetUserResponse], error) {
    // Extract user ID from context
    userID, ok := identity.UserIDFromContext(ctx)
    if !ok {
        return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
    }
    
    // Your business logic
    user, err := s.userStore.GetUser(ctx, userID)
    if err != nil {
        return nil, connect.NewError(connect.CodeNotFound, err)
    }
    
    return connect.NewResponse(&GetUserResponse{User: user}), nil
}
```

### Error Handling

```go
func (s *MyService) CreateUser(ctx context.Context, req *connect.Request[CreateUserRequest]) (*connect.Response[CreateUserResponse], error) {
    // Validate request
    if req.Msg.Email == "" {
        return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email is required"))
    }
    
    // Business logic
    user, err := s.userStore.CreateUser(ctx, req.Msg)
    if err != nil {
        // Map internal errors to Connect errors
        switch {
        case errors.IsCode(err, "DUPLICATE_EMAIL"):
            return nil, connect.NewError(connect.CodeAlreadyExists, err)
        case errors.IsCode(err, "VALIDATION_FAILED"):
            return nil, connect.NewError(connect.CodeInvalidArgument, err)
        default:
            return nil, connect.NewError(connect.CodeInternal, err)
        }
    }
    
    return connect.NewResponse(&CreateUserResponse{User: user}), nil
}
```

## 🔧 Configuration

### Environment Variables

```bash
# Service identification
export SERVICE_NAME="my-service"
export SERVICE_VERSION="1.0.0"

# OpenTelemetry
export OTEL_EXPORTER_OTLP_ENDPOINT="http://otel-collector:4317"

# Logging
export LOG_LEVEL="info"
```

### Custom Interceptor Stack

```go
// Create custom interceptor stack
interceptors := []connect.Interceptor{
    connectx.NewRecoveryInterceptor(opts),
    connectx.NewLoggingInterceptor(opts),
    connectx.NewTracingInterceptor(opts),
    connectx.NewMetricsInterceptor(opts),
    connectx.NewIdentityInterceptor(opts),
    connectx.NewErrorInterceptor(opts),
    customInterceptor(), // Your custom interceptor
}
```

## 🧪 Testing

Run tests:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

## 📈 Test Coverage

| Component | Coverage |
|-----------|----------|
| ConnectX | 92.9% |

## 🔍 Troubleshooting

### Common Issues

1. **Missing OpenTelemetry Provider**
   ```go
   // Ensure OpenTelemetry provider is created
   otel, err := obsx.NewProvider(ctx, obsx.Options{
       ServiceName:    "my-service",
       ServiceVersion: "1.0.0",
   })
   if err != nil {
       log.Fatal("Failed to create OpenTelemetry provider:", err)
   }
   ```

2. **Identity Not Extracted**
   ```go
   // Ensure identity headers are set
   req.Header.Set("X-User-ID", "user123")
   req.Header.Set("X-Request-ID", "req456")
   ```

3. **Metrics Not Collected**
   ```go
   // Ensure metrics interceptor is included
   interceptors := connectx.DefaultInterceptors(opts)
   ```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Create a Pull Request

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](../../LICENSE) file for details.

---

<div align="center">

**Built with ❤️ by EggyByte Technology**

</div>
