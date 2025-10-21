# ⚡ RuntimeX Package

The `runtimex` package provides service lifecycle management and runtime infrastructure for the EggyByte framework.

## Overview

This package offers a comprehensive runtime system that handles service startup, shutdown, health checks, metrics collection, and graceful shutdown. It's designed to be production-ready with observability built-in.

## Features

- **Service lifecycle management** - Startup, shutdown, and health monitoring
- **HTTP server integration** - Built-in HTTP server with H2C support
- **Health checks** - Configurable health check endpoints
- **Metrics collection** - Prometheus metrics integration
- **Graceful shutdown** - Proper signal handling and cleanup
- **Observability** - Logging and tracing integration

## Quick Start

```go
import "github.com/eggybyte-technology/egg/runtimex"

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Create HTTP mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    })
    
    // Run the service
    err := runtimex.Run(ctx, nil, runtimex.Options{
        Logger: logger,
        HTTP: &runtimex.HTTPOptions{
            Addr: ":8080",
            H2C:  true,
            Mux:  mux,
        },
        Health:  &runtimex.Endpoint{Addr: ":8081"},
        Metrics: &runtimex.Endpoint{Addr: ":9091"},
        ShutdownTimeout: 15 * time.Second,
    })
    
    if err != nil {
        log.Fatal(err)
    }
}
```

## API Reference

### Types

#### Options

```go
type Options struct {
    Logger          log.Logger     // Logger for runtime operations
    HTTP            *HTTPOptions   // HTTP server configuration
    Health          *Endpoint      // Health check endpoint
    Metrics         *Endpoint      // Metrics endpoint
    ShutdownTimeout time.Duration  // Graceful shutdown timeout
}
```

#### HTTPOptions

```go
type HTTPOptions struct {
    Addr string           // Server address (e.g., ":8080")
    H2C  bool            // Enable HTTP/2 Cleartext
    Mux  *http.ServeMux  // HTTP request multiplexer
    TLS  *TLSConfig      // TLS configuration (optional)
}
```

#### Endpoint

```go
type Endpoint struct {
    Addr string // Endpoint address (e.g., ":8081")
}
```

#### TLSConfig

```go
type TLSConfig struct {
    CertFile string // Certificate file path
    KeyFile  string // Private key file path
}
```

### Functions

```go
// Run starts the runtime with the given options
func Run(ctx context.Context, cancel context.CancelFunc, opts Options) error

// Shutdown gracefully shuts down the runtime
func Shutdown(ctx context.Context, timeout time.Duration) error
```

## Usage Examples

### Basic HTTP Service

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Create HTTP mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", handleRoot)
    mux.HandleFunc("/api/users", handleUsers)
    
    // Run the service
    err := runtimex.Run(ctx, cancel, runtimex.Options{
        Logger: logger,
        HTTP: &runtimex.HTTPOptions{
            Addr: ":8080",
            H2C:  true,
            Mux:  mux,
        },
        Health:  &runtimex.Endpoint{Addr: ":8081"},
        Metrics: &runtimex.Endpoint{Addr: ":9091"},
        ShutdownTimeout: 15 * time.Second,
    })
    
    if err != nil {
        logger.Error(err, "Runtime failed")
        os.Exit(1)
    }
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Service is running"))
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
    // Handle users API
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"users": []}`))
}
```

### Connect Service Integration

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Create Connect service
    service := &UserService{}
    path, handler := userv1connect.NewUserServiceHandler(
        service,
        connect.WithInterceptors(interceptors...),
    )
    
    // Create HTTP mux
    mux := http.NewServeMux()
    mux.Handle(path, handler)
    
    // Add health check
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })
    
    // Run the service
    err := runtimex.Run(ctx, cancel, runtimex.Options{
        Logger: logger,
        HTTP: &runtimex.HTTPOptions{
            Addr: ":8080",
            H2C:  true,
            Mux:  mux,
        },
        Health:  &runtimex.Endpoint{Addr: ":8081"},
        Metrics: &runtimex.Endpoint{Addr: ":9091"},
        ShutdownTimeout: 15 * time.Second,
    })
    
    if err != nil {
        logger.Error(err, "Runtime failed")
        os.Exit(1)
    }
}
```

### TLS Configuration

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Create HTTP mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", handleRoot)
    
    // Run the service with TLS
    err := runtimex.Run(ctx, cancel, runtimex.Options{
        Logger: logger,
        HTTP: &runtimex.HTTPOptions{
            Addr: ":8443",
            H2C:  false,
            Mux:  mux,
            TLS: &runtimex.TLSConfig{
                CertFile: "/path/to/cert.pem",
                KeyFile:  "/path/to/key.pem",
            },
        },
        Health:  &runtimex.Endpoint{Addr: ":8081"},
        Metrics: &runtimex.Endpoint{Addr: ":9091"},
        ShutdownTimeout: 15 * time.Second,
    })
    
    if err != nil {
        logger.Error(err, "Runtime failed")
        os.Exit(1)
    }
}
```

### Custom Health Checks

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Create HTTP mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", handleRoot)
    
    // Add custom health check
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        // Check database connection
        if err := checkDatabase(); err != nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            w.Write([]byte("Database unavailable"))
            return
        }
        
        // Check external services
        if err := checkExternalServices(); err != nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            w.Write([]byte("External services unavailable"))
            return
        }
        
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })
    
    // Run the service
    err := runtimex.Run(ctx, cancel, runtimex.Options{
        Logger: logger,
        HTTP: &runtimex.HTTPOptions{
            Addr: ":8080",
            H2C:  true,
            Mux:  mux,
        },
        Health:  &runtimex.Endpoint{Addr: ":8081"},
        Metrics: &runtimex.Endpoint{Addr: ":9091"},
        ShutdownTimeout: 15 * time.Second,
    })
    
    if err != nil {
        logger.Error(err, "Runtime failed")
        os.Exit(1)
    }
}

func checkDatabase() error {
    // Implement database health check
    return nil
}

func checkExternalServices() error {
    // Implement external services health check
    return nil
}
```

### Metrics Integration

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Create HTTP mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", handleRoot)
    
    // Add metrics endpoint
    mux.Handle("/metrics", promhttp.Handler())
    
    // Run the service
    err := runtimex.Run(ctx, cancel, runtimex.Options{
        Logger: logger,
        HTTP: &runtimex.HTTPOptions{
            Addr: ":8080",
            H2C:  true,
            Mux:  mux,
        },
        Health:  &runtimex.Endpoint{Addr: ":8081"},
        Metrics: &runtimex.Endpoint{Addr: ":9091"},
        ShutdownTimeout: 15 * time.Second,
    })
    
    if err != nil {
        logger.Error(err, "Runtime failed")
        os.Exit(1)
    }
}
```

## Configuration Integration

### With ConfigX

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Load configuration
    var cfg AppConfig
    if err := configManager.Bind(&cfg); err != nil {
        logger.Error(err, "Failed to bind configuration")
        os.Exit(1)
    }
    
    // Create HTTP mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", handleRoot)
    
    // Run the service with configuration
    err := runtimex.Run(ctx, cancel, runtimex.Options{
        Logger: logger,
        HTTP: &runtimex.HTTPOptions{
            Addr: cfg.HTTPPort,
            H2C:  true,
            Mux:  mux,
        },
        Health:  &runtimex.Endpoint{Addr: cfg.HealthPort},
        Metrics: &runtimex.Endpoint{Addr: cfg.MetricsPort},
        ShutdownTimeout: cfg.ShutdownTimeout,
    })
    
    if err != nil {
        logger.Error(err, "Runtime failed")
        os.Exit(1)
    }
}
```

## Testing

```go
func TestRuntime(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Create test mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("test"))
    })
    
    // Start runtime in goroutine
    go func() {
        err := runtimex.Run(ctx, cancel, runtimex.Options{
            Logger: &TestLogger{},
            HTTP: &runtimex.HTTPOptions{
                Addr: ":0", // Use random port
                H2C:  true,
                Mux:  mux,
            },
            Health:  &runtimex.Endpoint{Addr: ":0"},
            Metrics: &runtimex.Endpoint{Addr: ":0"},
            ShutdownTimeout: 5 * time.Second,
        })
        assert.NoError(t, err)
    }()
    
    // Wait for service to start
    time.Sleep(100 * time.Millisecond)
    
    // Test HTTP endpoint
    resp, err := http.Get("http://localhost:8080/")
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    // Cancel context to shutdown
    cancel()
    
    // Wait for shutdown
    time.Sleep(100 * time.Millisecond)
}

type TestLogger struct{}

func (l *TestLogger) With(kv ...any) log.Logger { return l }
func (l *TestLogger) Debug(msg string, kv ...any) {}
func (l *TestLogger) Info(msg string, kv ...any) {}
func (l *TestLogger) Warn(msg string, kv ...any) {}
func (l *TestLogger) Error(err error, msg string, kv ...any) {}
```

## Best Practices

### 1. Proper Context Management

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Handle signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        cancel()
    }()
    
    // Run service
    err := runtimex.Run(ctx, cancel, opts)
    if err != nil {
        logger.Error(err, "Runtime failed")
        os.Exit(1)
    }
}
```

### 2. Health Check Implementation

```go
func healthCheck(w http.ResponseWriter, r *http.Request) {
    // Check critical dependencies
    if err := checkDatabase(); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("Database unavailable"))
        return
    }
    
    if err := checkCache(); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("Cache unavailable"))
        return
    }
    
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}
```

### 3. Graceful Shutdown

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Run service
    err := runtimex.Run(ctx, cancel, runtimex.Options{
        // ... options
        ShutdownTimeout: 30 * time.Second, // Allow time for cleanup
    })
    
    if err != nil {
        logger.Error(err, "Runtime failed")
        os.Exit(1)
    }
}
```

### 4. Error Handling

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Run service
    err := runtimex.Run(ctx, cancel, opts)
    if err != nil {
        logger.Error(err, "Runtime failed")
        os.Exit(1)
    }
}
```

## Thread Safety

The runtime is designed to be thread-safe and can handle concurrent requests safely.

## Dependencies

- **Go 1.21+** required
- **Standard library** only
- **No external dependencies**

## Version Compatibility

- **Go 1.21+** required
- **API Stability**: Stable (L2 module)
- **Breaking Changes**: None planned

## Contributing

Contributions are welcome! Please see the main project [Contributing Guide](../CONTRIBUTING.md) for details.

## License

This package is part of the EggyByte framework and is licensed under the MIT License.