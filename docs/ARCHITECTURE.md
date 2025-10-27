# Egg Framework Architecture

**Version**: 0.2.0  
**Last Updated**: October 25, 2025  
**Status**: Production-Ready

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Design Philosophy](#design-philosophy)
3. [Layered Architecture](#layered-architecture)
4. [Module Catalog](#module-catalog)
5. [Core Design Patterns](#core-design-patterns)
6. [Data Flow & Integration](#data-flow--integration)
7. [Configuration Management](#configuration-management)
8. [Observability Architecture](#observability-architecture)
9. [Database Integration](#database-integration)
10. [Security Considerations](#security-considerations)
11. [Performance Characteristics](#performance-characteristics)
12. [Deployment Architecture](#deployment-architecture)
13. [Evolution & Roadmap](#evolution--roadmap)

---

## Executive Summary

Egg is a production-ready Go microservices framework designed for building Connect-RPC services with comprehensive observability, configuration management, and clean architecture. The framework emphasizes:

- **Minimal Boilerplate**: One-line service startup with `servicex.Run()`
- **Clean Architecture**: Strict layered design preventing circular dependencies
- **Production-Ready**: Built-in observability, health checks, graceful shutdown
- **Developer Experience**: Environment-based configuration, hot reloading, comprehensive documentation
- **Type Safety**: Leveraging Go's type system for compile-time guarantees

### Key Metrics

- **Lines of Code**: ~15,000 (excluding examples and tests)
- **Module Count**: 13 core modules + 3 examples
- **Test Coverage**: >80% across core modules
- **Zero External Dependencies**: Core layer (L0) is completely self-contained
- **Startup Time**: <100ms for minimal service

---

## Design Philosophy

### 1. Clarity Over Cleverness

Code should be explicit and readable. We avoid:
- Magic behavior and hidden side effects
- Reflection-heavy patterns (except where necessary for config binding)
- Complex generic abstractions
- Implicit global state

### 2. Layered Dependencies

Strict dependency flow prevents circular imports and ensures maintainability:

```
L4 (Integration) → L3 (Runtime) → L2 (Capability) → L1 (Foundation) → L0 (Core)
```

**Rule**: A module can only depend on modules in the same or lower layers.

### 3. Interface-Driven Design

Public APIs are defined as interfaces, with implementations hidden in `internal/` packages:

```go
// Public API (module.go)
type Manager interface {
    Bind(target any) error
    Snapshot() map[string]string
}

// Implementation (internal/manager.go)
type managerImpl struct { /* ... */ }
```

### 4. Production-Ready Defaults

Services should work out-of-the-box with sensible defaults:
- Automatic health checks
- Prometheus metrics
- Structured logging
- OpenTelemetry tracing (when configured)
- Graceful shutdown

### 5. CLI-Driven Development

Never manually edit `go.mod` or `go.work`. Use CLI commands:

```bash
go work init ./module1 ./module2
go get example.com/lib@latest
go mod tidy
```

---

## Layered Architecture

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│  L4: Integration Layer                                      │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  servicex: One-line service startup                   │  │
│  │  - Configuration management (configx)                 │  │
│  │  - Logging setup (logx)                               │  │
│  │  - Database initialization (storex)                   │  │
│  │  - Observability (obsx)                               │  │
│  │  - Connect interceptors (connectx)                    │  │
│  │  - Lifecycle management (runtimex)                    │  │
│  └───────────────────────────────────────────────────────┘  │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────┴────────────────────────────────────┐
│  L3: Runtime & Communication Layer                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  runtimex    │  │  connectx    │  │  clientx     │      │
│  │  - Lifecycle │  │  - RPC       │  │  - HTTP      │      │
│  │  - Health    │  │  - Intercept │  │  - Retry     │      │
│  │  - Shutdown  │  │  - Tracing   │  │  - Circuit   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────┴────────────────────────────────────┐
│  L2: Capability Layer                                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  configx     │  │  obsx        │  │  httpx       │      │
│  │  - Multi-src │  │  - OTEL      │  │  - Binding   │      │
│  │  - Hot reload│  │  - Tracing   │  │  - Security  │      │
│  │  - Validation│  │  - Metrics   │  │  - Middleware│      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────┴────────────────────────────────────┐
│  L1: Foundation Layer                                       │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  logx: Structured logging (slog-based)                │  │
│  │  - Logfmt, JSON, Console formats                      │  │
│  │  - Field sorting, colorization                        │  │
│  │  - Sensitive field masking                            │  │
│  └───────────────────────────────────────────────────────┘  │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────┴────────────────────────────────────┐
│  L0: Core Layer (Zero Dependencies)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  errors      │  │  identity    │  │  log         │      │
│  │  - Codes     │  │  - Context   │  │  - Interface │      │
│  │  - Wrapping  │  │  - Metadata  │  │  - Helpers   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  Auxiliary Modules (Can depend on any layer)                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  storex      │  │  k8sx        │  │  testingx    │      │
│  │  - GORM      │  │  - ConfigMap │  │  - Helpers   │      │
│  │  - Health    │  │  - Discovery │  │  - Mocks     │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

### Layer Responsibilities

#### L0: Core Layer
- **Zero external dependencies** (only Go stdlib)
- **Interfaces and types** used across all layers
- **No business logic** - pure abstractions

#### L1: Foundation Layer
- **Logging implementation** based on `log/slog`
- **Depends only on** L0 and stdlib
- **Provides** structured logging with multiple formats

#### L2: Capability Layer
- **Horizontal capabilities** like configuration, observability, HTTP utilities
- **Depends on** L0, L1, and minimal external libraries
- **Provides** reusable building blocks for services

#### L3: Runtime & Communication Layer
- **Service lifecycle management** and RPC communication
- **Depends on** L0, L1, L2
- **Provides** runtime orchestration and Connect-RPC integration

#### L4: Integration Layer
- **Highest-level orchestration** - brings everything together
- **Depends on** all lower layers
- **Provides** one-line service startup with `servicex.Run()`

#### Auxiliary Modules
- **Cross-cutting concerns** like storage and Kubernetes integration
- **Can depend on any layer** as needed
- **Optional** - not required for basic services

---

## Module Catalog

### L0: Core Modules

#### core/errors
**Purpose**: Structured error handling with error codes

**Key Types**:
```go
type Error struct {
    Code    string
    Message string
    Details map[string]any
    Cause   error
}
```

**Features**:
- Error code constants (e.g., `ErrNotFound`, `ErrInvalidInput`)
- Error wrapping with context
- Connect-RPC error code mapping

#### core/identity
**Purpose**: Request metadata and user identity

**Key Types**:
```go
type Identity struct {
    UserID    string
    UserName  string
    Roles     []string
    RequestID string
    Metadata  map[string]string
}
```

**Features**:
- Context-based identity storage
- Header extraction helpers
- Request correlation

#### core/log
**Purpose**: Zero-dependency logger interface

**Key Interface**:
```go
type Logger interface {
    With(kv ...any) Logger
    Debug(msg string, kv ...any)
    Info(msg string, kv ...any)
    Warn(msg string, kv ...any)
    Error(err error, msg string, kv ...any)
}
```

**Features**:
- Structured logging helpers (`Str`, `Int`, `Int32`, `Int64`, `Float64`, `Bool`, `Dur`, `Time`)
- Context-aware logging
- Compatible with `log/slog` concepts

---

### L1: Foundation Layer

#### logx
**Purpose**: Production-ready structured logging

**Formats**:
- **Logfmt**: Machine-readable, grep-friendly
- **JSON**: Structured, for log aggregators
- **Console**: Human-readable with colors

**Features**:
- Field sorting for consistency
- Sensitive field masking (passwords, tokens)
- Payload size limiting
- Colorization for development
- Context propagation

**Configuration**:
```go
logger := logx.New(
    logx.WithFormat(logx.FormatConsole),
    logx.WithLevel(slog.LevelInfo),
    logx.WithColor(true),
    logx.WithSensitiveFields("password", "token"),
)
```

**Integration**: Automatically configured by `servicex` based on `LOG_LEVEL` environment variable.

---

### L2: Capability Layer

#### configx
**Purpose**: Unified configuration management with hot reloading

**Architecture**:
```
┌─────────────┐
│   Manager   │
└──────┬──────┘
       │
       ├─→ EnvSource (environment variables)
       ├─→ FileSource (YAML, JSON files)
       └─→ K8sConfigMapSource (Kubernetes ConfigMaps)
```

**Key Features**:
- **Multi-source merging**: Later sources override earlier ones
- **Hot reload**: Debounced updates from ConfigMaps
- **Struct binding**: Automatic type conversion with `env` tags
- **Validation**: Custom validation logic
- **Change notifications**: Callbacks on configuration updates

**BaseConfig**:
```go
type BaseConfig struct {
    ServiceName    string `env:"SERVICE_NAME" default:"app"`
    ServiceVersion string `env:"SERVICE_VERSION" default:"0.0.0"`
    Env            string `env:"ENV" default:"dev"`
    
    HTTPPort    string `env:"HTTP_PORT" default:":8080"`
    HealthPort  string `env:"HEALTH_PORT" default:":8081"`
    MetricsPort string `env:"METRICS_PORT" default:":9091"`
    
    Database DatabaseConfig  // Auto-detected by servicex
}
```

**Usage Pattern**:
```go
type AppConfig struct {
    configx.BaseConfig
    CustomField string `env:"CUSTOM_FIELD" default:"value"`
}

// servicex automatically creates manager and binds config
servicex.Run(ctx, servicex.WithAppConfig(&cfg))
```

#### obsx
**Purpose**: OpenTelemetry provider initialization

**Components**:
- **Tracer Provider**: Distributed tracing
- **Meter Provider**: Metrics collection
- **Resource Attributes**: Service metadata

**Features**:
- OTLP exporter configuration
- Sampling ratio control (default: 10%)
- Graceful shutdown
- Automatic span context propagation

**Integration**:
```go
// Automatic via servicex
servicex.Run(ctx, servicex.WithAppConfig(cfg))
// Set OTEL_EXPORTER_OTLP_ENDPOINT to enable

// Manual
provider, _ := obsx.NewProvider(ctx, obsx.Options{
    ServiceName:    "user-service",
    ServiceVersion: "1.0.0",
    OTLPEndpoint:   "otel-collector:4317",
})
```

#### httpx
**Purpose**: HTTP utilities and middleware

**Features**:
- Request binding and validation
- Security headers middleware
- CORS configuration
- Request/response helpers

---

### L3: Runtime & Communication Layer

#### runtimex
**Purpose**: Service lifecycle management

**Responsibilities**:
- Concurrent server startup (HTTP, Health, Metrics)
- Health check aggregation
- Graceful shutdown with timeout
- Signal handling (SIGTERM, SIGINT)

**Architecture**:
```go
type Runtime struct {
    HTTP    *HTTPOptions    // Main HTTP server
    Health  *Endpoint       // Health check endpoint
    Metrics *Endpoint       // Metrics endpoint
    Services []Service      // Additional services
}
```

**Shutdown Flow**:
1. Receive termination signal
2. Stop accepting new requests
3. Wait for in-flight requests (with timeout)
4. Execute shutdown hooks (LIFO order)
5. Close all servers

#### connectx
**Purpose**: Connect-RPC interceptor stack

**Interceptors** (in order):
1. **Timeout**: Enforce request deadlines
2. **Logging**: Structured request/response logging
3. **Metrics**: Request counters and latency histograms
4. **Tracing**: OpenTelemetry span creation
5. **Error Mapping**: Convert errors to Connect codes

**Features**:
- Header-based timeout override (`X-Timeout-Ms`)
- Slow request warnings
- Request/response body logging (debug mode)
- Payload size accounting
- Correlation ID propagation

**Configuration**:
```go
interceptors := connectx.DefaultInterceptors(connectx.Options{
    Logger:            logger,
    Otel:              provider,
    SlowRequestMillis: 1000,
    WithRequestBody:   true,  // Debug mode
    WithResponseBody:  true,
})
```

#### clientx
**Purpose**: Resilient HTTP client factory

**Features**:
- Configurable timeouts
- Exponential backoff retry
- Circuit breaker pattern
- Connection pooling

**Usage**:
```go
client := clientx.NewHTTPClient("https://api.example.com",
    clientx.WithTimeout(5*time.Second),
    clientx.WithRetry(3),
    clientx.WithCircuitBreaker(5, 10*time.Second),
)
```

---

### L4: Integration Layer

#### servicex
**Purpose**: One-line service startup with complete integration

**Initialization Stages**:
```
1. initializeLogger()       → Setup logging (LOG_LEVEL or default)
2. initializeConfig()        → Load configuration (configx)
3. initializeDatabase()      → Connect database (storex)
4. initializeObservability() → Setup tracing (obsx)
5. buildApp()                → Create app context
6. startServers()            → Start HTTP servers (runtimex)
7. gracefulShutdown()        → Cleanup on termination
```

**App Context**:
```go
type App struct {
    Mux           *http.ServeMux
    Logger        log.Logger
    Interceptors  []connect.Interceptor
    OtelProvider  *obsx.Provider
    Container     *Container  // DI container
    ShutdownHooks []func(context.Context) error
    DB            *gorm.DB
}
```

**Key Features**:
- **Auto-detection**: Database config from `BaseConfig` after env binding
- **Log level control**: Via `LOG_LEVEL` environment variable
- **DI container**: Type-safe dependency injection
- **Shutdown hooks**: LIFO cleanup on termination
- **Health checks**: Automatic `/health` endpoint
- **Metrics**: Automatic `/metrics` endpoint

**Usage Pattern**:
```go
func main() {
    ctx := context.Background()
    cfg := &AppConfig{}
    
    servicex.Run(ctx,
        servicex.WithService("user-service", "1.0.0"),
        servicex.WithAppConfig(cfg),
        servicex.WithAutoMigrate(&model.User{}),
        servicex.WithRegister(registerServices),
    )
}

func registerServices(app *servicex.App) error {
    // Get dependencies
    logger := app.Logger()
    db := app.DB()
    
    // Create service
    repo := repository.NewUserRepository(db)
    svc := service.NewUserService(repo, logger)
    handler := handler.NewUserHandler(svc, logger)
    
    // Register Connect handler
    path, h := userv1connect.NewUserServiceHandler(
        handler,
        connect.WithInterceptors(app.Interceptors()...),
    )
    app.Mux().Handle(path, h)
    
    return nil
}
```

---

### Auxiliary Modules

#### storex
**Purpose**: Storage abstraction and GORM integration

**Interfaces**:
```go
type Store interface {
    Ping(ctx context.Context) error
    Close() error
}

type GORMStore interface {
    Store
    GetDB() *gorm.DB
    AutoMigrate(models ...any) error
}
```

**Features**:
- Connection pooling configuration
- Health check support
- Auto-migration
- Multi-database support (MySQL, PostgreSQL, SQLite)

**Integration with servicex**:
```go
// Automatic via WithAppConfig
servicex.Run(ctx,
    servicex.WithAppConfig(cfg),
    servicex.WithAutoMigrate(&User{}),
)

// Access in register function
func register(app *servicex.App) error {
    db := app.DB()  // *gorm.DB or nil
    // ...
}
```

#### k8sx
**Purpose**: Kubernetes integration

**Features**:
- ConfigMap watching with hot reload
- Service discovery (headless and ClusterIP)
- In-cluster and out-of-cluster configuration

**Usage**:
```go
watcher := k8sx.NewConfigMapWatcher(k8sx.Options{
    Namespace:     "default",
    ConfigMapName: "app-config",
    Logger:        logger,
})

updates, _ := watcher.Watch(ctx)
for snapshot := range updates {
    // Handle configuration update
}
```

#### testingx
**Purpose**: Testing utilities (planned)

**Planned Features**:
- Test server helpers
- Mock implementations
- Assertion utilities
- Integration test helpers

---

## Core Design Patterns

### 1. Functional Options Pattern

All modules use functional options for configuration:

```go
type Option func(*Config)

func WithTimeout(d time.Duration) Option {
    return func(c *Config) {
        c.Timeout = d
    }
}

// Usage
NewClient(
    WithTimeout(5*time.Second),
    WithRetry(3),
)
```

**Benefits**:
- Backward compatibility (new options don't break existing code)
- Clear intent at call site
- Optional parameters with defaults

### 2. Interface-Implementation Separation

Public API defines interfaces, implementation is internal:

```
module/
├── module.go              # Public interfaces and constructors
└── internal/
    ├── implementation.go  # Actual implementation
    └── helpers.go         # Internal helpers
```

**Benefits**:
- Minimal public API surface
- Easy to test (mock interfaces)
- Implementation can change without breaking users

### 3. Multi-Stage Initialization

Complex initialization split into logical stages:

```go
type Runtime struct {
    config *Config
    logger log.Logger
    db     *gorm.DB
}

func (r *Runtime) Run(ctx context.Context) error {
    if err := r.initializeLogger(); err != nil {
        return err
    }
    if err := r.initializeConfig(ctx); err != nil {
        return err
    }
    if err := r.initializeDatabase(ctx); err != nil {
        return err
    }
    return r.startServers(ctx)
}
```

**Benefits**:
- Clear initialization order
- Easy to debug (each stage is isolated)
- Graceful error handling

### 4. Context Propagation

All I/O operations accept `context.Context` as first parameter:

```go
func (s *Service) GetUser(ctx context.Context, id string) (*User, error) {
    // Extract identity
    identity := identity.FromContext(ctx)
    
    // Extract logger with correlation
    logger := log.FromContext(ctx)
    logger.Info("getting user", log.Str("user_id", id))
    
    // Pass context to repository
    return s.repo.GetUser(ctx, id)
}
```

**Benefits**:
- Request cancellation
- Deadline propagation
- Correlation ID tracking
- Distributed tracing

### 5. Error Wrapping

Errors are wrapped with context using `fmt.Errorf`:

```go
func (r *Repository) GetUser(ctx context.Context, id string) (*User, error) {
    var user User
    if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, core.NewError(core.ErrNotFound, "user not found")
        }
        return nil, fmt.Errorf("failed to get user %s: %w", id, err)
    }
    return &user, nil
}
```

**Benefits**:
- Error context preserved
- Stack trace available
- Error classification

---

## Data Flow & Integration

### Request Flow (Connect RPC)

```
1. Client Request
   ↓
2. HTTP/2 Server (runtimex)
   ↓
3. Connect Handler (generated)
   ↓
4. Interceptor Chain (connectx)
   ├─→ Timeout Interceptor
   ├─→ Logging Interceptor
   ├─→ Metrics Interceptor
   ├─→ Tracing Interceptor (obsx)
   └─→ Error Mapping Interceptor
   ↓
5. Service Handler (user code)
   ├─→ Service Layer (business logic)
   ├─→ Repository Layer (data access)
   └─→ Database (storex + GORM)
   ↓
6. Response (reverse through interceptors)
   ↓
7. Client Response
```

### Configuration Flow

```
1. Environment Variables
   ↓
2. configx.EnvSource
   ↓
3. configx.Manager (merging)
   ↓
4. configx.Manager.Bind() → AppConfig struct
   ↓
5. servicex extracts BaseConfig.Database
   ↓
6. storex.NewGORMStore() → Database connection
   ↓
7. app.DB() available in register function
```

### Observability Flow

```
1. Request arrives
   ↓
2. connectx Tracing Interceptor
   ├─→ Create span (obsx)
   ├─→ Extract parent span from headers
   └─→ Inject span into context
   ↓
3. Service code
   ├─→ Logger from context (includes trace_id)
   ├─→ Metrics recorded (connectx)
   └─→ Child spans created (if needed)
   ↓
4. Response
   ├─→ Span finished
   ├─→ Metrics updated
   └─→ Logs correlated via trace_id
```

---

## Configuration Management

### Configuration Hierarchy

```
1. Defaults (in struct tags)
   ↓
2. Environment Variables (highest priority)
   ↓
3. Configuration Files (if specified)
   ↓
4. Kubernetes ConfigMaps (if in cluster)
```

### BaseConfig Structure

```go
type BaseConfig struct {
    // Service Identity
    ServiceName    string `env:"SERVICE_NAME" default:"app"`
    ServiceVersion string `env:"SERVICE_VERSION" default:"0.0.0"`
    Env            string `env:"ENV" default:"dev"`
    
    // Network Ports
    HTTPPort    string `env:"HTTP_PORT" default:":8080"`
    HealthPort  string `env:"HEALTH_PORT" default:":8081"`
    MetricsPort string `env:"METRICS_PORT" default:":9091"`
    
    // Observability
    OTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" default:""`
    
    // Configuration Management
    ConfigMapName  string `env:"APP_CONFIGMAP_NAME" default:""`
    DebounceMillis int    `env:"CONFIG_DEBOUNCE_MS" default:"200"`
    
    // Database (auto-detected by servicex)
    Database DatabaseConfig
}

type DatabaseConfig struct {
    Driver      string        `env:"DB_DRIVER" default:"mysql"`
    DSN         string        `env:"DB_DSN" default:""`
    MaxIdle     int           `env:"DB_MAX_IDLE" default:"10"`
    MaxOpen     int           `env:"DB_MAX_OPEN" default:"100"`
    MaxLifetime time.Duration `env:"DB_MAX_LIFETIME" default:"1h"`
}
```

### Hot Reload Mechanism

```
1. K8s ConfigMap updated
   ↓
2. k8sx watcher detects change
   ↓
3. Debounce timer (200ms default)
   ↓
4. configx.Manager merges new values
   ↓
5. Registered callbacks invoked
   ↓
6. Application reacts to changes
```

**Debouncing**: Prevents flapping when multiple keys updated simultaneously.

---

## Observability Architecture

### Three Pillars

#### 1. Logging (logx)

**Structured Logging**:
```go
logger.Info("user created",
    log.Str("user_id", user.ID),
    log.Str("email", user.Email),
    log.Int64("timestamp", time.Now().Unix()),
)
```

**Output Formats**:
- **Logfmt**: `level=INFO msg="user created" user_id=u-123 email=user@example.com`
- **JSON**: `{"level":"INFO","msg":"user created","user_id":"u-123","email":"user@example.com"}`
- **Console**: Colored, human-readable with indentation

**Correlation**:
- Request ID propagated via context
- Trace ID included in logs (when tracing enabled)
- User identity attached to logger

#### 2. Metrics (Prometheus + OpenTelemetry)

**Dual Export Architecture**:
- **Local Prometheus endpoint**: Pull-based metrics at `/metrics` (port 9091)
- **OTLP export**: Push-based metrics to OpenTelemetry Collector (if configured)

**Automatic Setup** (via servicex):
- Prometheus endpoint automatically started when `ENABLE_METRICS=true` (default)
- Metrics server runs on dedicated port (default: 9091)
- Both local and OTLP export work simultaneously

**Metrics Exported**:
- Service metadata (`target_info` with name, version)
- OpenTelemetry instrumentation metrics
- Custom application metrics via Meter API

**Custom Metrics**:
```go
meter := provider.MeterProvider().Meter("user-service")
counter, _ := meter.Int64Counter("user.operations")
counter.Add(ctx, 1, attribute.String("operation", "create"))
```

**Access Metrics**:
```bash
# Local Prometheus endpoint
curl http://localhost:9091/metrics

# Or via OTLP Collector's Prometheus endpoint
curl http://otel-collector:8889/metrics
```

**Format**: Prometheus text exposition format (OpenMetrics compatible)

#### 3. Tracing (OpenTelemetry)

**Automatic Tracing** (via connectx):
- Span created for each RPC call
- Parent span extracted from headers
- Span context propagated to downstream services

**Span Attributes**:
- `rpc.system`: "connect"
- `rpc.service`: Service name
- `rpc.method`: Method name
- `http.status_code`: Response code
- `error`: Error message (if failed)

**Sampling**:
- Default: 10% of requests traced
- Configurable via `TraceSamplerRatio`
- Always-on for errors

---

## Database Integration

### GORM Integration (storex)

**Connection Pooling**:
```go
type DatabaseConfig struct {
    MaxIdle     int           // Max idle connections (default: 10)
    MaxOpen     int           // Max open connections (default: 100)
    MaxLifetime time.Duration // Connection max lifetime (default: 1h)
}
```

**Health Checks**:
```go
func (s *GORMStore) Ping(ctx context.Context) error {
    sqlDB, _ := s.db.DB()
    return sqlDB.PingContext(ctx)
}
```

**Auto-Migration**:
```go
servicex.Run(ctx,
    servicex.WithAppConfig(cfg),
    servicex.WithAutoMigrate(&User{}, &Order{}, &Product{}),
)
```

### Repository Pattern

**Recommended Structure**:
```go
type UserRepository interface {
    GetUser(ctx context.Context, id string) (*User, error)
    CreateUser(ctx context.Context, user *User) error
    UpdateUser(ctx context.Context, user *User) error
    DeleteUser(ctx context.Context, id string) error
    ListUsers(ctx context.Context, page, size int) ([]*User, error)
}

type userRepository struct {
    db     *gorm.DB
    logger log.Logger
}

func (r *userRepository) GetUser(ctx context.Context, id string) (*User, error) {
    var user User
    if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, core.NewError(core.ErrNotFound, "user not found")
        }
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    return &user, nil
}
```

**Benefits**:
- Testable (mock repository interface)
- Database-agnostic service layer
- Clear separation of concerns

---

## Security Considerations

### 1. Sensitive Data Handling

**Log Masking**:
```go
logger := logx.New(
    logx.WithSensitiveFields("password", "token", "secret", "apiKey"),
)

// "password" field will be masked as "***"
logger.Info("user login", "username", "john", "password", "secret123")
// Output: level=INFO msg="user login" username=john password=***
```

**DSN Masking**:
```go
// servicex automatically masks passwords in DSN logs
// Input:  user:password@tcp(localhost:3306)/db
// Logged: user:***@tcp(localhost:3306)/db
```

### 2. Authentication & Authorization

**Identity Extraction**:
```go
// Extract from headers
identity := identity.FromContext(ctx)

// Check permissions
if !identity.HasRole("admin") {
    return connect.NewError(connect.CodePermissionDenied, 
        errors.New("admin role required"))
}
```

**Header Mapping**:
```go
type HeaderMapping struct {
    RequestID     string // "X-Request-Id"
    InternalToken string // "X-Internal-Token"
    UserID        string // "X-User-Id"
    UserName      string // "X-User-Name"
    Roles         string // "X-User-Roles"
}
```

### 3. Input Validation

**Struct Tags**:
```go
type CreateUserRequest struct {
    Name  string `validate:"required,min=1,max=100"`
    Email string `validate:"required,email"`
}
```

**Validation in Handler**:
```go
func (h *Handler) CreateUser(ctx context.Context, req *connect.Request[CreateUserRequest]) (*connect.Response[CreateUserResponse], error) {
    if err := validate.Struct(req.Msg); err != nil {
        return nil, connect.NewError(connect.CodeInvalidArgument, err)
    }
    // ...
}
```

### 4. Rate Limiting (Future)

Planned integration with rate limiting middleware.

---

## Performance Characteristics

### Startup Performance

**Minimal Service** (no database):
- Cold start: ~50ms
- With tracing: ~80ms

**Full Service** (with database):
- Cold start: ~100ms
- Database connection: ~20ms
- Auto-migration: ~50ms (first run)

### Runtime Performance

**Request Latency** (p99):
- Interceptor overhead: <1ms
- Logging: <0.5ms
- Tracing: <0.3ms (when enabled)

**Memory Usage**:
- Minimal service: ~20MB
- With database: ~50MB
- Per request: ~10KB (without large payloads)

**Throughput**:
- Simple RPC: >10,000 req/s (single core)
- With database: ~5,000 req/s (depends on DB)

### Optimization Strategies

1. **Connection Pooling**: Configure `MaxOpen` and `MaxIdle` based on load
2. **Sampling**: Reduce trace sampling ratio for high-traffic services
3. **Log Level**: Use `info` in production, `debug` only when needed
4. **Payload Logging**: Disable request/response body logging in production

---

## Deployment Architecture

### Container Deployment

**Dockerfile Pattern**:
```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /service cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /service /service
EXPOSE 8080 8081 9091
CMD ["/service"]
```

**Multi-Stage Benefits**:
- Small image size (~20MB)
- No build dependencies in runtime
- Security (minimal attack surface)

### Kubernetes Deployment

**Deployment YAML**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: user-service
  template:
    metadata:
      labels:
        app: user-service
    spec:
      containers:
      - name: user-service
        image: user-service:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 8081
          name: health
        - containerPort: 9091
          name: metrics
        env:
        - name: LOG_LEVEL
          value: "info"
        - name: DB_DSN
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: dsn
        - name: OTEL_EXPORTER_OTLP_ENDPOINT
          value: "otel-collector:4317"
        livenessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
```

### Service Mesh Integration

**Istio Compatibility**:
- HTTP/2 support for Connect-RPC
- Automatic mTLS
- Traffic management
- Observability integration

**Linkerd Compatibility**:
- Lightweight proxy
- Automatic retries
- Load balancing

---

## Evolution & Roadmap

### Current Status (v0.2.0)

✅ **Stable**:
- Core interfaces (L0)
- Logging (logx)
- Configuration (configx)
- Connect integration (connectx)
- Service integration (servicex)

✅ **Production-Ready**:
- OpenTelemetry tracing (obsx)
- Database integration (storex)
- Runtime management (runtimex)

🚧 **Beta**:
- Kubernetes integration (k8sx)
- HTTP client (clientx)

### Planned Features (v0.3.0)

- **Redis integration**: Caching and session storage
- **Message queue**: Kafka/RabbitMQ integration
- **Rate limiting**: Token bucket and sliding window
- **Circuit breaker**: Enhanced failure handling
- **Service mesh**: Native Istio/Linkerd integration

### Long-Term Vision (v1.0.0)

- **GraphQL support**: Alongside Connect-RPC
- **gRPC compatibility**: Full gRPC server support
- **CLI tool**: Project scaffolding and code generation
- **Admin UI**: Service dashboard and configuration
- **Multi-tenancy**: Built-in tenant isolation

---

## Appendix

### A. Dependency Graph

```
servicex
├── configx
│   ├── core/log
│   └── k8sx (optional)
├── logx
│   └── core/log
├── obsx
│   └── core/log
├── connectx
│   ├── core/log
│   ├── core/identity
│   ├── core/errors
│   └── obsx
├── runtimex
│   └── core/log
└── storex
    └── core/log
```

### B. Port Allocation

| Service | HTTP | Health | Metrics |
|---------|------|--------|---------|
| Default | 8080 | 8081   | 9091    |
| Example 1 | 8080 | 8081 | 9091 |
| Example 2 | 8082 | 8083 | 9092 |

### C. Environment Variables Reference

See [servicex/README.md](servicex/README.md#environment-variables) for complete list.

### D. Performance Benchmarks

Run benchmarks:
```bash
go test -bench=. -benchmem ./...
```

### E. Contributing Guidelines

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and contribution process.

---

**Document Version**: 1.0  
**Framework Version**: 0.2.0  
**Last Updated**: October 25, 2025  
**Maintained By**: EggyByte Technology Team

