# egg/connectx

## Overview

`connectx` provides a unified Connect-RPC interceptor stack for egg microservices.
It delivers production-ready interceptors for timeout control, structured logging,
error mapping, and OpenTelemetry integration with a single function call.

## Key Features

- Timeout enforcement with header-based override
- Structured request/response logging with correlation
- Automatic error mapping from `core/errors` to Connect codes
- OpenTelemetry tracing and metrics integration
- Panic recovery with graceful error responses
- Identity injection from HTTP headers
- Configurable slow request detection
- Payload size accounting

## Dependencies

Layer: **L3 (Runtime Communication Layer)**  
Depends on: `core/log`, `core/identity`, `core/errors`, `logx`, `obsx`

## Installation

```bash
go get github.com/eggybyte-technology/egg/connectx@latest
```

## Basic Usage

```go
import (
    "connectrpc.com/connect"
    "github.com/eggybyte-technology/egg/connectx"
    userv1connect "myapp/gen/go/user/v1/userv1connect"
)

func main() {
    // Create default interceptor stack
    interceptors := connectx.DefaultInterceptors(connectx.Options{
        Logger:            logger,
        Otel:              otelProvider,
        SlowRequestMillis: 1000,
    })
    
    // Create Connect handler with interceptors
    path, handler := userv1connect.NewUserServiceHandler(
        service,
        connect.WithInterceptors(interceptors...),
    )
    
    mux.Handle(path, handler)
}
```

## Configuration Options

| Option                | Type             | Description                                |
| --------------------- | ---------------- | ------------------------------------------ |
| `Logger`              | `log.Logger`     | Logger for interceptor operations          |
| `Otel`                | `*obsx.Provider` | OpenTelemetry provider (nil disables)      |
| `Headers`             | `HeaderMapping`  | Header mapping configuration               |
| `WithRequestBody`     | `bool`           | Log request body (default: false)          |
| `WithResponseBody`    | `bool`           | Log response body (default: false)         |
| `SlowRequestMillis`   | `int64`          | Slow request threshold in ms               |
| `PayloadAccounting`   | `bool`           | Track payload sizes                        |
| `DefaultTimeoutMs`    | `int64`          | Default RPC timeout in ms                  |
| `EnableTimeout`       | `bool`           | Enable timeout interceptor                 |

## API Reference

### Main Function

```go
// DefaultInterceptors returns a set of interceptors with the given options
func DefaultInterceptors(opts Options) []connect.Interceptor
```

### Header Mapping

```go
type HeaderMapping struct {
    RequestID     string // "X-Request-Id"
    InternalToken string // "X-Internal-Token"
    UserID        string // "X-User-Id"
    UserName      string // "X-User-Name"
    Roles         string // "X-User-Roles"
    RealIP        string // "X-Real-IP"
    ForwardedFor  string // "X-Forwarded-For"
    UserAgent     string // "User-Agent"
}

// DefaultHeaderMapping returns the default header mapping for Higress
func DefaultHeaderMapping() HeaderMapping
```

### Utility Function

```go
// Bind is a utility function to bind Connect handlers to HTTP mux
func Bind(mux *http.ServeMux, path string, handler http.Handler)
```

## Architecture

The connectx module provides a unified interceptor stack:

```
connectx/
├── connectx.go          # Public API (~140 lines)
│   ├── Options          # Configuration structs
│   ├── HeaderMapping    # Header configuration
│   └── DefaultInterceptors()  # Main entry point
└── internal/
    └── interceptors.go  # Interceptor implementations
        ├── RecoveryInterceptor()
        ├── TimeoutInterceptor()
        ├── IdentityInterceptor()
        ├── ErrorMappingInterceptor()
        └── LoggingInterceptor()
```

**Interceptor Order** (optimized for performance and correctness):
1. **Recovery** - Panic handling
2. **Timeout** - Deadline enforcement
3. **Identity** - Header extraction
4. **Error Mapping** - Error code translation
5. **Logging** - Request/response logging

## Example: Complete Service Setup

```go
package main

import (
    "context"
    "net/http"
    
    "connectrpc.com/connect"
    "github.com/eggybyte-technology/egg/connectx"
    "github.com/eggybyte-technology/egg/logx"
    "github.com/eggybyte-technology/egg/obsx"
    userv1connect "myapp/gen/go/user/v1/userv1connect"
)

func main() {
    ctx := context.Background()
    
    // Create logger
    logger := logx.New(
        logx.WithFormat(logx.FormatLogfmt),
    )
    
    // Create OpenTelemetry provider
    otelProvider, err := obsx.NewProvider(ctx, obsx.Options{
        ServiceName:    "user-service",
        ServiceVersion: "1.0.0",
        OTLPEndpoint:   "otel-collector:4317",
    })
    if err != nil {
        logger.Error(err, "failed to create otel provider")
    }
    
    // Create interceptors
    interceptors := connectx.DefaultInterceptors(connectx.Options{
        Logger:            logger,
        Otel:              otelProvider,
        SlowRequestMillis: 1000,
        DefaultTimeoutMs:  30000,
        PayloadAccounting: true,
    })
    
    // Create service
    service := &UserService{logger: logger}
    
    // Create Connect handler
    path, handler := userv1connect.NewUserServiceHandler(
        service,
        connect.WithInterceptors(interceptors...),
    )
    
    // Register handler
    mux := http.NewServeMux()
    mux.Handle(path, handler)
    
    // Start server
    http.ListenAndServe(":8080", mux)
}
```

## Example: Custom Header Mapping

```go
// Custom header mapping for your gateway
headers := connectx.HeaderMapping{
    RequestID:     "X-Trace-Id",        // Custom trace ID header
    InternalToken: "X-Internal-Auth",   // Custom auth header
    UserID:        "X-Auth-User-Id",    // Custom user ID header
    UserName:      "X-Auth-User-Name",
    Roles:         "X-Auth-Roles",
    RealIP:        "X-Forwarded-For",
    ForwardedFor:  "X-Forwarded-For",
    UserAgent:     "User-Agent",
}

interceptors := connectx.DefaultInterceptors(connectx.Options{
    Logger:  logger,
    Headers: headers,
})
```

## Interceptor Details

### Recovery Interceptor

Catches panics and converts them to proper Connect errors:

```go
func MyHandler(ctx context.Context, req *connect.Request[Msg]) (*connect.Response[Msg], error) {
    panic("something went wrong")  // Caught by recovery interceptor
    // Client receives: code=Internal, message="internal server error"
}
```

### Timeout Interceptor

Enforces request timeouts with header override support:

```go
// Server-side default timeout: 30s
interceptors := connectx.DefaultInterceptors(connectx.Options{
    DefaultTimeoutMs: 30000,
})

// Client can override (if allowed):
// Header: X-Timeout-Ms: 5000  (5 seconds)
```

### Identity Interceptor

Extracts identity from headers and injects into context:

```go
func MyHandler(ctx context.Context, req *connect.Request[Msg]) (*connect.Response[Msg], error) {
    // Extract user from context
    user, ok := identity.UserFrom(ctx)
    if ok {
        log.Info("user request", "user_id", user.UserID, "roles", user.Roles)
    }
    
    // Extract request metadata
    meta, ok := identity.MetaFrom(ctx)
    if ok {
        log.Info("request metadata", "request_id", meta.RequestID)
    }
    
    return connect.NewResponse(&Msg{}), nil
}
```

### Error Mapping Interceptor

Maps `core/errors` to Connect codes:

```go
import "github.com/eggybyte-technology/egg/core/errors"

func MyHandler(ctx context.Context, req *connect.Request[Msg]) (*connect.Response[Msg], error) {
    // Return domain error
    return nil, errors.New("NOT_FOUND", "user not found")
    // Client receives: code=NotFound, message="user not found"
    
    // Or validation error
    return nil, errors.New("INVALID_ARGUMENT", "email is required")
    // Client receives: code=InvalidArgument, message="email is required"
}
```

### Logging Interceptor

Logs requests and responses with structured fields:

```go
// Logged fields:
// - service, method, procedure
// - request_id, user_id (from context)
// - duration_ms, status_code
// - payload_in_bytes, payload_out_bytes (if enabled)
// - error (if request failed)

// Example log output:
// level=INFO msg="rpc completed" duration_ms=45 method=GetUser payload_in_bytes=128 payload_out_bytes=512 procedure=/user.v1.UserService/GetUser request_id=req-123 service=user.v1.UserService status_code=0 user_id=u-456
```

## Integration with servicex

connectx interceptors are automatically configured by servicex:

```go
import "github.com/eggybyte-technology/egg/servicex"

func register(app *servicex.App) error {
    // Get pre-configured interceptors
    interceptors := app.Interceptors()
    
    // Use with your handlers
    path, handler := userv1connect.NewUserServiceHandler(
        service,
        connect.WithInterceptors(interceptors...),
    )
    
    app.Mux().Handle(path, handler)
    return nil
}
```

## Observability

### Logging

All RPC calls are logged with structured fields:

```
level=INFO msg="rpc completed" duration_ms=45 method=GetUser procedure=/user.v1.UserService/GetUser request_id=req-123 status_code=0
```

Slow requests are logged at WARN level:

```
level=WARN msg="slow rpc" duration_ms=1500 method=GetUser procedure=/user.v1.UserService/GetUser request_id=req-123 threshold_ms=1000
```

### Tracing

When OpenTelemetry provider is configured, spans are automatically created:

- Span name: `{service}/{method}`
- Attributes: service, method, status_code, error (if any)
- Duration: Automatically recorded

### Metrics

When payload accounting is enabled:

- Request payload size (bytes)
- Response payload size (bytes)
- Request duration (milliseconds)

## Error Code Mapping

| core/errors Code      | Connect Code           | HTTP Status |
| --------------------- | ---------------------- | ----------- |
| `INVALID_ARGUMENT`    | `InvalidArgument`      | 400         |
| `NOT_FOUND`           | `NotFound`             | 404         |
| `ALREADY_EXISTS`      | `AlreadyExists`        | 409         |
| `PERMISSION_DENIED`   | `PermissionDenied`     | 403         |
| `UNAUTHENTICATED`     | `Unauthenticated`      | 401         |
| `RESOURCE_EXHAUSTED`  | `ResourceExhausted`    | 429         |
| `UNIMPLEMENTED`       | `Unimplemented`        | 501         |
| `INTERNAL`            | `Internal`             | 500         |
| `UNAVAILABLE`         | `Unavailable`          | 503         |
| `DEADLINE_EXCEEDED`   | `DeadlineExceeded`     | 504         |

## Best Practices

1. **Always use default interceptors** - They provide essential production features
2. **Set appropriate timeouts** - Prevent hanging requests
3. **Enable slow request logging** - Identify performance issues
4. **Use structured errors** - Better error handling on client side
5. **Configure header mapping** - Match your gateway configuration
6. **Enable tracing in production** - Essential for debugging distributed systems
7. **Disable body logging in production** - Reduces log volume, prevents sensitive data leaks

## Testing

For testing, you can create interceptors without logger or OpenTelemetry:

```go
func TestMyHandler(t *testing.T) {
    // Create test interceptors (minimal)
    interceptors := connectx.DefaultInterceptors(connectx.Options{
        SlowRequestMillis: 1000,
    })
    
    // Create test client
    client := userv1connect.NewUserServiceClient(
        http.DefaultClient,
        "http://localhost:8080",
        connect.WithInterceptors(interceptors...),
    )
    
    // Test handler
    resp, err := client.GetUser(context.Background(), connect.NewRequest(&userv1.GetUserRequest{
        UserId: "test-user",
    }))
    
    require.NoError(t, err)
    assert.NotNil(t, resp)
}
```

## Stability

**Status**: Stable  
**Layer**: L3 (Runtime Communication)  
**API Guarantees**: Backward-compatible changes only

The connectx module is production-ready and follows semantic versioning.

## License

This package is part of the egg framework and is licensed under the MIT License.
See the root LICENSE file for details.
