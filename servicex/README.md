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
- Graceful shutdown with hooks
- Dependency injection container
- Clean multi-stage initialization

## Dependencies

Layer: **L4 (Integration Layer)**  
Depends on: `configx`, `logx`, `obsx`, `connectx`, `storex`, `runtimex`

## Installation

```bash
go get github.com/eggybyte-technology/egg/servicex@latest
```

## Basic Usage

```go
import (
    "context"
    "github.com/eggybyte-technology/egg/servicex"
)

type AppConfig struct {
    servicex.BaseConfig
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
    cfg := &AppConfig{}
    
    err := servicex.Run(ctx,
        servicex.WithConfig(cfg),
        servicex.WithRegister(register),
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

## Configuration Options

| Option                    | Description                                      |
| ------------------------- | ------------------------------------------------ |
| `WithService(name, ver)`  | Set service name and version                     |
| `WithConfig(cfg)`         | Set configuration struct                         |
| `WithLogger(logger)`      | Set custom logger                                |
| `WithTracing(enabled)`    | Enable OpenTelemetry tracing                     |
| `WithMetrics(enabled)`    | Enable metrics collection                        |
| `WithRegister(fn)`        | Set service registration function                |
| `WithTimeout(ms)`         | Set default RPC timeout in milliseconds          |
| `WithSlowRequestThreshold(ms)` | Set slow request warning threshold          |
| `WithShutdownTimeout(dur)`| Set graceful shutdown timeout                    |
| `WithDebugLogs(enabled)`  | Enable debug-level logging                       |
| `WithDatabase(cfg)`       | Enable database support                          |
| `WithAutoMigrate(models...)`| Auto-migrate database models                   |

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
    "github.com/eggybyte-technology/egg/configx"
    "github.com/eggybyte-technology/egg/servicex"
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

## Best Practices

1. **Configuration**: Always use `configx.BaseConfig` as embedded struct
2. **Logging**: Use the logger from `app.Logger()`, never create new instances
3. **Database**: Use `app.MustDB()` if database is required, `app.DB()` if optional
4. **Shutdown**: Register cleanup in shutdown hooks for graceful termination
5. **Dependencies**: Use DI container for complex dependency graphs
6. **Testing**: Use short timeouts and random ports for tests

## Stability

**Status**: Stable  
**Layer**: L4 (Integration)  
**API Guarantees**: Backward-compatible changes only

The servicex module is production-ready and follows semantic versioning.

## License

This package is part of the egg framework and is licensed under the MIT License.
See the root LICENSE file for details.
