# egg/obsx

## Overview

`obsx` provides OpenTelemetry provider initialization and lifecycle management for
egg microservices. It offers a simple, production-ready way to instrument services
with distributed tracing and metrics collection using OpenTelemetry.

## Key Features

- Simplified OpenTelemetry provider initialization
- Support for OTLP trace and metric export
- Configurable sampling strategies
- Custom resource attributes
- Graceful provider shutdown
- Clean separation of interface and implementation

## Dependencies

Layer: **L2 (Capability Layer)**  
Depends on: None (zero dependencies outside OpenTelemetry SDK)

## Installation

```bash
go get github.com/eggybyte-technology/egg/obsx@latest
```

## Basic Usage

```go
import (
    "context"
    "github.com/eggybyte-technology/egg/obsx"
)

func main() {
    ctx := context.Background()
    
    // Initialize OpenTelemetry provider
    provider, err := obsx.NewProvider(ctx, obsx.Options{
        ServiceName:    "user-service",
        ServiceVersion: "1.0.0",
        OTLPEndpoint:   "otel-collector:4317",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Shutdown(ctx)
    
    // Access tracer and meter providers
    tracerProvider := provider.TracerProvider()
    meterProvider := provider.MeterProvider()
    
    // Use providers with OpenTelemetry SDK
    tracer := tracerProvider.Tracer("user-service")
    meter := meterProvider.Meter("user-service")
}
```

## Configuration Options

| Option                | Type              | Description                                    |
| --------------------- | ----------------- | ---------------------------------------------- |
| `ServiceName`         | `string`          | Service name for tracing (required)            |
| `ServiceVersion`      | `string`          | Service version                                |
| `OTLPEndpoint`        | `string`          | OTLP collector endpoint (e.g., "host:4317")    |
| `EnableRuntimeMetrics`| `bool`            | Enable Go runtime metrics (future)             |
| `ResourceAttrs`       | `map[string]string`| Additional resource attributes                |
| `TraceSamplerRatio`   | `float64`         | Trace sampling ratio (0.0-1.0, default: 0.1)  |

## API Reference

### Types

#### Provider

```go
type Provider struct {
    // Public methods only
}

// TracerProvider returns the OpenTelemetry tracer provider
func (p *Provider) TracerProvider() *sdktrace.TracerProvider

// MeterProvider returns the OpenTelemetry meter provider
func (p *Provider) MeterProvider() *metric.MeterProvider

// Shutdown gracefully shuts down the provider
func (p *Provider) Shutdown(ctx context.Context) error
```

#### Options

```go
type Options struct {
    ServiceName          string            // Service name (required)
    ServiceVersion       string            // Service version
    OTLPEndpoint         string            // OTLP endpoint
    EnableRuntimeMetrics bool              // Enable runtime metrics
    ResourceAttrs        map[string]string // Custom attributes
    TraceSamplerRatio    float64           // Sampling ratio (0.0-1.0)
}
```

### Functions

```go
// NewProvider creates a new observability provider
func NewProvider(ctx context.Context, opts Options) (*Provider, error)
```

## Architecture

The obsx module follows a clean architecture pattern:

```
obsx/
├── obsx.go              # Public API (~100 lines)
│   ├── Options          # Configuration struct
│   ├── Provider         # Wrapper type
│   └── NewProvider()    # Constructor (delegates to internal)
└── internal/
    └── provider.go      # Implementation (~200 lines)
        ├── createResource()       # Resource creation
        ├── createTracerProvider() # Tracer provider setup
        └── createMeterProvider()  # Meter provider setup
```

**Design Highlights:**
- Public interface is minimal and focused
- Complex initialization logic isolated in internal package
- Provider lifecycle managed through simple Shutdown() method
- All OpenTelemetry globals set automatically

## Integration with servicex

The obsx provider integrates seamlessly with servicex:

```go
import (
    "github.com/eggybyte-technology/egg/servicex"
    "github.com/eggybyte-technology/egg/obsx"
)

func main() {
    ctx := context.Background()
    cfg := &AppConfig{}
    
    err := servicex.Run(ctx,
        servicex.WithConfig(cfg),
        servicex.WithTracing(true),  // Enables obsx provider
        servicex.WithRegister(register),
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

## Example: Manual Tracing

```go
func (s *UserService) GetUser(ctx context.Context, req *connect.Request[userv1.GetUserRequest]) (*connect.Response[userv1.GetUserResponse], error) {
    // Get tracer from provider
    tracer := s.provider.TracerProvider().Tracer("user-service")
    
    // Start span
    ctx, span := tracer.Start(ctx, "GetUser")
    defer span.End()
    
    // Add attributes
    span.SetAttributes(
        attribute.String("user.id", req.Msg.UserId),
    )
    
    // Business logic
    user, err := s.repo.GetUser(ctx, req.Msg.UserId)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }
    
    return connect.NewResponse(&userv1.GetUserResponse{User: user}), nil
}
```

## Example: Manual Metrics

```go
func (s *UserService) init() {
    meter := s.provider.MeterProvider().Meter("user-service")
    
    // Create counter
    s.requestCounter, _ = meter.Int64Counter(
        "user.requests.total",
        metric.WithDescription("Total user service requests"),
    )
    
    // Create histogram
    s.durationHistogram, _ = meter.Int64Histogram(
        "user.request.duration",
        metric.WithDescription("Request duration in milliseconds"),
        metric.WithUnit("ms"),
    )
}

func (s *UserService) GetUser(ctx context.Context, req *connect.Request[userv1.GetUserRequest]) (*connect.Response[userv1.GetUserResponse], error) {
    start := time.Now()
    
    // Increment counter
    s.requestCounter.Add(ctx, 1, metric.WithAttributes(
        attribute.String("method", "GetUser"),
    ))
    
    // Business logic
    user, err := s.repo.GetUser(ctx, req.Msg.UserId)
    
    // Record duration
    duration := time.Since(start).Milliseconds()
    s.durationHistogram.Record(ctx, duration, metric.WithAttributes(
        attribute.String("method", "GetUser"),
        attribute.String("status", statusFromError(err)),
    ))
    
    if err != nil {
        return nil, err
    }
    
    return connect.NewResponse(&userv1.GetUserResponse{User: user}), nil
}
```

## Observability

When properly configured with an OTLP collector, obsx enables:

- **Distributed Tracing**: Spans exported to Jaeger, Tempo, or similar
- **Metrics Collection**: Metrics exported to Prometheus, OTLP-compatible backends
- **Resource Attributes**: Automatic service.name, service.version tagging
- **Sampling**: Configurable trace sampling to control volume

## Testing

For testing, use a no-op provider or omit the OTLP endpoint:

```go
func TestUserService(t *testing.T) {
    ctx := context.Background()
    
    // Create provider without OTLP endpoint (no-op exporters)
    provider, err := obsx.NewProvider(ctx, obsx.Options{
        ServiceName:    "test-service",
        ServiceVersion: "test",
    })
    require.NoError(t, err)
    defer provider.Shutdown(ctx)
    
    // Test service with provider
    service := NewUserService(provider)
    // ... run tests
}
```

## Stability

**Status**: Stable  
**Layer**: L2 (Capability)  
**API Guarantees**: Backward-compatible changes only

The obsx module is production-ready and follows semantic versioning.
Breaking changes will only occur in major version updates.

## License

This package is part of the egg framework and is licensed under the MIT License.
See the root LICENSE file for details.
