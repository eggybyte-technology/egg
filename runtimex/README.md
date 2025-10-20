# RuntimeX Module

<div align="center">

**Runtime management and lifecycle orchestration for Egg services**

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg)](LICENSE)

</div>

## 📦 Overview

The `runtimex` module provides runtime management and lifecycle orchestration for Egg services. It handles service startup, shutdown, health checks, and metrics endpoints with a unified port strategy.

## ✨ Features

- 🚀 **Lifecycle Management** - Graceful startup and shutdown
- 🎯 **Unified Port Strategy** - HTTP/Connect/gRPC-Web on single port
- ❤️ **Health Checks** - Built-in health endpoint
- 📊 **Metrics Endpoint** - Prometheus metrics collection
- ⏱️ **Graceful Shutdown** - Configurable shutdown timeout
- 🔧 **H2C Support** - HTTP/2 Cleartext support
- 📝 **Structured Logging** - Context-aware logging

## 🏗️ Architecture

```
runtimex/
├── runtime.go      # Main runtime interface
├── internal/
│   └── runtime.go  # Internal implementation
└── runtimex_test.go # Tests
```

## 🚀 Quick Start

### Installation

```bash
go get github.com/eggybyte-technology/egg/runtimex@latest
```

### Basic Usage

```go
package main

import (
    "context"
    "net/http"
    "time"

    "github.com/eggybyte-technology/egg/runtimex"
    "github.com/eggybyte-technology/egg/core/log"
)

func main() {
    // Create logger
    logger := &YourLogger{} // Implement log.Logger interface

    // Create HTTP mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Hello, World!"))
    })

    // Configure runtime options
    opts := runtimex.Options{
        Logger: logger,
        HTTP: &runtimex.HTTPOptions{
            Addr: ":8080",
            H2C:  true,
            Mux:  mux,
        },
        Health: &runtimex.Endpoint{
            Addr: ":8081",
        },
        Metrics: &runtimex.Endpoint{
            Addr: ":9091",
        },
        ShutdownTimeout: 15 * time.Second,
    }

    // Run the service
    ctx := context.Background()
    if err := runtimex.Run(ctx, nil, opts); err != nil {
        logger.Log(ctx, log.LevelError, "service failed", log.Attr{Key: "error", Value: err})
    }
}
```

## 📖 API Reference

### Runtime Options

```go
type Options struct {
    Logger          log.Logger
    HTTP            *HTTPOptions
    Health          *Endpoint
    Metrics         *Endpoint
    ShutdownTimeout time.Duration
}

type HTTPOptions struct {
    Addr string
    H2C  bool
    Mux  *http.ServeMux
}

type Endpoint struct {
    Addr string
}
```

### Main Functions

```go
// Run starts the runtime with the given options
func Run(ctx context.Context, ready chan<- struct{}, opts Options) error

// New creates a new runtime instance
func New(opts Options) *Runtime

// Start starts the runtime
func (r *Runtime) Start(ctx context.Context) error

// Stop stops the runtime gracefully
func (r *Runtime) Stop(ctx context.Context) error
```

## 🔧 Configuration

### Environment Variables

```bash
# HTTP server port
export HTTP_PORT=":8080"

# Health check port
export HEALTH_PORT=":8081"

# Metrics port
export METRICS_PORT=":9091"

# Shutdown timeout
export SHUTDOWN_TIMEOUT="15s"
```

### HTTP/2 Cleartext (H2C)

Enable H2C support for better performance:

```go
opts := runtimex.Options{
    HTTP: &runtimex.HTTPOptions{
        Addr: ":8080",
        H2C:  true, // Enable H2C
        Mux:  mux,
    },
}
```

## 📊 Health Checks

The runtime automatically provides health check endpoints:

- **Health**: `GET /health` - Returns service health status
- **Readiness**: `GET /ready` - Returns service readiness status
- **Liveness**: `GET /live` - Returns service liveness status

### Custom Health Checks

```go
// Add custom health check logic
mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    // Your health check logic
    if isHealthy() {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("healthy"))
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("unhealthy"))
    }
})
```

## 📈 Metrics

The runtime provides Prometheus metrics endpoints:

- **Metrics**: `GET /metrics` - Returns Prometheus metrics

### Custom Metrics

```go
import "github.com/prometheus/client_golang/prometheus"

// Register custom metrics
counter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "requests_total",
        Help: "Total number of requests",
    },
    []string{"method", "status"},
)
prometheus.MustRegister(counter)
```

## 🛠️ Advanced Usage

### Graceful Shutdown

```go
func main() {
    // Create runtime
    runtime := runtimex.New(opts)

    // Start runtime
    ctx := context.Background()
    if err := runtime.Start(ctx); err != nil {
        log.Fatal("Failed to start runtime:", err)
    }

    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    // Graceful shutdown
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    if err := runtime.Stop(shutdownCtx); err != nil {
        log.Fatal("Failed to stop runtime:", err)
    }
}
```

### Custom Middleware

```go
// Add custom middleware
mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    // Add CORS headers
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

    // Your handler
    handler(w, r)
})
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
| Runtime | 58.1% |

## 🔍 Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```bash
   # Check if port is in use
   lsof -i :8080
   
   # Kill process using port
   kill -9 $(lsof -t -i:8080)
   ```

2. **Graceful Shutdown Timeout**
   ```go
   // Increase shutdown timeout
   opts.ShutdownTimeout = 30 * time.Second
   ```

3. **H2C Not Working**
   ```go
   // Ensure H2C is enabled
   opts.HTTP.H2C = true
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
