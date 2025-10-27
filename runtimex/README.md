# egg/runtimex

## Overview

`runtimex` provides runtime lifecycle management and unified port strategy for
egg microservices. It handles service startup, shutdown, and graceful termination
with support for HTTP, RPC, health, and metrics endpoints.

## Key Features

- Service lifecycle management (Start/Stop)
- Concurrent service startup with error handling
- Graceful shutdown with configurable timeout
- Multiple server support (HTTP, RPC, Health, Metrics)
- HTTP/2 and HTTP/2 Cleartext (h2c) support
- Health check aggregation and registration
- Clean separation of runtime logic from public API

## Dependencies

Layer: **L3 (Runtime Communication Layer)**  
Depends on: `core/log`

## Installation

```bash
go get go.eggybyte.com/egg/runtimex@latest
```

## Basic Usage

```go
import (
    "context"
    "go.eggybyte.com/egg/runtimex"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    })
    
    err := runtimex.Run(ctx, nil, runtimex.Options{
        Logger: logger,
        HTTP: &runtimex.HTTPOptions{
            Port: 8080,
            Mux:  mux,
        },
        Health: &runtimex.Endpoint{Port: 8081},
        ShutdownTimeout: 15 * time.Second,
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

## Configuration Options

| Option            | Type                  | Description                                |
| ----------------- | --------------------- | ------------------------------------------ |
| `Logger`          | `log.Logger`          | Logger instance (required)                 |
| `HTTP`            | `*HTTPOptions`        | HTTP server configuration                  |
| `RPC`             | `*RPCOptions`         | RPC server configuration (optional)        |
| `Health`          | `*Endpoint`           | Health check endpoint                      |
| `Metrics`         | `*Endpoint`           | Metrics endpoint                           |
| `ShutdownTimeout` | `time.Duration`       | Graceful shutdown timeout (default: 15s)   |

### HTTPOptions

| Field | Type            | Description                             |
| ----- | --------------- | --------------------------------------- |
| `Port`| `int`           | Port number (e.g., 8080)                |
| `H2C` | `bool`          | Enable HTTP/2 Cleartext support         |
| `Mux` | `*http.ServeMux`| HTTP request multiplexer                |

### Endpoint

| Field | Type     | Description                       |
| ----- | -------- | --------------------------------- |
| `Port`| `int`    | Port number (e.g., 8081)          |

## API Reference

### Service Interface

```go
// Service defines the interface for services that can be started and stopped
type Service interface {
    // Start begins the service operation
    Start(ctx context.Context) error
    
    // Stop gracefully shuts down the service
    Stop(ctx context.Context) error
}
```

### Main Function

```go
// Run starts all services and manages their lifecycle
func Run(ctx context.Context, services []Service, opts Options) error
```

### Health Check API

```go
// HealthChecker defines the interface for health checks
type HealthChecker interface {
    Name() string
    Check(ctx context.Context) error
}

// RegisterHealthChecker registers a global health checker
func RegisterHealthChecker(checker HealthChecker)

// CheckHealth runs all registered health checkers
func CheckHealth(ctx context.Context) error

// ClearHealthCheckers clears all registered health checkers (for testing)
func ClearHealthCheckers()
```

## Architecture

The runtimex module follows a clean architecture pattern:

```
runtimex/
├── runtimex.go          # Public API (~100 lines)
│   ├── Service          # Service interface
│   ├── Options          # Configuration structs
│   ├── Run()            # Main entry point (delegates to internal)
│   └── Health APIs      # Health check wrappers
└── internal/
    ├── runtime.go       # Runtime implementation (~206 lines)
    │   ├── Start()      # Concurrent service startup
    │   └── Stop()       # Graceful shutdown
    └── health.go        # Health check registry (~50 lines)
        ├── RegisterHealthChecker()
        └── CheckHealth()
```

**Design Highlights:**
- Public interface provides simple Run() entry point
- Complex runtime management isolated in internal package
- Health check system completely decoupled
- Service interface allows custom service implementations

## Example: Complete Service

```go
package main

import (
    "context"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    
    "go.eggybyte.com/egg/logx"
    "go.eggybyte.com/egg/runtimex"
)

func main() {
    // Create logger
    logger := logx.New(
        logx.WithFormat(logx.FormatLogfmt),
        logx.WithColor(true),
    )
    
    // Create context with signal handling
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigChan
        logger.Info("received shutdown signal")
        cancel()
    }()
    
    // Create HTTP mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    })
    mux.HandleFunc("/api/users", handleUsers)
    
    // Run service
    err := runtimex.Run(ctx, nil, runtimex.Options{
        Logger: logger,
        HTTP: &runtimex.HTTPOptions{
            Port: 8080,
            Mux:  mux,
        },
        Health: &runtimex.Endpoint{Port: 8081},
        Metrics: &runtimex.Endpoint{Port: 9091},
        ShutdownTimeout: 15 * time.Second,
    })
    if err != nil {
        logger.Error(err, "service failed")
        os.Exit(1)
    }
}
```

## Example: Custom Service Implementation

```go
type DatabaseService struct {
    db     *sql.DB
    logger log.Logger
}

func (s *DatabaseService) Start(ctx context.Context) error {
    s.logger.Info("connecting to database")
    
    var err error
    s.db, err = sql.Open("postgres", "connection-string")
    if err != nil {
        return fmt.Errorf("failed to open database: %w", err)
    }
    
    if err := s.db.PingContext(ctx); err != nil {
        return fmt.Errorf("failed to ping database: %w", err)
    }
    
    s.logger.Info("database connected")
    return nil
}

func (s *DatabaseService) Stop(ctx context.Context) error {
    s.logger.Info("closing database connection")
    
    if err := s.db.Close(); err != nil {
        return fmt.Errorf("failed to close database: %w", err)
    }
    
    s.logger.Info("database connection closed")
    return nil
}

func main() {
    logger := logx.New()
    dbService := &DatabaseService{logger: logger}
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    err := runtimex.Run(ctx, []runtimex.Service{dbService}, runtimex.Options{
        Logger: logger,
        HTTP: &runtimex.HTTPOptions{Port: 8080, Mux: http.NewServeMux()},
        ShutdownTimeout: 10 * time.Second,
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

## Example: Health Check Registration

```go
type DatabaseHealthChecker struct {
    db *sql.DB
}

func (c *DatabaseHealthChecker) Name() string {
    return "database"
}

func (c *DatabaseHealthChecker) Check(ctx context.Context) error {
    return c.db.PingContext(ctx)
}

func main() {
    // Register health checker
    dbChecker := &DatabaseHealthChecker{db: db}
    runtimex.RegisterHealthChecker(dbChecker)
    
    // Health check endpoint will automatically include this checker
    err := runtimex.Run(ctx, nil, runtimex.Options{
        Logger: logger,
        HTTP:   &runtimex.HTTPOptions{Port: 8080, Mux: mux},
        Health: &runtimex.Endpoint{Port: 8081},
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

## Lifecycle Flow

```
1. Run() called
2. Services started concurrently
3. HTTP/RPC/Health/Metrics servers started
4. Wait for context cancellation
5. Shutdown triggered
6. Services stopped concurrently (with timeout)
7. Servers shutdown gracefully
8. Run() returns
```

**Startup:**
- All services start concurrently
- If any service fails to start, Run() returns immediately
- Servers start in background goroutines

**Shutdown:**
- Triggered by context cancellation
- Services stopped in reverse order
- Shutdown timeout prevents hanging
- Servers gracefully drain connections

## Health Check Integration

The built-in health check endpoint (`/healthz`) automatically aggregates all
registered health checkers:

```bash
# Check service health
curl http://localhost:8081/healthz

# Response (200 OK if all checks pass)
{"status": "healthy"}

# Response (503 Service Unavailable if any check fails)
{"status": "unhealthy", "error": "database: connection timeout"}
```

## Split Port vs Single Port Strategy

**Split Port (Recommended):**
```go
runtimex.Options{
    HTTP:   &runtimex.HTTPOptions{Port: 8080, Mux: mux},
    Health: &runtimex.Endpoint{Port: 8081},
    Metrics: &runtimex.Endpoint{Port: 9091},
}
```

Benefits:
- Separate network policies for application, health, and metrics
- Health checks don't compete with application traffic
- Easier Kubernetes liveness/readiness probe configuration

**Single Port (Alternative):**
```go
mux := http.NewServeMux()
mux.Handle("/healthz", healthHandler)
mux.Handle("/metrics", metricsHandler)
mux.Handle("/api/", apiHandler)

runtimex.Options{
    HTTP: &runtimex.HTTPOptions{Port: 8080, Mux: mux},
}
```

## Testing

```go
func TestServiceLifecycle(t *testing.T) {
    logger := logx.New(logx.WithWriter(io.Discard))
    
    // Create test service
    service := &TestService{}
    
    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // Run in goroutine
    errChan := make(chan error, 1)
    go func() {
        errChan <- runtimex.Run(ctx, []runtimex.Service{service}, runtimex.Options{
            Logger: logger,
            HTTP: &runtimex.HTTPOptions{Port: 0, Mux: http.NewServeMux()},
            ShutdownTimeout: 1 * time.Second,
        })
    }()
    
    // Wait a bit for startup
    time.Sleep(100 * time.Millisecond)
    
    // Cancel context
    cancel()
    
    // Wait for shutdown
    err := <-errChan
    assert.NoError(t, err)
}
```

## Stability

**Status**: Stable  
**Layer**: L3 (Runtime Communication)  
**API Guarantees**: Backward-compatible changes only

The runtimex module is production-ready and follows semantic versioning.

## License

This package is part of the egg framework and is licensed under the MIT License.
See the root LICENSE file for details.
