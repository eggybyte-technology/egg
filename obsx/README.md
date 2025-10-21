# 📊 ObsX Package

The `obsx` package provides OpenTelemetry integration for the EggyByte framework.

## Overview

This package offers a comprehensive observability solution with metrics, tracing, and logging integration using OpenTelemetry. It's designed to be production-ready with minimal configuration and maximum observability.

## Features

- **OpenTelemetry integration** - Full OpenTelemetry support
- **Metrics collection** - Prometheus-compatible metrics
- **Distributed tracing** - Request tracing across services
- **Logging integration** - Structured logging with trace correlation
- **Zero configuration** - Works out of the box
- **Production ready** - Optimized for production environments

## Quick Start

```go
import "github.com/eggybyte-technology/egg/obsx"

func main() {
    // Initialize OpenTelemetry
    provider, err := obsx.NewProvider(ctx, logger, obsx.Options{
        ServiceName:    "user-service",
        ServiceVersion: "1.0.0",
        Environment:    "production",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Shutdown(ctx)
    
    // Use in Connect interceptors
    interceptors := connectx.DefaultInterceptors(connectx.Options{
        Logger: logger,
        Otel:   provider,
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

#### Provider

```go
type Provider struct {
    TracerProvider trace.TracerProvider
    MeterProvider  metric.MeterProvider
    Logger         log.Logger
}

// Shutdown shuts down the provider
func (p *Provider) Shutdown(ctx context.Context) error

// GetTracer returns a tracer
func (p *Provider) GetTracer(name string) trace.Tracer

// GetMeter returns a meter
func (p *Provider) GetMeter(name string) metric.Meter
```

#### Options

```go
type Options struct {
    ServiceName    string        // Service name for tracing
    ServiceVersion string        // Service version
    Environment    string        // Environment (dev, staging, production)
    Endpoint       string        // OTLP endpoint
    Insecure       bool          // Use insecure connection
    Headers        map[string]string // Additional headers
    Timeout        time.Duration // Export timeout
    BatchTimeout   time.Duration // Batch timeout
    BatchSize      int           // Batch size
}
```

### Functions

```go
// NewProvider creates a new OpenTelemetry provider
func NewProvider(ctx context.Context, logger log.Logger, opts Options) (*Provider, error)

// NewNoOpProvider creates a no-op provider for testing
func NewNoOpProvider() *Provider

// NewTracer creates a new tracer
func NewTracer(provider *Provider, name string) trace.Tracer

// NewMeter creates a new meter
func NewMeter(provider *Provider, name string) metric.Meter
```

## Usage Examples

### Basic Setup

```go
func main() {
    // Initialize OpenTelemetry
    provider, err := obsx.NewProvider(ctx, logger, obsx.Options{
        ServiceName:    "user-service",
        ServiceVersion: "1.0.0",
        Environment:    "production",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Shutdown(ctx)
    
    // Use in Connect interceptors
    interceptors := connectx.DefaultInterceptors(connectx.Options{
        Logger: logger,
        Otel:   provider,
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

### With Custom Configuration

```go
func main() {
    // Initialize OpenTelemetry with custom configuration
    provider, err := obsx.NewProvider(ctx, logger, obsx.Options{
        ServiceName:    "user-service",
        ServiceVersion: "1.0.0",
        Environment:    "production",
        Endpoint:       "otel-collector:4317",
        Insecure:       true,
        Headers: map[string]string{
            "Authorization": "Bearer token",
        },
        Timeout:      10 * time.Second,
        BatchTimeout: 5 * time.Second,
        BatchSize:    512,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Shutdown(ctx)
    
    // Use provider
    useProvider(provider)
}
```

### Manual Tracing

```go
func (s *UserService) GetUser(ctx context.Context, req *connect.Request[GetUserRequest]) (*connect.Response[GetUserResponse], error) {
    // Get tracer
    tracer := s.provider.GetTracer("user-service")
    
    // Start span
    ctx, span := tracer.Start(ctx, "GetUser")
    defer span.End()
    
    // Add attributes
    span.SetAttributes(
        attribute.String("user.id", req.Msg.UserId),
        attribute.String("service.name", "user-service"),
    )
    
    // Business logic
    user, err := s.repo.GetUser(ctx, req.Msg.UserId)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }
    
    // Add result attributes
    span.SetAttributes(
        attribute.String("user.name", user.Name),
        attribute.String("user.email", user.Email),
    )
    
    return connect.NewResponse(&GetUserResponse{User: user}), nil
}
```

### Manual Metrics

```go
func (s *UserService) GetUser(ctx context.Context, req *connect.Request[GetUserRequest]) (*connect.Response[GetUserResponse], error) {
    // Get meter
    meter := s.provider.GetMeter("user-service")
    
    // Create counters
    requestCounter, _ := meter.Int64Counter("requests_total")
    durationHistogram, _ := meter.Int64Histogram("request_duration_ms")
    
    // Record metrics
    start := time.Now()
    requestCounter.Add(ctx, 1, attribute.String("method", "GetUser"))
    
    defer func() {
        duration := time.Since(start).Milliseconds()
        durationHistogram.Record(ctx, duration, attribute.String("method", "GetUser"))
    }()
    
    // Business logic
    user, err := s.repo.GetUser(ctx, req.Msg.UserId)
    if err != nil {
        requestCounter.Add(ctx, 1, attribute.String("method", "GetUser"), attribute.String("status", "error"))
        return nil, err
    }
    
    requestCounter.Add(ctx, 1, attribute.String("method", "GetUser"), attribute.String("status", "success"))
    return connect.NewResponse(&GetUserResponse{User: user}), nil
}
```

### Database Tracing

```go
func (r *UserRepository) GetUser(ctx context.Context, userID string) (*User, error) {
    // Get tracer
    tracer := r.provider.GetTracer("user-repository")
    
    // Start span
    ctx, span := tracer.Start(ctx, "GetUser")
    defer span.End()
    
    // Add database attributes
    span.SetAttributes(
        attribute.String("db.system", "mysql"),
        attribute.String("db.operation", "SELECT"),
        attribute.String("db.table", "users"),
        attribute.String("user.id", userID),
    )
    
    // Execute query
    var user User
    err := r.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }
    
    // Add result attributes
    span.SetAttributes(
        attribute.String("user.name", user.Name),
        attribute.String("user.email", user.Email),
    )
    
    return &user, nil
}
```

### External Service Tracing

```go
func (c *EmailClient) SendEmail(ctx context.Context, email *Email) error {
    // Get tracer
    tracer := c.provider.GetTracer("email-client")
    
    // Start span
    ctx, span := tracer.Start(ctx, "SendEmail")
    defer span.End()
    
    // Add external service attributes
    span.SetAttributes(
        attribute.String("http.method", "POST"),
        attribute.String("http.url", c.endpoint),
        attribute.String("email.to", email.To),
        attribute.String("email.subject", email.Subject),
    )
    
    // Make HTTP request
    req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(email.Body))
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }
    
    resp, err := c.client.Do(req)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }
    defer resp.Body.Close()
    
    // Add response attributes
    span.SetAttributes(
        attribute.Int("http.status_code", resp.StatusCode),
    )
    
    if resp.StatusCode >= 400 {
        err := errors.New("EXTERNAL_SERVICE_ERROR", "email service returned error")
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }
    
    return nil
}
```

## Configuration

### Environment Variables

```bash
# OpenTelemetry configuration
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
OTEL_EXPORTER_OTLP_INSECURE=true
OTEL_SERVICE_NAME=user-service
OTEL_SERVICE_VERSION=1.0.0
OTEL_RESOURCE_ATTRIBUTES=environment=production

# Custom configuration
OBSX_TIMEOUT=10s
OBSX_BATCH_TIMEOUT=5s
OBSX_BATCH_SIZE=512
```

### Configuration Integration

```go
type AppConfig struct {
    configx.BaseConfig
    
    // Observability configuration
    Observability ObservabilityConfig
}

type ObservabilityConfig struct {
    Enabled       bool              `env:"OBSX_ENABLED" default:"true"`
    Endpoint      string            `env:"OTEL_EXPORTER_OTLP_ENDPOINT" default:"otel-collector:4317"`
    Insecure      bool              `env:"OTEL_EXPORTER_OTLP_INSECURE" default:"true"`
    Timeout       time.Duration     `env:"OBSX_TIMEOUT" default:"10s"`
    BatchTimeout  time.Duration     `env:"OBSX_BATCH_TIMEOUT" default:"5s"`
    BatchSize     int               `env:"OBSX_BATCH_SIZE" default:"512"`
}

func main() {
    // Load configuration
    var cfg AppConfig
    if err := configManager.Bind(&cfg); err != nil {
        log.Fatal(err)
    }
    
    // Initialize OpenTelemetry
    var provider *obsx.Provider
    if cfg.Observability.Enabled {
        var err error
        provider, err = obsx.NewProvider(ctx, logger, obsx.Options{
            ServiceName:    cfg.ServiceName,
            ServiceVersion: cfg.ServiceVersion,
            Environment:    cfg.Env,
            Endpoint:       cfg.Observability.Endpoint,
            Insecure:       cfg.Observability.Insecure,
            Timeout:        cfg.Observability.Timeout,
            BatchTimeout:   cfg.Observability.BatchTimeout,
            BatchSize:      cfg.Observability.BatchSize,
        })
        if err != nil {
            log.Fatal(err)
        }
        defer provider.Shutdown(ctx)
    }
    
    // Use provider
    useProvider(provider)
}
```

## Testing

```go
func TestObservability(t *testing.T) {
    // Create no-op provider for testing
    provider := obsx.NewNoOpProvider()
    
    // Test tracer
    tracer := provider.GetTracer("test-service")
    ctx, span := tracer.Start(context.Background(), "TestSpan")
    span.End()
    
    // Test meter
    meter := provider.GetMeter("test-service")
    counter, _ := meter.Int64Counter("test_counter")
    counter.Add(ctx, 1)
    
    // Test shutdown
    err := provider.Shutdown(ctx)
    assert.NoError(t, err)
}

func TestServiceWithObservability(t *testing.T) {
    // Create test provider
    provider := obsx.NewNoOpProvider()
    
    // Create test service
    service := &TestUserService{provider: provider}
    
    // Test service method
    resp, err := service.GetUser(context.Background(), &connect.Request[GetUserRequest]{
        Msg: &GetUserRequest{UserId: "test-user"},
    })
    
    assert.NoError(t, err)
    assert.NotNil(t, resp)
}

type TestUserService struct {
    provider *obsx.Provider
}

func (s *TestUserService) GetUser(ctx context.Context, req *connect.Request[GetUserRequest]) (*connect.Response[GetUserResponse], error) {
    // Get tracer
    tracer := s.provider.GetTracer("test-service")
    
    // Start span
    ctx, span := tracer.Start(ctx, "GetUser")
    defer span.End()
    
    // Add attributes
    span.SetAttributes(attribute.String("user.id", req.Msg.UserId))
    
    // Return test response
    return connect.NewResponse(&GetUserResponse{
        User: &User{Id: req.Msg.UserId, Name: "Test User"},
    }), nil
}
```

## Best Practices

### 1. Use Structured Attributes

```go
func (s *UserService) GetUser(ctx context.Context, req *connect.Request[GetUserRequest]) (*connect.Response[GetUserResponse], error) {
    tracer := s.provider.GetTracer("user-service")
    ctx, span := tracer.Start(ctx, "GetUser")
    defer span.End()
    
    // Use structured attributes
    span.SetAttributes(
        attribute.String("user.id", req.Msg.UserId),
        attribute.String("service.name", "user-service"),
        attribute.String("service.version", "1.0.0"),
    )
    
    // Business logic
    user, err := s.repo.GetUser(ctx, req.Msg.UserId)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }
    
    // Add result attributes
    span.SetAttributes(
        attribute.String("user.name", user.Name),
        attribute.String("user.email", user.Email),
    )
    
    return connect.NewResponse(&GetUserResponse{User: user}), nil
}
```

### 2. Record Errors Properly

```go
func (s *UserService) CreateUser(ctx context.Context, req *connect.Request[CreateUserRequest]) (*connect.Response[CreateUserResponse], error) {
    tracer := s.provider.GetTracer("user-service")
    ctx, span := tracer.Start(ctx, "CreateUser")
    defer span.End()
    
    // Business logic
    user, err := s.repo.CreateUser(ctx, req.Msg.User)
    if err != nil {
        // Record error with context
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        span.SetAttributes(
            attribute.String("error.type", "database_error"),
            attribute.String("error.message", err.Error()),
        )
        return nil, err
    }
    
    return connect.NewResponse(&CreateUserResponse{User: user}), nil
}
```

### 3. Use Metrics for Business Logic

```go
func (s *UserService) GetUser(ctx context.Context, req *connect.Request[GetUserRequest]) (*connect.Response[GetUserResponse], error) {
    meter := s.provider.GetMeter("user-service")
    
    // Create counters
    requestCounter, _ := meter.Int64Counter("requests_total")
    durationHistogram, _ := meter.Int64Histogram("request_duration_ms")
    
    // Record metrics
    start := time.Now()
    requestCounter.Add(ctx, 1, attribute.String("method", "GetUser"))
    
    defer func() {
        duration := time.Since(start).Milliseconds()
        durationHistogram.Record(ctx, duration, attribute.String("method", "GetUser"))
    }()
    
    // Business logic
    user, err := s.repo.GetUser(ctx, req.Msg.UserId)
    if err != nil {
        requestCounter.Add(ctx, 1, attribute.String("method", "GetUser"), attribute.String("status", "error"))
        return nil, err
    }
    
    requestCounter.Add(ctx, 1, attribute.String("method", "GetUser"), attribute.String("status", "success"))
    return connect.NewResponse(&GetUserResponse{User: user}), nil
}
```

### 4. Handle Provider Lifecycle

```go
func main() {
    // Initialize OpenTelemetry
    provider, err := obsx.NewProvider(ctx, logger, obsx.Options{
        ServiceName:    "user-service",
        ServiceVersion: "1.0.0",
        Environment:    "production",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Ensure proper shutdown
    defer func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        
        if err := provider.Shutdown(ctx); err != nil {
            logger.Error(err, "Failed to shutdown OpenTelemetry provider")
        }
    }()
    
    // Use provider
    useProvider(provider)
}
```

## Thread Safety

All functions in this package are safe for concurrent use. The OpenTelemetry provider is designed to handle concurrent access safely.

## Dependencies

- **Go 1.21+** required
- **OpenTelemetry Go** - Core observability
- **OpenTelemetry OTLP** - Export functionality
- **Standard library** - Core functionality

## Version Compatibility

- **Go 1.21+** required
- **API Stability**: Evolving (L3 module)
- **Breaking Changes**: Possible in minor versions

## Contributing

Contributions are welcome! Please see the main project [Contributing Guide](../CONTRIBUTING.md) for details.

## License

This package is part of the EggyByte framework and is licensed under the MIT License.