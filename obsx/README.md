# ObsX Module

<div align="center">

**OpenTelemetry observability provider for Egg services**

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg)](LICENSE)

</div>

## 📦 Overview

The `obsx` module provides OpenTelemetry observability provider for Egg services. It offers tracing, metrics, and logging integration with OTLP exporters and runtime metrics collection.

## ✨ Features

- 📊 **OpenTelemetry Integration** - Full OpenTelemetry support
- 🔍 **Distributed Tracing** - Request tracing across services
- 📈 **Metrics Collection** - Prometheus metrics
- 📝 **Structured Logging** - Context-aware logging
- 🌐 **OTLP Exporters** - OpenTelemetry Protocol exporters
- ⚡ **Runtime Metrics** - Automatic runtime metrics collection
- 🔧 **Easy Configuration** - Simple setup and configuration
- 🎯 **Service Integration** - Seamless integration with Egg services

## 🏗️ Architecture

```
obsx/
├── obsx.go        # Main observability provider
└── obsx_test.go   # Tests
```

## 🚀 Quick Start

### Installation

```bash
go get github.com/eggybyte-technology/egg/obsx@latest
```

### Basic Usage

```go
package main

import (
    "context"
    "time"

    "github.com/eggybyte-technology/egg/obsx"
    "github.com/eggybyte-technology/egg/core/log"
)

func main() {
    // Create logger
    logger := &YourLogger{} // Implement log.Logger interface

    // Create OpenTelemetry provider
    ctx := context.Background()
    otel, err := obsx.NewProvider(ctx, obsx.Options{
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
        Environment:   "production",
        OTLPEndpoint:  "http://otel-collector:4317",
    })
    if err != nil {
        log.Fatal("Failed to create OpenTelemetry provider:", err)
    }
    defer otel.Shutdown(ctx)

    // Your application logic
    runApplication(ctx, otel)
}
```

## 📖 API Reference

### Provider Options

```go
type Options struct {
    ServiceName    string
    ServiceVersion string
    Environment    string
    OTLPEndpoint   string
    OTLPHeaders    map[string]string
    SamplingRate   float64
    MetricsPort    string
    LogLevel       string
}

type Provider interface {
    Shutdown(ctx context.Context) error
    Tracer(name string) trace.Tracer
    Meter(name string) metric.Meter
    Logger() log.Logger
}
```

### Main Functions

```go
// NewProvider creates a new OpenTelemetry provider
func NewProvider(ctx context.Context, opts Options) (Provider, error)

// DefaultProvider creates a provider with default options
func DefaultProvider(ctx context.Context, serviceName string) (Provider, error)
```

## 🔧 Configuration

### Environment Variables

```bash
# Service identification
export SERVICE_NAME="my-service"
export SERVICE_VERSION="1.0.0"
export ENV="production"

# OpenTelemetry
export OTEL_EXPORTER_OTLP_ENDPOINT="http://otel-collector:4317"
export OTEL_EXPORTER_OTLP_HEADERS="Authorization=Bearer token"
export OTEL_SAMPLING_RATE="1.0"

# Metrics
export METRICS_PORT=":9091"

# Logging
export LOG_LEVEL="info"
```

### Configuration File

```yaml
# config.yaml
service_name: "my-service"
service_version: "1.0.0"
environment: "production"

otel_exporter_otlp_endpoint: "http://otel-collector:4317"
otel_exporter_otlp_headers:
  Authorization: "Bearer token"
otel_sampling_rate: 1.0

metrics_port: ":9091"
log_level: "info"
```

## 🛠️ Advanced Usage

### Custom Tracer

```go
func main() {
    // Create provider
    otel, err := obsx.NewProvider(ctx, obsx.Options{
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
        OTLPEndpoint:  "http://otel-collector:4317",
    })
    if err != nil {
        log.Fatal("Failed to create provider:", err)
    }
    defer otel.Shutdown(ctx)

    // Get tracer
    tracer := otel.Tracer("my-service")

    // Create span
    ctx, span := tracer.Start(ctx, "operation")
    defer span.End()

    // Add attributes
    span.SetAttributes(
        attribute.String("user.id", "user123"),
        attribute.Int("request.id", 456),
    )

    // Your business logic
    result, err := performOperation(ctx)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }

    span.SetAttributes(attribute.String("result", result))
    return nil
}
```

### Custom Metrics

```go
func main() {
    // Create provider
    otel, err := obsx.NewProvider(ctx, obsx.Options{
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
        OTLPEndpoint:  "http://otel-collector:4317",
    })
    if err != nil {
        log.Fatal("Failed to create provider:", err)
    }
    defer otel.Shutdown(ctx)

    // Get meter
    meter := otel.Meter("my-service")

    // Create counter
    counter, err := meter.Int64Counter(
        "requests_total",
        metric.WithDescription("Total number of requests"),
    )
    if err != nil {
        log.Fatal("Failed to create counter:", err)
    }

    // Create histogram
    histogram, err := meter.Float64Histogram(
        "request_duration_seconds",
        metric.WithDescription("Request duration in seconds"),
    )
    if err != nil {
        log.Fatal("Failed to create histogram:", err)
    }

    // Record metrics
    counter.Add(ctx, 1, metric.WithAttributes(
        attribute.String("method", "GET"),
        attribute.String("status", "200"),
    ))

    histogram.Record(ctx, 0.1, metric.WithAttributes(
        attribute.String("method", "GET"),
        attribute.String("status", "200"),
    ))
}
```

### Custom Logger

```go
func main() {
    // Create provider
    otel, err := obsx.NewProvider(ctx, obsx.Options{
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
        OTLPEndpoint:  "http://otel-collector:4317",
    })
    if err != nil {
        log.Fatal("Failed to create provider:", err)
    }
    defer otel.Shutdown(ctx)

    // Get logger
    logger := otel.Logger()

    // Log with context
    logger.Log(ctx, log.LevelInfo, "Processing request",
        log.Attr{Key: "user.id", Value: "user123"},
        log.Attr{Key: "request.id", Value: "req456"},
    )
}
```

## 🔧 Integration with Other Modules

### ConnectX Integration

```go
func main() {
    // Create observability provider
    otel, err := obsx.NewProvider(ctx, obsx.Options{
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
        OTLPEndpoint:  "http://otel-collector:4317",
    })
    if err != nil {
        log.Fatal("Failed to create provider:", err)
    }
    defer otel.Shutdown(ctx)

    // Create Connect interceptors with observability
    interceptors := connectx.DefaultInterceptors(connectx.Options{
        Logger: otel.Logger(),
        Otel:   otel,
    })

    // Use interceptors in Connect service
    handler := NewMyServiceHandler(service, connect.WithInterceptors(interceptors...))
}
```

### RuntimeX Integration

```go
func main() {
    // Create observability provider
    otel, err := obsx.NewProvider(ctx, obsx.Options{
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
        OTLPEndpoint:  "http://otel-collector:4317",
    })
    if err != nil {
        log.Fatal("Failed to create provider:", err)
    }
    defer otel.Shutdown(ctx)

    // Configure runtime with observability
    opts := runtimex.Options{
        Logger: otel.Logger(),
        HTTP: &runtimex.HTTPOptions{
            Addr: ":8080",
            H2C:  true,
            Mux:  mux,
        },
        Health:  &runtimex.Endpoint{Addr: ":8081"},
        Metrics: &runtimex.Endpoint{Addr: ":9091"},
    }

    // Run service
    runtimex.Run(ctx, nil, opts)
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
| ObsX | Good |

## 🔍 Troubleshooting

### Common Issues

1. **OTLP Endpoint Not Reachable**
   ```bash
   # Check if OTLP endpoint is accessible
   curl -v http://otel-collector:4317/v1/traces
   ```

2. **Metrics Not Collected**
   ```go
   // Ensure metrics port is configured
   otel, err := obsx.NewProvider(ctx, obsx.Options{
       ServiceName:    "my-service",
       ServiceVersion: "1.0.0",
       MetricsPort:   ":9091",
   })
   ```

3. **Tracing Not Working**
   ```go
   // Check sampling rate
   otel, err := obsx.NewProvider(ctx, obsx.Options{
       ServiceName:    "my-service",
       ServiceVersion: "1.0.0",
       SamplingRate:  1.0, // 100% sampling
   })
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
