# egg/obsx

## Overview

`obsx` provides OpenTelemetry provider initialization and lifecycle management for
egg microservices. It offers a simple, production-ready way to instrument services
with distributed tracing and metrics collection using OpenTelemetry.

## Module Structure

Following egg's module organization standards:

```
obsx/
├── obsx.go              # Public API (types, constructors, exported methods)
├── doc.go               # Package documentation
├── obsx_test.go         # Public API tests
└── internal/            # Internal implementation
    ├── provider.go      # Provider lifecycle management
    ├── runtime_metrics.go   # Go runtime metrics
    ├── process_metrics.go   # Process-level metrics
    └── db_metrics.go    # Database pool metrics
```

**Key principle**: All implementation logic resides in `internal/`, while `obsx.go` only exports the public API.

## Key Features

- Simplified OpenTelemetry provider initialization
- **Dual metrics export**: Local Prometheus endpoint + OTLP export
- **Production-grade metrics**: Runtime, process, and database pool metrics
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
go get go.eggybyte.com/egg/obsx@latest
```

## Basic Usage

### With servicex (Recommended)

When using `servicex`, OpenTelemetry is automatically configured:

```go
import (
    "go.eggybyte.com/egg/configx"
    "go.eggybyte.com/egg/servicex"
)

type AppConfig struct {
    configx.BaseConfig  // Includes OTEL_EXPORTER_OTLP_ENDPOINT
}

func register(app *servicex.App) error {
    // Get OpenTelemetry provider (nil if not configured)
    provider := app.OtelProvider()
    if provider != nil {
        tracer := provider.TracerProvider().Tracer("my-service")
        // Use tracer...
    }
    return nil
}

func main() {
    ctx := context.Background()
    cfg := &AppConfig{}
    
    servicex.Run(ctx,
        servicex.WithAppConfig(cfg),
        servicex.WithRegister(register),
    )
}
```

Set `OTEL_EXPORTER_OTLP_ENDPOINT` environment variable to enable tracing.

### Standalone Usage

```go
import (
    "context"
    "go.eggybyte.com/egg/obsx"
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
    
    // Expose Prometheus metrics endpoint
    http.Handle("/metrics", provider.PrometheusHandler())
    go http.ListenAndServe(":9091", nil)
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

## Metrics Export

obsx provides **dual metrics export** - metrics are simultaneously available via:

1. **Local Prometheus Endpoint**: Pull-based metrics via HTTP `/metrics` endpoint
2. **OTLP Export**: Push-based metrics to OpenTelemetry Collector (if configured)

### Prometheus Endpoint

The Prometheus endpoint is always available through `PrometheusHandler()`:

```go
provider, _ := obsx.NewProvider(ctx, obsx.Options{
    ServiceName: "my-service",
    ServiceVersion: "1.0.0",
})

// Get Prometheus HTTP handler
metricsHandler := provider.PrometheusHandler()

// Mount on HTTP server
mux := http.NewServeMux()
mux.Handle("/metrics", metricsHandler)

// Access metrics
// curl http://localhost:9091/metrics
```

**When using servicex**, the metrics endpoint is automatically started on port 9091 (configurable via `METRICS_PORT`).

### Custom Metrics

The `Meter()` method provides access to OpenTelemetry's Meter API for creating custom business metrics:

```go
provider, _ := obsx.NewProvider(ctx, obsx.Options{
    ServiceName: "user-service",
})

// Get a meter for your service
meter := provider.Meter("user-service")

// Create a counter for tracking registrations
registrationCounter, _ := meter.Int64Counter(
    "user.registrations.total",
    metric.WithDescription("Total user registrations"),
    metric.WithUnit("{registration}"),
)

// Increment counter
registrationCounter.Add(ctx, 1, 
    metric.WithAttributes(
        attribute.String("source", "web"),
        attribute.String("country", "US"),
    ),
)

// Create a histogram for tracking request durations
durationHistogram, _ := meter.Float64Histogram(
    "payment.process.duration",
    metric.WithDescription("Payment processing duration in seconds"),
    metric.WithUnit("s"),
)

// Record duration
durationHistogram.Record(ctx, 0.125, 
    metric.WithAttributes(
        attribute.String("payment_method", "credit_card"),
        attribute.String("status", "success"),
    ),
)

// Create an async gauge for tracking queue depth
queueDepthGauge, _ := meter.Int64ObservableGauge(
    "queue.depth",
    metric.WithDescription("Current queue depth"),
    metric.WithUnit("{item}"),
)
meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
    depth := getQueueDepth() // Your function to get queue depth
    o.ObserveInt64(queueDepthGauge, depth)
    return nil
}, queueDepthGauge)
```

**All custom metrics are automatically:**

- Exposed via the `/metrics` Prometheus endpoint
- Exported to OTLP collector (if configured)
- Available for querying and alerting

**Best practices for custom metrics:**

- Use descriptive metric names following OpenTelemetry conventions (e.g., `service.operation.metric_type`)
- Add relevant attributes (labels) for filtering and grouping
- Choose appropriate metric types:
  - **Counter**: Monotonically increasing values (requests, errors, bytes)
  - **Histogram**: Distribution of values (durations, sizes)
  - **Gauge**: Current value that can go up or down (queue depth, active connections)

### OTLP Export

If `OTLPEndpoint` is configured, metrics are also sent to the OpenTelemetry Collector:

```go
provider, _ := obsx.NewProvider(ctx, obsx.Options{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    OTLPEndpoint:   "otel-collector:4317",  // Enables OTLP export
})
// Metrics are now sent to both Prometheus endpoint AND OTLP collector
```

**Benefits of dual export:**
- **Development**: Direct Prometheus endpoint for local testing and debugging
- **Production**: OTLP export for centralized collection and aggregation
- **Flexibility**: Choose based on infrastructure (Prometheus scrape vs. OTLP push)

### Metrics Format

Metrics are exported in Prometheus text exposition format:

```
# HELP target_info Target metadata
# TYPE target_info gauge
target_info{service_name="my-service",service_version="1.0.0"} 1
```

Custom metrics can be added using the OpenTelemetry Meter API:

```go
meter := provider.MeterProvider().Meter("my-service")
counter, _ := meter.Int64Counter("requests_total")
counter.Add(ctx, 1, metric.WithAttributes(
    attribute.String("method", "GET"),
))
```

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

// PrometheusHandler returns an HTTP handler for Prometheus metrics endpoint
func (p *Provider) PrometheusHandler() http.Handler

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
    "go.eggybyte.com/egg/servicex"
    "go.eggybyte.com/egg/obsx"
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
