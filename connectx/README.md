# 🔗 ConnectX Package

The `connectx` package provides Connect protocol support and unified interceptors for the EggyByte framework.

## Overview

This package extends the Connect protocol with a comprehensive interceptor stack that handles recovery, logging, tracing, metrics, and identity injection. It's designed to provide zero business intrusion while offering production-ready observability.

## Features

- **Connect protocol support** - Full Connect/gRPC-Web compatibility
- **Unified interceptor stack** - Recovery, logging, tracing, metrics, identity
- **Zero business intrusion** - Transparent request/response handling
- **Production-ready** - Built-in observability and error handling
- **Configurable** - Flexible header mapping and options
- **Performance optimized** - Minimal overhead and allocations

## Quick Start

```go
import "github.com/eggybyte-technology/egg/connectx"

func main() {
    // Create service
    service := &UserService{}
    
    // Setup interceptors
    interceptors := connectx.DefaultInterceptors(connectx.Options{
        Logger:            logger,
        SlowRequestMillis: 1000,
        PayloadAccounting: true,
    })
    
    // Create Connect handler
    path, handler := userv1connect.NewUserServiceHandler(
        service,
        connect.WithInterceptors(interceptors...),
    )
    
    // Register handler
    mux.Handle(path, handler)
}
```

## API Reference

### Types

#### Options

```go
type Options struct {
    Logger            log.Logger     // Logger for interceptor operations
    Otel              *obsx.Provider // OpenTelemetry provider (optional)
    Headers           HeaderMapping  // Header mapping configuration
    WithRequestBody   bool           // Log request body (default: false)
    WithResponseBody  bool           // Log response body (default: false)
    SlowRequestMillis int64          // Slow request threshold in milliseconds
    PayloadAccounting bool           // Track inbound/outbound payload sizes
}
```

#### HeaderMapping

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
```

### Functions

```go
// DefaultInterceptors returns a set of interceptors with the given options
func DefaultInterceptors(opts Options) []connect.Interceptor

// DefaultHeaderMapping returns the default header mapping for Higress
func DefaultHeaderMapping() HeaderMapping
```

## Usage Examples

### Basic Service Setup

```go
func main() {
    // Create service
    service := &UserService{}
    
    // Setup interceptors
    interceptors := connectx.DefaultInterceptors(connectx.Options{
        Logger:            logger,
        SlowRequestMillis: 1000,
        PayloadAccounting: true,
    })
    
    // Create Connect handler
    path, handler := userv1connect.NewUserServiceHandler(
        service,
        connect.WithInterceptors(interceptors...),
    )
    
    // Register handler
    mux := http.NewServeMux()
    mux.Handle(path, handler)
    
    // Start server
    err := runtimex.Run(ctx, cancel, runtimex.Options{
        Logger: logger,
        HTTP: &runtimex.HTTPOptions{
            Addr: ":8080",
            H2C:  true,
            Mux:  mux,
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
}
```

### Custom Header Mapping

```go
func main() {
    // Create custom header mapping
    customHeaders := connectx.HeaderMapping{
        RequestID:     "X-Custom-Request-Id",
        InternalToken: "X-Custom-Token",
        UserID:        "X-Custom-User-Id",
        UserName:      "X-Custom-User-Name",
        Roles:         "X-Custom-Roles",
        RealIP:        "X-Custom-Real-IP",
        ForwardedFor:  "X-Custom-Forwarded-For",
        UserAgent:     "X-Custom-User-Agent",
    }
    
    // Setup interceptors with custom headers
    interceptors := connectx.DefaultInterceptors(connectx.Options{
        Logger:            logger,
        Headers:           customHeaders,
        SlowRequestMillis: 500,
        PayloadAccounting: true,
    })
    
    // Create Connect handler
    path, handler := userv1connect.NewUserServiceHandler(
        service,
        connect.WithInterceptors(interceptors...),
    )
    
    // Register handler
    mux.Handle(path, handler)
}
```

### With OpenTelemetry

```go
func main() {
    // Initialize OpenTelemetry
    otelProvider, err := obsx.NewProvider(ctx, logger, obsx.Options{
        ServiceName:    "user-service",
        ServiceVersion: "1.0.0",
        Environment:    "production",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Setup interceptors with OpenTelemetry
    interceptors := connectx.DefaultInterceptors(connectx.Options{
        Logger:            logger,
        Otel:              otelProvider,
        SlowRequestMillis: 1000,
        PayloadAccounting: true,
    })
    
    // Create Connect handler
    path, handler := userv1connect.NewUserServiceHandler(
        service,
        connect.WithInterceptors(interceptors...),
    )
    
    // Register handler
    mux.Handle(path, handler)
}
```

### Debug Mode

```go
func main() {
    // Setup interceptors for debugging
    interceptors := connectx.DefaultInterceptors(connectx.Options{
        Logger:            logger,
        WithRequestBody:   true,  // Log request bodies
        WithResponseBody:  true,  // Log response bodies
        SlowRequestMillis: 100,   // Lower threshold for debugging
        PayloadAccounting: true,
    })
    
    // Create Connect handler
    path, handler := userv1connect.NewUserServiceHandler(
        service,
        connect.WithInterceptors(interceptors...),
    )
    
    // Register handler
    mux.Handle(path, handler)
}
```

## Service Implementation

### Basic Service

```go
type UserService struct {
    logger log.Logger
    repo   UserRepository
}

func (s *UserService) GetUser(ctx context.Context, req *connect.Request[GetUserRequest]) (*connect.Response[GetUserResponse], error) {
    // User information is automatically injected by interceptors
    if user, ok := identity.UserFrom(ctx); ok {
        s.logger.Info("GetUser called", log.Str("user_id", user.UserID))
    }
    
    // Business logic
    user, err := s.repo.GetUser(ctx, req.Msg.UserId)
    if err != nil {
        return nil, connect.NewError(connect.CodeNotFound, err)
    }
    
    return connect.NewResponse(&GetUserResponse{User: user}), nil
}
```

### With Permission Checks

```go
func (s *UserService) DeleteUser(ctx context.Context, req *connect.Request[DeleteUserRequest]) (*connect.Response[DeleteUserResponse], error) {
    // Check permissions
    if !identity.HasRole(ctx, "admin") {
        return nil, connect.NewError(connect.CodePermissionDenied, errors.New("PERMISSION_DENIED", "admin role required"))
    }
    
    // Business logic
    err := s.repo.DeleteUser(ctx, req.Msg.UserId)
    if err != nil {
        return nil, connect.NewError(connect.CodeInternal, err)
    }
    
    return connect.NewResponse(&DeleteUserResponse{}), nil
}
```

### With Request Validation

```go
func (s *UserService) CreateUser(ctx context.Context, req *connect.Request[CreateUserRequest]) (*connect.Response[CreateUserResponse], error) {
    // Validate request
    if req.Msg.User == nil {
        return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("VALIDATION_ERROR", "user is required"))
    }
    
    if utils.IsEmpty(req.Msg.User.Email) {
        return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("VALIDATION_ERROR", "email is required"))
    }
    
    if !utils.IsValidEmail(req.Msg.User.Email) {
        return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("VALIDATION_ERROR", "invalid email format"))
    }
    
    // Business logic
    user, err := s.repo.CreateUser(ctx, req.Msg.User)
    if err != nil {
        return nil, connect.NewError(connect.CodeInternal, err)
    }
    
    return connect.NewResponse(&CreateUserResponse{User: user}), nil
}
```

## Interceptor Details

### Recovery Interceptor

Automatically recovers from panics and returns proper error responses:

```go
func (s *UserService) RiskyMethod(ctx context.Context, req *connect.Request[RiskyRequest]) (*connect.Response[RiskyResponse], error) {
    // This might panic
    result := riskyOperation(req.Msg.Data)
    
    return connect.NewResponse(&RiskyResponse{Result: result}), nil
}
```

### Logging Interceptor

Provides structured logging for all requests:

```go
// Logs include:
// - Request method and path
// - User information (if available)
// - Request duration
// - Response status
// - Error details (if any)
```

### Tracing Interceptor

Integrates with OpenTelemetry for distributed tracing:

```go
// Traces include:
// - Request span
// - Database operations
// - External service calls
// - Error propagation
```

### Metrics Interceptor

Collects Prometheus metrics:

```go
// Metrics include:
// - Request count
// - Request duration
// - Error rate
// - Payload sizes
```

### Identity Interceptor

Injects user identity and request metadata:

```go
// Injects:
// - User information from headers
// - Request metadata
// - Internal service tokens
// - Client information
```

## Configuration

### Environment Variables

```bash
# Service configuration
SERVICE_NAME=user-service
SERVICE_VERSION=1.0.0
ENV=production

# HTTP configuration
HTTP_PORT=:8080
HEALTH_PORT=:8081
METRICS_PORT=:9091

# Observability
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
ENABLE_DEBUG_LOGS=false
ENABLE_METRICS=true
ENABLE_TRACING=true
```

### Configuration Integration

```go
type AppConfig struct {
    configx.BaseConfig
    
    // Connect-specific configuration
    Connect ConnectConfig
}

type ConnectConfig struct {
    SlowRequestMillis int64 `env:"SLOW_REQUEST_MILLIS" default:"1000"`
    PayloadAccounting bool  `env:"PAYLOAD_ACCOUNTING" default:"true"`
    WithRequestBody   bool  `env:"WITH_REQUEST_BODY" default:"false"`
    WithResponseBody  bool  `env:"WITH_RESPONSE_BODY" default:"false"`
}

func main() {
    // Load configuration
    var cfg AppConfig
    if err := configManager.Bind(&cfg); err != nil {
        log.Fatal(err)
    }
    
    // Setup interceptors with configuration
    interceptors := connectx.DefaultInterceptors(connectx.Options{
        Logger:            logger,
        SlowRequestMillis: cfg.Connect.SlowRequestMillis,
        PayloadAccounting: cfg.Connect.PayloadAccounting,
        WithRequestBody:   cfg.Connect.WithRequestBody,
        WithResponseBody:  cfg.Connect.WithResponseBody,
    })
    
    // Create Connect handler
    path, handler := userv1connect.NewUserServiceHandler(
        service,
        connect.WithInterceptors(interceptors...),
    )
    
    // Register handler
    mux.Handle(path, handler)
}
```

## Testing

```go
func TestConnectService(t *testing.T) {
    // Create test service
    service := &TestUserService{}
    
    // Setup interceptors
    interceptors := connectx.DefaultInterceptors(connectx.Options{
        Logger:            &TestLogger{},
        SlowRequestMillis: 1000,
        PayloadAccounting: true,
    })
    
    // Create Connect handler
    path, handler := userv1connect.NewUserServiceHandler(
        service,
        connect.WithInterceptors(interceptors...),
    )
    
    // Create test server
    mux := http.NewServeMux()
    mux.Handle(path, handler)
    
    server := httptest.NewServer(mux)
    defer server.Close()
    
    // Test client
    client := userv1connect.NewUserServiceClient(
        http.DefaultClient,
        server.URL,
    )
    
    // Test request
    resp, err := client.GetUser(context.Background(), connect.NewRequest(&GetUserRequest{
        UserId: "user-123",
    }))
    
    assert.NoError(t, err)
    assert.NotNil(t, resp)
}

type TestUserService struct{}

func (s *TestUserService) GetUser(ctx context.Context, req *connect.Request[GetUserRequest]) (*connect.Response[GetUserResponse], error) {
    return connect.NewResponse(&GetUserResponse{
        User: &User{Id: req.Msg.UserId, Name: "Test User"},
    }), nil
}
```

## Best Practices

### 1. Use Structured Errors

```go
func (s *UserService) GetUser(ctx context.Context, req *connect.Request[GetUserRequest]) (*connect.Response[GetUserResponse], error) {
    user, err := s.repo.GetUser(ctx, req.Msg.UserId)
    if err != nil {
        // Use structured errors
        return nil, connect.NewError(connect.CodeNotFound, errors.New("NOT_FOUND", "user not found"))
    }
    
    return connect.NewResponse(&GetUserResponse{User: user}), nil
}
```

### 2. Validate Input

```go
func (s *UserService) CreateUser(ctx context.Context, req *connect.Request[CreateUserRequest]) (*connect.Response[CreateUserResponse], error) {
    // Validate input
    if req.Msg.User == nil {
        return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("VALIDATION_ERROR", "user is required"))
    }
    
    if utils.IsEmpty(req.Msg.User.Email) {
        return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("VALIDATION_ERROR", "email is required"))
    }
    
    // Business logic
    user, err := s.repo.CreateUser(ctx, req.Msg.User)
    if err != nil {
        return nil, connect.NewError(connect.CodeInternal, err)
    }
    
    return connect.NewResponse(&CreateUserResponse{User: user}), nil
}
```

### 3. Check Permissions

```go
func (s *UserService) DeleteUser(ctx context.Context, req *connect.Request[DeleteUserRequest]) (*connect.Response[DeleteUserResponse], error) {
    // Check permissions
    if !identity.HasRole(ctx, "admin") {
        return nil, connect.NewError(connect.CodePermissionDenied, errors.New("PERMISSION_DENIED", "admin role required"))
    }
    
    // Business logic
    err := s.repo.DeleteUser(ctx, req.Msg.UserId)
    if err != nil {
        return nil, connect.NewError(connect.CodeInternal, err)
    }
    
    return connect.NewResponse(&DeleteUserResponse{}), nil
}
```

### 4. Use Context

```go
func (s *UserService) GetUser(ctx context.Context, req *connect.Request[GetUserRequest]) (*connect.Response[GetUserResponse], error) {
    // Use context for cancellation and timeouts
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    user, err := s.repo.GetUser(ctx, req.Msg.UserId)
    if err != nil {
        return nil, connect.NewError(connect.CodeNotFound, err)
    }
    
    return connect.NewResponse(&GetUserResponse{User: user}), nil
}
```

## Thread Safety

All functions in this package are safe for concurrent use. The interceptor stack is designed to handle concurrent requests safely.

## Dependencies

- **Go 1.21+** required
- **Connect** - Protocol support
- **OpenTelemetry** - Observability (optional)
- **Standard library** - Core functionality

## Version Compatibility

- **Go 1.21+** required
- **API Stability**: Evolving (L3 module)
- **Breaking Changes**: Possible in minor versions

## Contributing

Contributions are welcome! Please see the main project [Contributing Guide](../CONTRIBUTING.md) for details.

## License

This package is part of the EggyByte framework and is licensed under the MIT License.