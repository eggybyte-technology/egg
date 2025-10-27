# egg/configx

## Overview

`configx` provides unified configuration management with hot reloading support for
egg microservices. It manages configuration from multiple sources (environment
variables, Kubernetes ConfigMaps, files) with automatic merging, validation, and
change notification.

## Key Features

- Multiple configuration sources with priority-based merging
- Hot reload with debouncing for Kubernetes ConfigMap changes
- Struct binding with `env` and `default` tags
- Change notification via callbacks
- Support for nested structs and embedded config
- Thread-safe concurrent access
- Clean separation of manager logic from public API

## Dependencies

Layer: **L2 (Capability Layer)**  
Depends on: `core/log`, optionally `k8s.io/client-go` for ConfigMap watching

## Installation

```bash
go get go.eggybyte.com/egg/configx@latest
```

## Basic Usage

```go
import (
    "context"
    "go.eggybyte.com/egg/configx"
)

type AppConfig struct {
    configx.BaseConfig
    
    DatabaseURL string `env:"DATABASE_URL" default:"postgres://localhost/mydb"`
    MaxConns    int    `env:"MAX_CONNS" default:"10"`
    Debug       bool   `env:"DEBUG" default:"false"`
}

func main() {
    ctx := context.Background()
    
    // Create config manager with default sources (Env + optional K8s)
    manager, err := configx.DefaultManager(ctx, logger)
    if err != nil {
        log.Fatal(err)
    }
    
    // Bind configuration
    var cfg AppConfig
    err = manager.Bind(&cfg)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use configuration
    fmt.Printf("Database URL: %s\n", cfg.DatabaseURL)
    fmt.Printf("Max Connections: %d\n", cfg.MaxConns)
}
```

## Configuration Options

### Manager Options

| Option     | Type             | Description                                      |
| ---------- | ---------------- | ------------------------------------------------ |
| `Logger`   | `log.Logger`     | Logger for configuration operations (required)   |
| `Sources`  | `[]Source`       | Configuration sources (later overrides earlier)  |
| `Debounce` | `time.Duration`  | Debounce duration for updates (default: 200ms)   |

### Source Types

| Source             | Description                                      |
| ------------------ | ------------------------------------------------ |
| `EnvSource`        | Environment variables                            |
| `FileSource`       | Configuration files (JSON, YAML, etc.)           |
| `K8sConfigMapSource`| Kubernetes ConfigMap with hot reload           |

## API Reference

### Manager Interface

```go
type Manager interface {
    // Snapshot returns a copy of the current merged configuration
    Snapshot() map[string]string
    
    // Value returns the value for a key
    Value(key string) (string, bool)
    
    // Bind decodes configuration into a struct
    Bind(target any, opts ...BindOption) error
    
    // OnUpdate subscribes to configuration update events
    OnUpdate(fn func(snapshot map[string]string)) (unsubscribe func())
}
```

### Source Interface

```go
type Source interface {
    // Load reads the current configuration snapshot
    Load(ctx context.Context) (map[string]string, error)
    
    // Watch monitors for updates and publishes snapshots
    Watch(ctx context.Context) (<-chan map[string]string, error)
}
```

### BaseConfig

`BaseConfig` provides standard configuration fields for egg services. It's designed to be embedded in your application configuration struct.

```go
type BaseConfig struct {
    ServiceName    string `env:"SERVICE_NAME" default:"app"`
    ServiceVersion string `env:"SERVICE_VERSION" default:"0.0.0"`
    Env            string `env:"ENV" default:"dev"`
    
    HTTPPort    string `env:"HTTP_PORT" default:":8080"`
    HealthPort  string `env:"HEALTH_PORT" default:":8081"`
    MetricsPort string `env:"METRICS_PORT" default:":9091"`
    
    OTLPEndpoint   string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" default:""`
    ConfigMapName  string `env:"APP_CONFIGMAP_NAME" default:""`
    DebounceMillis int    `env:"CONFIG_DEBOUNCE_MS" default:"200"`
    
    Database DatabaseConfig  // Auto-detected by servicex.WithAppConfig()
}

type DatabaseConfig struct {
    Driver      string        `env:"DB_DRIVER" default:"mysql"`
    DSN         string        `env:"DB_DSN" default:""`
    MaxIdle     int           `env:"DB_MAX_IDLE" default:"10"`
    MaxOpen     int           `env:"DB_MAX_OPEN" default:"100"`
    MaxLifetime time.Duration `env:"DB_MAX_LIFETIME" default:"1h"`
}
```

## Integration with servicex

When using `BaseConfig` with `servicex`, database configuration is automatically detected and used to initialize the database connection.

### Recommended Pattern

```go
import (
    "context"
    "go.eggybyte.com/egg/configx"
    "go.eggybyte.com/egg/servicex"
)

type AppConfig struct {
    configx.BaseConfig  // Includes Database, HTTP/Health/Metrics ports
    
    // Application-specific settings
    DefaultPageSize int `env:"DEFAULT_PAGE_SIZE" default:"10"`
    MaxPageSize     int `env:"MAX_PAGE_SIZE" default:"100"`
}

func main() {
    ctx := context.Background()
    cfg := &AppConfig{}
    
    // servicex automatically:
    // 1. Creates configx.Manager
    // 2. Binds environment variables to cfg
    // 3. Extracts Database config from BaseConfig
    // 4. Initializes database connection
    servicex.Run(ctx,
        servicex.WithService("my-service", "1.0.0"),
        servicex.WithAppConfig(cfg), // Auto-detects database from BaseConfig
        servicex.WithAutoMigrate(&model.User{}),
        servicex.WithRegister(registerServices),
    )
}
```

### How Auto-Detection Works

1. **Configuration Binding**: `servicex` creates a `configx.Manager` and calls `Bind(cfg)`
2. **Environment Loading**: All environment variables are loaded into `cfg.BaseConfig.Database`
3. **Database Extraction**: After binding, `servicex` extracts the `Database` field from `BaseConfig`
4. **Connection Initialization**: If `DB_DSN` is set, database connection is established

### Environment Variables

```bash
# Service configuration (from BaseConfig)
SERVICE_NAME=user-service
SERVICE_VERSION=1.0.0
ENV=production
HTTP_PORT=8080
HEALTH_PORT=8081
METRICS_PORT=9091

# Database configuration (from BaseConfig.Database)
DB_DRIVER=mysql
DB_DSN=user:password@tcp(mysql:3306)/mydb?charset=utf8mb4&parseTime=True
DB_MAX_IDLE=10
DB_MAX_OPEN=100
DB_MAX_LIFETIME=1h

# OpenTelemetry (from BaseConfig)
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
```

### Benefits

- **Zero boilerplate**: No manual database configuration code
- **Type-safe**: Configuration validated at startup
- **Hot-reloadable**: ConfigMap changes automatically applied
- **Consistent**: Same pattern across all services

### Migration from Manual Configuration

```go
// Old pattern (verbose)
manager, _ := configx.DefaultManager(ctx, logger)
manager.Bind(&cfg)
servicex.Run(ctx,
    servicex.WithConfig(cfg),
    servicex.WithDatabase(servicex.FromBaseConfig(&cfg.Database)),
    // ...
)

// New pattern (recommended)
servicex.Run(ctx,
    servicex.WithAppConfig(cfg), // Handles everything
    // ...
)
```

## Architecture

The configx module follows a clean architecture pattern:

```
configx/
├── configx.go           # Public API (~220 lines)
│   ├── Source           # Source interface
│   ├── Manager          # Manager interface
│   ├── BaseConfig       # Common config struct
│   └── Constructors     # NewManager, DefaultManager, etc.
└── internal/
    ├── manager.go       # Manager implementation (~240 lines)
    │   ├── loadInitial()    # Initial config load
    │   ├── startWatching()  # Watch all sources
    │   ├── watchSource()    # Watch single source
    │   └── applyUpdate()    # Merge updates
    ├── binding.go       # Struct binding (~115 lines)
    │   ├── BindToStruct()   # Main binding logic
    │   ├── bindStructFields()  # Recursive field binding
    │   └── setFieldValue()  # Type conversion
    ├── sources.go       # Source implementations
    └── builders.go      # Source builders
```

**Design Highlights:**
- Public interface defines contracts
- Complex merging logic isolated in internal package
- Binding logic supports nested structs and type conversion
- Hot reload debouncing prevents flapping

## Example: Multiple Sources

```go
func main() {
    ctx := context.Background()
    
    // Create sources with priority order
    sources := []configx.Source{
        configx.NewEnvSource(configx.EnvOptions{}),
        configx.NewFileSource("config.yaml", configx.FileOptions{
            Watch: true,
            Format: "yaml",
        }),
        configx.NewK8sConfigMapSource("app-config", configx.K8sOptions{
            Namespace: "default",
            Logger: logger,
        }),
    }
    
    // Create manager (later sources override earlier ones)
    manager, err := configx.NewManager(ctx, configx.Options{
        Logger: logger,
        Sources: sources,
        Debounce: 300 * time.Millisecond,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Bind configuration
    var cfg AppConfig
    err = manager.Bind(&cfg)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Example: Hot Reload with Callback

```go
func main() {
    ctx := context.Background()
    
    manager, err := configx.DefaultManager(ctx, logger)
    if err != nil {
        log.Fatal(err)
    }
    
    var cfg AppConfig
    var mu sync.RWMutex
    
    // Bind with update callback
    err = manager.Bind(&cfg, configx.WithUpdateCallback(func() {
        mu.Lock()
        defer mu.Unlock()
        
        // Re-bind on updates
        if err := manager.Bind(&cfg); err != nil {
            logger.Error(err, "failed to reload config")
            return
        }
        
        logger.Info("configuration reloaded",
            "database_url", cfg.DatabaseURL,
            "max_conns", cfg.MaxConns,
        )
    }))
    if err != nil {
        log.Fatal(err)
    }
    
    // Subscribe to raw updates
    unsubscribe := manager.OnUpdate(func(snapshot map[string]string) {
        logger.Info("config updated", "keys", len(snapshot))
    })
    defer unsubscribe()
    
    // Run application
    runApp(&cfg, &mu)
}

func runApp(cfg *AppConfig, mu *sync.RWMutex) {
    for {
        mu.RLock()
        dbURL := cfg.DatabaseURL
        maxConns := cfg.MaxConns
        mu.RUnlock()
        
        // Use configuration safely
        useConfig(dbURL, maxConns)
    }
}
```

## Example: Custom Configuration Struct

```go
type MyServiceConfig struct {
    configx.BaseConfig
    
    // API Configuration
    API APIConfig
    
    // Feature Flags
    Features FeatureFlags
    
    // Custom Settings
    RateLimit   int           `env:"RATE_LIMIT" default:"100"`
    CacheTTL    time.Duration `env:"CACHE_TTL" default:"5m"`
    EnableRetry bool          `env:"ENABLE_RETRY" default:"true"`
}

type APIConfig struct {
    Endpoint string        `env:"API_ENDPOINT" default:"https://api.example.com"`
    Timeout  time.Duration `env:"API_TIMEOUT" default:"30s"`
    APIKey   string        `env:"API_KEY" default:""`
}

type FeatureFlags struct {
    EnableNewUI     bool `env:"FEATURE_NEW_UI" default:"false"`
    EnableBetaAPI   bool `env:"FEATURE_BETA_API" default:"false"`
    EnableAnalytics bool `env:"FEATURE_ANALYTICS" default:"true"`
}

func main() {
    ctx := context.Background()
    manager, err := configx.DefaultManager(ctx, logger)
    if err != nil {
        log.Fatal(err)
    }
    
    var cfg MyServiceConfig
    err = manager.Bind(&cfg)
    if err != nil {
        log.Fatal(err)
    }
    
    // Access nested configuration
    fmt.Printf("API Endpoint: %s\n", cfg.API.Endpoint)
    fmt.Printf("API Timeout: %s\n", cfg.API.Timeout)
    fmt.Printf("New UI Enabled: %t\n", cfg.Features.EnableNewUI)
}
```

## Kubernetes ConfigMap Integration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: default
data:
  DATABASE_URL: "postgres://prod-db:5432/myapp"
  MAX_CONNS: "50"
  DEBUG: "false"
  RATE_LIMIT: "1000"
  CACHE_TTL: "10m"
```

```go
func main() {
    ctx := context.Background()
    
    // Set ConfigMap name via environment variable
    os.Setenv("APP_CONFIGMAP_NAME", "app-config")
    
    // DefaultManager automatically detects and watches ConfigMap
    manager, err := configx.DefaultManager(ctx, logger)
    if err != nil {
        log.Fatal(err)
    }
    
    var cfg AppConfig
    err = manager.Bind(&cfg, configx.WithUpdateCallback(func() {
        logger.Info("configuration updated from ConfigMap")
    }))
    if err != nil {
        log.Fatal(err)
    }
    
    // Configuration will automatically reload when ConfigMap changes
    runApp(cfg)
}
```

## Supported Types

The Bind() method supports the following field types:

- `string`
- `int`, `int8`, `int16`, `int32`, `int64`
- `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- `bool`
- `float32`, `float64`
- `time.Duration` (parsed from string like "5m", "30s")
- Nested structs (recursively bound)

## Configuration Priority

When using multiple sources, later sources override earlier ones:

```
Environment Variables (lowest priority)
    ↓
File Sources
    ↓
Kubernetes ConfigMaps (highest priority)
```

Example:
```go
sources := []configx.Source{
    configx.NewEnvSource(...),      // Priority 1 (lowest)
    configx.NewFileSource(...),     // Priority 2
    configx.NewK8sConfigMapSource(...), // Priority 3 (highest)
}
```

If `DATABASE_URL` is set in all three sources, the ConfigMap value wins.

## Integration with servicex

```go
import (
    "go.eggybyte.com/egg/servicex"
    "go.eggybyte.com/egg/configx"
)

type AppConfig struct {
    configx.BaseConfig
    CustomField string `env:"CUSTOM_FIELD" default:"value"`
}

func main() {
    ctx := context.Background()
    cfg := &AppConfig{}
    
    err := servicex.Run(ctx,
        servicex.WithConfig(cfg),  // Automatically uses configx.DefaultManager
        servicex.WithRegister(register),
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

## Testing

```go
func TestConfigBinding(t *testing.T) {
    ctx := context.Background()
    logger := logx.New(logx.WithWriter(io.Discard))
    
    // Set test environment variables
    os.Setenv("DATABASE_URL", "postgres://test-db/myapp")
    os.Setenv("MAX_CONNS", "5")
    os.Setenv("DEBUG", "true")
    defer func() {
        os.Unsetenv("DATABASE_URL")
        os.Unsetenv("MAX_CONNS")
        os.Unsetenv("DEBUG")
    }()
    
    // Create manager
    manager, err := configx.DefaultManager(ctx, logger)
    require.NoError(t, err)
    
    // Bind configuration
    var cfg AppConfig
    err = manager.Bind(&cfg)
    require.NoError(t, err)
    
    // Verify values
    assert.Equal(t, "postgres://test-db/myapp", cfg.DatabaseURL)
    assert.Equal(t, 5, cfg.MaxConns)
    assert.True(t, cfg.Debug)
}
```

## Stability

**Status**: Stable  
**Layer**: L2 (Capability)  
**API Guarantees**: Backward-compatible changes only

The configx module is production-ready and follows semantic versioning.

## License

This package is part of the egg framework and is licensed under the MIT License.
See the root LICENSE file for details.
