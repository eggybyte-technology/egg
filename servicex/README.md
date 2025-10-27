# egg/servicex

## Overview

`servicex` is the unified microservice initialization framework for egg. It provides
a one-line service startup with integrated configuration, logging, database, tracing,
and Connect RPC support. This is the highest-level integration layer that brings all
egg components together.

## Key Features

- One-line service startup with `Run()`
- Integrated configuration management (configx)
- Automatic logging setup (logx)
- Optional database support with migrations (storex)
- OpenTelemetry tracing integration (obsx)
- Connect RPC interceptors (connectx)
- Health check endpoints
- **Prometheus metrics endpoint** (automatically exposed on port 9091)
- Graceful shutdown with hooks
- Dependency injection container
- Clean multi-stage initialization

## Dependencies

Layer: **L4 (Integration Layer)**  
Depends on: `configx`, `logx`, `obsx`, `connectx`, `storex`, `runtimex`

## Installation

```bash
go get go.eggybyte.com/egg/servicex@latest
```

## Basic Usage

```go
import (
    "context"
    "log/slog"
    "go.eggybyte.com/egg/configx"
    "go.eggybyte.com/egg/logx"
    "go.eggybyte.com/egg/servicex"
)

type AppConfig struct {
    configx.BaseConfig  // Includes Database, HTTP/Health/Metrics ports
    CustomField string `env:"CUSTOM_FIELD" default:"value"`
}

func register(app *servicex.App) error {
    // Get logger
    logger := app.Logger()
    
    // Register your service handlers
    handler := myhandler.New(logger, app.DB())
    
    // Get interceptors and bind to mux
    path, svcHandler := userv1connect.NewUserServiceHandler(
        handler,
        connect.WithInterceptors(app.Interceptors()...),
    )
    app.Mux().Handle(path, svcHandler)
    
    return nil
}

func main() {
    ctx := context.Background()
    
    // Create logger (optional - servicex creates one if not provided)
    logger := logx.New(
        logx.WithFormat(logx.FormatConsole),
        logx.WithLevel(slog.LevelInfo),
        logx.WithColor(true),
    )
    
    cfg := &AppConfig{}
    
    err := servicex.Run(ctx,
        servicex.WithService("my-service", "1.0.0"),
        servicex.WithLogger(logger),
        servicex.WithAppConfig(cfg), // Auto-detects database from BaseConfig
        servicex.WithAutoMigrate(&model.User{}),
        servicex.WithRegister(register),
    )
    if err != nil {
        logger.Error(err, "service failed to start")
    }
}
```

Run with:
```bash
# Default log level (info)
go run main.go

# With debug logging
LOG_LEVEL=debug go run main.go

# With database
DB_DRIVER=mysql DB_DSN=user:pass@tcp(localhost:3306)/mydb go run main.go
```

## Configuration Options

| Option                    | Description                                      |
| ------------------------- | ------------------------------------------------ |
| `WithService(name, ver)`  | Set service name and version                     |
| `WithConfig(cfg)`         | Set configuration struct (use `WithAppConfig` for BaseConfig) |
| `WithAppConfig(cfg)`      | **Recommended**: Set config + auto-detect database from BaseConfig |
| `WithLogger(logger)`      | Set custom logger (optional, creates default if not provided) |
| `WithTracing(enabled)`    | Enable OpenTelemetry tracing                     |
| `WithMetrics(enabled)`    | Enable metrics collection                        |
| `WithRegister(fn)`        | Set service registration function                |
| `WithTimeout(ms)`         | Set default RPC timeout in milliseconds          |
| `WithSlowRequestThreshold(ms)` | Set slow request warning threshold          |
| `WithShutdownTimeout(dur)`| Set graceful shutdown timeout                    |
| `WithDebugLogs(enabled)`  | **Deprecated**: Use `LOG_LEVEL` environment variable instead |
| `WithDatabase(cfg)`       | Enable database support (auto-detected by `WithAppConfig`) |
| `WithAutoMigrate(models...)`| Auto-migrate database models                   |

### Environment Variables

| Variable              | Description                          | Default  | Example                    |
| --------------------- | ------------------------------------ | -------- | -------------------------- |
| `LOG_LEVEL`           | Log level (debug/info/warn/error)    | `info`   | `LOG_LEVEL=debug`          |
| `SERVICE_NAME`        | Service name                         | -        | `SERVICE_NAME=user-service`|
| `SERVICE_VERSION`     | Service version                      | -        | `SERVICE_VERSION=1.0.0`    |
| `ENV`                 | Environment (dev/staging/production) | `dev`    | `ENV=production`           |
| `DB_DRIVER`           | Database driver (mysql/postgres)     | -        | `DB_DRIVER=mysql`          |
| `DB_DSN`              | Database connection string           | -        | `DB_DSN=user:pass@tcp(...)`|
| `DB_MAX_IDLE`         | Max idle connections                 | `10`     | `DB_MAX_IDLE=20`           |
| `DB_MAX_OPEN`         | Max open connections                 | `100`    | `DB_MAX_OPEN=200`          |
| `DB_MAX_LIFETIME`     | Connection max lifetime              | `1h`     | `DB_MAX_LIFETIME=30m`      |
| `HTTP_PORT`           | HTTP server port                     | `8080`   | `HTTP_PORT=9000`           |
| `HEALTH_PORT`         | Health check port                    | `8081`   | `HEALTH_PORT=9001`         |
| `METRICS_PORT`        | Metrics endpoint port                | `9091`   | `METRICS_PORT=9002`        |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry collector endpoint | -   | `otel-collector:4317`      |

## Database Configuration Auto-Detection

When using `WithAppConfig()` with a config struct that embeds `configx.BaseConfig`, servicex automatically detects and configures the database after environment variables are loaded.

**How it works:**

1. `WithAppConfig(cfg)` registers your configuration struct
2. During initialization, `configx.Manager.Bind()` loads environment variables into `cfg`
3. servicex extracts `BaseConfig.Database` fields after binding
4. If `DB_DSN` is set, database connection is initialized automatically

**Example:**

```go
type AppConfig struct {
    configx.BaseConfig  // Contains Database field
    CustomSetting string `env:"CUSTOM_SETTING"`
}

func main() {
    cfg := &AppConfig{}
    
    // Database config is auto-detected from environment variables
    servicex.Run(ctx,
        servicex.WithAppConfig(cfg), // Automatically handles database
        servicex.WithAutoMigrate(&model.User{}),
        servicex.WithRegister(register),
    )
}
```

**Environment variables:**
```bash
DB_DRIVER=mysql
DB_DSN=user:password@tcp(mysql:3306)/mydb?charset=utf8mb4&parseTime=True
DB_MAX_IDLE=10
DB_MAX_OPEN=100
DB_MAX_LIFETIME=1h
```

**Migration from old pattern:**

```go
// Old pattern (still works, but verbose)
servicex.WithConfig(cfg),
servicex.WithDatabase(servicex.FromBaseConfig(&cfg.Database)),

// New pattern (recommended)
servicex.WithAppConfig(cfg), // Auto-detects database from BaseConfig
```

## Log Level Control

servicex supports environment-based log level control through the `LOG_LEVEL` environment variable.

**Supported levels:**
- `debug` - Detailed debugging information
- `info` - General informational messages (default)
- `warn` - Warning messages
- `error` - Error messages only

**Usage:**

```bash
# Set log level via environment
LOG_LEVEL=debug go run main.go

# Or in Docker Compose
environment:
  LOG_LEVEL: debug
```

**Priority:**

1. If `WithDebugLogs(true)` is used → `debug` level (deprecated)
2. If `LOG_LEVEL` environment variable is set → use that level
3. Otherwise → `info` level (default)

**Migration from `WithDebugLogs`:**

```go
// Old pattern (deprecated)
servicex.Run(ctx,
    servicex.WithDebugLogs(true),
    // ...
)

// New pattern (recommended)
// Just set LOG_LEVEL=debug environment variable
servicex.Run(ctx,
    // No WithDebugLogs needed
    // ...
)
```

**Custom logger:**

If you provide a custom logger via `WithLogger()`, its log level takes precedence:

```go
logger := logx.New(
    logx.WithLevel(slog.LevelDebug), // This level is used
)

servicex.Run(ctx,
    servicex.WithLogger(logger), // Custom logger level takes precedence
    // ...
)
```

## API Reference

### App Type

```go
type App struct {
    // Public methods only
}

// Mux returns the HTTP mux for handler registration
func (a *App) Mux() *http.ServeMux

// Logger returns the logger instance
func (a *App) Logger() log.Logger

// Interceptors returns the configured Connect interceptors
func (a *App) Interceptors() []connect.Interceptor

// OtelProvider returns the OpenTelemetry provider (may be nil)
func (a *App) OtelProvider() *obsx.Provider

// Provide registers a constructor in the DI container
func (a *App) Provide(constructor any) error

// Resolve resolves a dependency from the DI container
func (a *App) Resolve(target any) error

// AddShutdownHook registers a shutdown hook (executed in LIFO order)
func (a *App) AddShutdownHook(hook func(context.Context) error)

// DB returns the GORM database instance or nil if not configured
func (a *App) DB() *gorm.DB

// MustDB returns the GORM database instance or panics
func (a *App) MustDB() *gorm.DB
```

### Main Function

```go
// Run starts the service with the given options
func Run(ctx context.Context, opts ...Option) error
```

### Database Config

```go
type DatabaseConfig struct {
    Driver          string        `env:"DB_DRIVER" default:"mysql"`
    DSN             string        `env:"DB_DSN" default:""`
    MaxIdleConns    int           `env:"DB_MAX_IDLE" default:"10"`
    MaxOpenConns    int           `env:"DB_MAX_OPEN" default:"100"`
    ConnMaxLifetime time.Duration `env:"DB_MAX_LIFETIME" default:"1h"`
    PingTimeout     time.Duration `env:"DB_PING_TIMEOUT" default:"5s"`
}

// FromBaseConfig converts configx.DatabaseConfig to servicex.DatabaseConfig
func FromBaseConfig(dbCfg *configx.DatabaseConfig) *DatabaseConfig
```

## Observability

servicex provides built-in observability features out of the box:

### Metrics Endpoint

When `ENABLE_METRICS=true` (default) and OpenTelemetry is initialized, servicex automatically starts a metrics server that exposes Prometheus-compatible metrics at `/metrics`.

**Default Configuration:**
- **Port**: 9091 (configurable via `METRICS_PORT`)
- **Format**: Prometheus text exposition format
- **Path**: `/metrics`
- **Export**: Dual mode - both local Prometheus endpoint and OTLP export to collector

**Access metrics:**
```bash
# Check if metrics are being collected
curl http://localhost:9091/metrics

# Use with Prometheus scrape config
scrape_configs:
  - job_name: 'my-service'
    static_configs:
      - targets: ['localhost:9091']
```

**What's exported:**
- Service metadata (name, version via `target_info`)
- **RPC metrics** (automatically collected by connectx interceptor):
  - `rpc_requests_total`: Counter of RPC requests by service, method, and code
  - `rpc_request_duration_seconds`: Histogram of request durations in seconds
  - `rpc_request_size_bytes`: Histogram of request payload sizes in bytes
  - `rpc_response_size_bytes`: Histogram of response payload sizes in bytes
- OpenTelemetry instrumentation metrics
- Custom application metrics (when using OTel Meter API)

**Example RPC metrics:**
```prometheus
# Total requests
rpc_requests_total{rpc_code="ok",rpc_method="GetUser",rpc_service="user.v1.UserService"} 523

# Request duration
rpc_request_duration_seconds_bucket{rpc_code="ok",rpc_method="GetUser",rpc_service="user.v1.UserService",le="0.01"} 498
rpc_request_duration_seconds_sum{rpc_code="ok",rpc_method="GetUser",rpc_service="user.v1.UserService"} 3.124
rpc_request_duration_seconds_count{rpc_code="ok",rpc_method="GetUser",rpc_service="user.v1.UserService"} 523
```

**Querying metrics:**
```promql
# Request rate by service
sum(rate(rpc_requests_total[5m])) by (rpc_service)

# Error rate
sum(rate(rpc_requests_total{rpc_code!="ok"}[5m])) / sum(rate(rpc_requests_total[5m]))

# P95 latency by service
histogram_quantile(0.95, sum(rate(rpc_request_duration_seconds_bucket[5m])) by (rpc_service, le))
```

**Disable metrics:**
```bash
# Disable metrics server
ENABLE_METRICS=false go run main.go
```

**Note**: The metrics endpoint works alongside OTLP export. If `OTEL_EXPORTER_OTLP_ENDPOINT` is set, metrics are sent to both the local Prometheus endpoint and the OTLP collector.

### Tracing

OpenTelemetry tracing is enabled by default (`ENABLE_TRACING=true`). Traces are exported to the OTLP collector if configured:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317 go run main.go
```

### Health Checks

Health check endpoints are always available on the health port (default 8081):
- `/health` - Overall health status
- `/ready` - Readiness probe (includes database connectivity)
- `/live` - Liveness probe (always returns OK)

```bash
curl http://localhost:8081/health
curl http://localhost:8081/ready
curl http://localhost:8081/live
```

## Architecture

The servicex module follows a multi-stage initialization pattern:

```
servicex/
├── servicex.go          # Public API (~280 lines)
│   ├── App              # Application context
│   ├── Options          # Configuration functions
│   └── Run()            # Entry point (delegates to internal)
└── internal/
    ├── runtime.go       # Runtime lifecycle (~330 lines)
    │   ├── initializeLogger()
    │   ├── initializeConfig()
    │   ├── initializeDatabase()
    │   ├── initializeObservability()
    │   ├── buildApp()
    │   ├── startServers()
    │   └── gracefulShutdown()
    ├── config.go        # Configuration types
    ├── container.go     # DI container
    ├── health.go        # Health check setup
    └── interceptors.go  # Interceptor builders
```

**Design Highlights:**
- Public interface provides simple Run() entry point
- Complex initialization split into logical stages
- Each stage handles one concern (logger, config, db, observability)
- Clean error handling and rollback on failure
- Graceful shutdown with resource cleanup

## Initialization Flow

```
1. initializeLogger()      → Setup logging (logx)
2. initializeConfig()       → Load configuration (configx)
3. initializeDatabase()     → Connect database + migrations (storex)
4. initializeObservability()→ Setup tracing/metrics (obsx)
5. buildApp()               → Create App with interceptors (connectx)
6. User register()          → Register service handlers
7. startServers()           → Start HTTP + health servers
8. Wait for shutdown        → Block until context cancelled
9. gracefulShutdown()       → Cleanup in LIFO order
```

## Example: Complete Service

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    
    "connectrpc.com/connect"
    "go.eggybyte.com/egg/configx"
    "go.eggybyte.com/egg/servicex"
    userv1 "myapp/gen/go/user/v1"
    userv1connect "myapp/gen/go/user/v1/userv1connect"
)

type AppConfig struct {
    configx.BaseConfig
    
    JWTSecret string `env:"JWT_SECRET" default:"secret"`
    SMTPHost  string `env:"SMTP_HOST" default:"localhost"`
}

type User struct {
    ID    uint   `gorm:"primarykey"`
    Name  string `gorm:"not null"`
    Email string `gorm:"uniqueIndex;not null"`
}

func register(app *servicex.App) error {
    // Get components
    logger := app.Logger()
    db := app.MustDB()
    
    // Create handler
    handler := &UserServiceHandler{
        logger: logger,
        db:     db,
    }
    
    // Register Connect service
    path, svcHandler := userv1connect.NewUserServiceHandler(
        handler,
        connect.WithInterceptors(app.Interceptors()...),
    )
    app.Mux().Handle(path, svcHandler)
    
    logger.Info("user service registered", "path", path)
    return nil
}

func main() {
    // Create context with signal handling
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigChan
        cancel()
    }()
    
    // Load configuration
    cfg := &AppConfig{}
    
    // Run service
    err := servicex.Run(ctx,
        servicex.WithService("user-service", "1.0.0"),
        servicex.WithConfig(cfg),
        servicex.WithDatabase(servicex.FromBaseConfig(&cfg.Database)),
        servicex.WithAutoMigrate(&User{}),
        servicex.WithTracing(true),
        servicex.WithRegister(register),
        servicex.WithDebugLogs(cfg.Env == "dev"),
    )
    if err != nil {
        os.Exit(1)
    }
}
```

## Example: With Dependency Injection

```go
type UserRepository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
    return &UserRepository{db: db}
}

type UserService struct {
    repo   *UserRepository
    logger log.Logger
}

func NewUserService(repo *UserRepository, logger log.Logger) *UserService {
    return &UserService{
        repo:   repo,
        logger: logger,
    }
}

func register(app *servicex.App) error {
    // Register constructors
    app.Provide(NewUserRepository)
    app.Provide(NewUserService)
    app.Provide(func() *gorm.DB { return app.MustDB() })
    app.Provide(func() log.Logger { return app.Logger() })
    
    // Resolve service
    var userService *UserService
    if err := app.Resolve(&userService); err != nil {
        return err
    }
    
    // Register handler
    handler := NewUserServiceHandler(userService)
    path, svcHandler := userv1connect.NewUserServiceHandler(
        handler,
        connect.WithInterceptors(app.Interceptors()...),
    )
    app.Mux().Handle(path, svcHandler)
    
    return nil
}
```

## Example: With Shutdown Hooks

```go
func register(app *servicex.App) error {
    // Create background worker
    worker := NewBackgroundWorker(app.Logger())
    
    // Start worker
    go worker.Start()
    
    // Register shutdown hook
    app.AddShutdownHook(func(ctx context.Context) error {
        app.Logger().Info("stopping background worker")
        return worker.Stop(ctx)
    })
    
    // Register handlers...
    return nil
}
```

## Example: Custom HTTP Endpoints

```go
func register(app *servicex.App) error {
    mux := app.Mux()
    
    // Register Connect service
    path, svcHandler := userv1connect.NewUserServiceHandler(
        handler,
        connect.WithInterceptors(app.Interceptors()...),
    )
    mux.Handle(path, svcHandler)
    
    // Add custom REST endpoints
    mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"ok"}`))
    })
    
    mux.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, `{"version":"%s"}`, cfg.ServiceVersion)
    })
    
    return nil
}
```

## Database Migrations

```go
// Define models
type User struct {
    ID        uint      `gorm:"primarykey"`
    CreatedAt time.Time
    UpdatedAt time.Time
    Name      string    `gorm:"not null"`
    Email     string    `gorm:"uniqueIndex;not null"`
}

type Post struct {
    ID        uint      `gorm:"primarykey"`
    CreatedAt time.Time
    UpdatedAt time.Time
    UserID    uint      `gorm:"not null;index"`
    Title     string    `gorm:"not null"`
    Content   string    `gorm:"type:text"`
    User      User      `gorm:"foreignKey:UserID"`
}

func main() {
    cfg := &AppConfig{}
    
    err := servicex.Run(ctx,
        servicex.WithConfig(cfg),
        servicex.WithDatabase(servicex.FromBaseConfig(&cfg.Database)),
        servicex.WithAutoMigrate(&User{}, &Post{}),  // Auto-migrate on startup
        servicex.WithRegister(register),
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

## Environment Variables

```bash
# Service Configuration
SERVICE_NAME=user-service
SERVICE_VERSION=1.0.0
ENV=production

# Server Ports
HTTP_PORT=8080
HEALTH_PORT=8081
METRICS_PORT=9091

# Database
DB_DRIVER=mysql
DB_DSN=user:pass@tcp(mysql:3306)/mydb?parseTime=true
DB_MAX_IDLE=10
DB_MAX_OPEN=100
DB_MAX_LIFETIME=1h

# Observability
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317

# Logging
ENABLE_DEBUG_LOGS=false

# RPC
SLOW_REQUEST_MILLIS=1000
```

## Health Checks

servicex automatically provides health check endpoints:

```bash
# Liveness probe (always returns 200 OK when server is running)
curl http://localhost:8081/healthz

# Readiness probe (checks database and other dependencies)
curl http://localhost:8081/readyz
```

Custom health checks can be registered via `runtimex.RegisterHealthChecker()`.

## Integration with Other Modules

servicex automatically integrates:

- **configx**: Loads configuration from environment + optional ConfigMap
- **logx**: Sets up structured logging with logfmt format
- **obsx**: Initializes OpenTelemetry provider if tracing enabled
- **connectx**: Builds default interceptor stack (timeout, logging, errors, metrics)
- **storex**: Connects to database and runs migrations
- **runtimex**: Manages HTTP server lifecycle and graceful shutdown

## Testing

```go
func TestService(t *testing.T) {
    // Create test context
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    // Set test environment
    os.Setenv("HTTP_PORT", "0")  // Random port
    os.Setenv("HEALTH_PORT", "0")
    defer func() {
        os.Unsetenv("HTTP_PORT")
        os.Unsetenv("HEALTH_PORT")
    }()
    
    // Create test config
    cfg := &AppConfig{}
    
    // Run service in goroutine
    errChan := make(chan error, 1)
    go func() {
        errChan <- servicex.Run(ctx,
            servicex.WithConfig(cfg),
            servicex.WithRegister(testRegister),
            servicex.WithDebugLogs(true),
        )
    }()
    
    // Wait for startup
    time.Sleep(100 * time.Millisecond)
    
    // Test service...
    
    // Shutdown
    cancel()
    err := <-errChan
    assert.NoError(t, err)
}
```

## Performance Considerations

- Database connection pooling configured via `MaxIdleConns` and `MaxOpenConns`
- Slow request logging threshold configurable
- OpenTelemetry sampling ratio defaults to 10% (configurable)
- HTTP/2 support for Connect RPC
- Graceful shutdown ensures in-flight requests complete

## Troubleshooting

### Database Connection Issues

**Problem**: "DSN is required" error

**Solution**: Ensure `DB_DSN` environment variable is set:
```bash
DB_DRIVER=mysql
DB_DSN=user:password@tcp(localhost:3306)/mydb?charset=utf8mb4&parseTime=True
```

If using `WithAppConfig()`, make sure your config embeds `configx.BaseConfig`:
```go
type AppConfig struct {
    configx.BaseConfig  // Required for auto-detection
    // ...
}
```

**Problem**: Database connection fails but DSN is set

**Solution**: Check that:
1. Database server is running and accessible
2. Credentials are correct
3. Database name exists
4. Network connectivity (especially in Docker)

### Log Level Not Working

**Problem**: `LOG_LEVEL=debug` doesn't show debug logs

**Solution**: 
1. Ensure no custom logger with different level is provided via `WithLogger()`
2. Remove deprecated `WithDebugLogs()` calls
3. Verify environment variable is actually set: `echo $LOG_LEVEL`

### Service Startup Fails

**Problem**: Service fails to start with unclear error

**Solution**: Enable debug logging to see detailed initialization:
```bash
LOG_LEVEL=debug go run main.go
```

Check the initialization stages:
- Logger initialization
- Configuration loading
- Database connection
- Observability setup
- Server startup

## Best Practices

1. **Configuration**: Always use `configx.BaseConfig` as embedded struct for auto-detection
2. **Logging**: Use `LOG_LEVEL` environment variable instead of `WithDebugLogs()`
3. **Database**: Use `WithAppConfig()` for automatic database configuration
4. **Shutdown**: Register cleanup in shutdown hooks for graceful termination
5. **Dependencies**: Use DI container for complex dependency graphs
6. **Testing**: Use short timeouts and random ports for tests
7. **Migration**: Prefer `WithAppConfig()` over separate `WithConfig()` + `WithDatabase()`

## Stability

**Status**: Stable  
**Layer**: L4 (Integration)  
**API Guarantees**: Backward-compatible changes only

The servicex module is production-ready and follows semantic versioning.

## License

This package is part of the egg framework and is licensed under the MIT License.
See the root LICENSE file for details.
