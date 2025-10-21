# ⚙️ ConfigX Package

The `configx` package provides unified configuration management with hot updates for the EggyByte framework.

## Overview

This package offers a comprehensive configuration system that supports environment variables, configuration files, Kubernetes ConfigMaps, and hot updates. It's designed to be production-ready with validation and type safety.

## Features

- **Unified configuration** - Single interface for all configuration sources
- **Hot updates** - Configuration changes without service restart
- **Type safety** - Strongly typed configuration structures
- **Validation** - Built-in configuration validation
- **Multiple sources** - Environment variables, files, ConfigMaps
- **Kubernetes native** - ConfigMap monitoring and updates

## Quick Start

```go
import "github.com/eggybyte-technology/egg/configx"

func main() {
    // Create configuration manager
    manager, err := configx.DefaultManager(ctx, logger)
    if err != nil {
        log.Fatal(err)
    }
    
    // Define configuration structure
    var cfg AppConfig
    if err := manager.Bind(&cfg); err != nil {
        log.Fatal(err)
    }
    
    // Use configuration
    fmt.Printf("Service: %s:%s\n", cfg.ServiceName, cfg.ServiceVersion)
}
```

## API Reference

### Types

#### Manager

```go
type Manager interface {
    // Bind binds configuration to the given struct
    Bind(target interface{}) error
    
    // Watch watches for configuration changes
    Watch(ctx context.Context, target interface{}, callback func()) error
    
    // Get gets a configuration value by key
    Get(key string) (interface{}, error)
    
    // Set sets a configuration value by key
    Set(key string, value interface{}) error
}
```

#### BaseConfig

```go
type BaseConfig struct {
    ServiceName    string `env:"SERVICE_NAME" default:"app"`
    ServiceVersion string `env:"SERVICE_VERSION" default:"0.0.0"`
    Env            string `env:"ENV" default:"dev"`
    HTTPPort       string `env:"HTTP_PORT" default:":8080"`
    HealthPort     string `env:"HEALTH_PORT" default:":8081"`
    MetricsPort    string `env:"METRICS_PORT" default:":9091"`
}
```

#### Options

```go
type Options struct {
    ConfigFile     string            // Configuration file path
    ConfigMapName  string            // Kubernetes ConfigMap name
    Namespace      string            // Kubernetes namespace
    WatchInterval  time.Duration     // Watch interval for hot updates
    ValidationFunc func(interface{}) error // Custom validation function
}
```

### Functions

```go
// DefaultManager creates a default configuration manager
func DefaultManager(ctx context.Context, logger log.Logger) (Manager, error)

// NewManager creates a new configuration manager with options
func NewManager(ctx context.Context, logger log.Logger, opts Options) (Manager, error)

// NewBaseConfig creates a new base configuration
func NewBaseConfig() *BaseConfig
```

## Usage Examples

### Basic Configuration

```go
type AppConfig struct {
    configx.BaseConfig
    
    // Database configuration
    Database DatabaseConfig
}

type DatabaseConfig struct {
    Driver string `env:"DB_DRIVER" default:"mysql"`
    DSN    string `env:"DB_DSN" default:"user:password@tcp(localhost:3306)/db"`
    MaxIdle int   `env:"DB_MAX_IDLE" default:"10"`
    MaxOpen int   `env:"DB_MAX_OPEN" default:"100"`
}

func main() {
    // Create configuration manager
    manager, err := configx.DefaultManager(ctx, logger)
    if err != nil {
        log.Fatal(err)
    }
    
    // Load configuration
    var cfg AppConfig
    if err := manager.Bind(&cfg); err != nil {
        log.Fatal(err)
    }
    
    // Use configuration
    fmt.Printf("Service: %s:%s\n", cfg.ServiceName, cfg.ServiceVersion)
    fmt.Printf("Database: %s\n", cfg.Database.Driver)
}
```

### With Configuration File

```go
func main() {
    // Create configuration manager with file
    manager, err := configx.NewManager(ctx, logger, configx.Options{
        ConfigFile:    "config.yaml",
        WatchInterval: 5 * time.Second,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Load configuration
    var cfg AppConfig
    if err := manager.Bind(&cfg); err != nil {
        log.Fatal(err)
    }
    
    // Watch for changes
    if err := manager.Watch(ctx, &cfg, func() {
        logger.Info("Configuration updated")
    }); err != nil {
        log.Fatal(err)
    }
    
    // Use configuration
    startService(cfg)
}
```

### With Kubernetes ConfigMap

```go
func main() {
    // Create configuration manager with ConfigMap
    manager, err := configx.NewManager(ctx, logger, configx.Options{
        ConfigMapName: "app-config",
        Namespace:     "default",
        WatchInterval: 10 * time.Second,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Load configuration
    var cfg AppConfig
    if err := manager.Bind(&cfg); err != nil {
        log.Fatal(err)
    }
    
    // Watch for changes
    if err := manager.Watch(ctx, &cfg, func() {
        logger.Info("Configuration updated from ConfigMap")
        // Reload configuration-dependent components
        reloadComponents(cfg)
    }); err != nil {
        log.Fatal(err)
    }
    
    // Use configuration
    startService(cfg)
}
```

### Custom Validation

```go
func validateConfig(cfg interface{}) error {
    appCfg, ok := cfg.(*AppConfig)
    if !ok {
        return errors.New("CONFIG_ERROR", "invalid configuration type")
    }
    
    // Validate service name
    if utils.IsEmpty(appCfg.ServiceName) {
        return errors.New("CONFIG_ERROR", "service name is required")
    }
    
    // Validate database configuration
    if utils.IsEmpty(appCfg.Database.DSN) {
        return errors.New("CONFIG_ERROR", "database DSN is required")
    }
    
    // Validate ports
    if !utils.IsValidPort(extractPort(appCfg.HTTPPort)) {
        return errors.New("CONFIG_ERROR", "invalid HTTP port")
    }
    
    return nil
}

func main() {
    // Create configuration manager with validation
    manager, err := configx.NewManager(ctx, logger, configx.Options{
        ConfigFile:     "config.yaml",
        WatchInterval:  5 * time.Second,
        ValidationFunc: validateConfig,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Load configuration
    var cfg AppConfig
    if err := manager.Bind(&cfg); err != nil {
        log.Fatal(err)
    }
    
    // Use configuration
    startService(cfg)
}
```

### Hot Updates

```go
func main() {
    // Create configuration manager
    manager, err := configx.DefaultManager(ctx, logger)
    if err != nil {
        log.Fatal(err)
    }
    
    // Load initial configuration
    var cfg AppConfig
    if err := manager.Bind(&cfg); err != nil {
        log.Fatal(err)
    }
    
    // Watch for changes
    if err := manager.Watch(ctx, &cfg, func() {
        logger.Info("Configuration updated",
            log.Str("service_name", cfg.ServiceName),
            log.Str("service_version", cfg.ServiceVersion),
        )
        
        // Update configuration-dependent components
        updateComponents(cfg)
    }); err != nil {
        log.Fatal(err)
    }
    
    // Start service
    startService(cfg)
}

func updateComponents(cfg AppConfig) {
    // Update database connection
    if err := updateDatabase(cfg.Database); err != nil {
        logger.Error(err, "Failed to update database configuration")
    }
    
    // Update HTTP server
    if err := updateHTTPServer(cfg.HTTPPort); err != nil {
        logger.Error(err, "Failed to update HTTP server configuration")
    }
    
    // Update metrics configuration
    if err := updateMetrics(cfg.MetricsPort); err != nil {
        logger.Error(err, "Failed to update metrics configuration")
    }
}
```

## Configuration Sources

### Environment Variables

```bash
# Service configuration
SERVICE_NAME=user-service
SERVICE_VERSION=1.0.0
ENV=production

# HTTP configuration
HTTP_PORT=:8080
HEALTH_PORT=:8081
METRICS_PORT=:9091

# Database configuration
DB_DRIVER=mysql
DB_DSN=user:password@tcp(localhost:3306)/db
DB_MAX_IDLE=10
DB_MAX_OPEN=100
```

### Configuration File (YAML)

```yaml
service_name: "user-service"
service_version: "1.0.0"
env: "production"

http_port: ":8080"
health_port: ":8081"
metrics_port: ":9091"

database:
  driver: "mysql"
  dsn: "user:password@tcp(localhost:3306)/db"
  max_idle: 10
  max_open: 100

features:
  enable_debug_logs: false
  enable_metrics: true
  enable_tracing: true
```

### Kubernetes ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: default
data:
  config.yaml: |
    service_name: "user-service"
    service_version: "1.0.0"
    env: "production"
    
    http_port: ":8080"
    health_port: ":8081"
    metrics_port: ":9091"
    
    database:
      driver: "mysql"
      dsn: "user:password@tcp(mysql:3306)/db"
      max_idle: 10
      max_open: 100
```

## Service Integration

### With RuntimeX

```go
func main() {
    // Create configuration manager
    manager, err := configx.DefaultManager(ctx, logger)
    if err != nil {
        log.Fatal(err)
    }
    
    // Load configuration
    var cfg AppConfig
    if err := manager.Bind(&cfg); err != nil {
        log.Fatal(err)
    }
    
    // Create HTTP mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", handleRoot)
    
    // Start runtime with configuration
    err = runtimex.Run(ctx, cancel, runtimex.Options{
        Logger: logger,
        HTTP: &runtimex.HTTPOptions{
            Addr: cfg.HTTPPort,
            H2C:  true,
            Mux:  mux,
        },
        Health:  &runtimex.Endpoint{Addr: cfg.HealthPort},
        Metrics: &runtimex.Endpoint{Addr: cfg.MetricsPort},
        ShutdownTimeout: 15 * time.Second,
    })
    
    if err != nil {
        logger.Error(err, "Runtime failed")
        os.Exit(1)
    }
}
```

### With ConnectX

```go
func main() {
    // Create configuration manager
    manager, err := configx.DefaultManager(ctx, logger)
    if err != nil {
        log.Fatal(err)
    }
    
    // Load configuration
    var cfg AppConfig
    if err := manager.Bind(&cfg); err != nil {
        log.Fatal(err)
    }
    
    // Setup Connect interceptors with configuration
    interceptors := connectx.DefaultInterceptors(connectx.Options{
        Logger:            logger,
        SlowRequestMillis: cfg.Connect.SlowRequestMillis,
        PayloadAccounting: cfg.Connect.PayloadAccounting,
        WithRequestBody:   cfg.Connect.WithRequestBody,
        WithResponseBody:  cfg.Connect.WithResponseBody,
    })
    
    // Create Connect handler
    path, handler := userv1connect.NewUserServiceHandler(
        service,
        connect.WithInterceptors(interceptors...),
    )
    
    // Register handler
    mux.Handle(path, handler)
}
```

## Testing

```go
func TestConfiguration(t *testing.T) {
    // Create test configuration manager
    manager, err := configx.NewManager(ctx, &TestLogger{}, configx.Options{
        ConfigFile: "test-config.yaml",
    })
    assert.NoError(t, err)
    
    // Load configuration
    var cfg AppConfig
    err = manager.Bind(&cfg)
    assert.NoError(t, err)
    
    // Verify configuration
    assert.Equal(t, "test-service", cfg.ServiceName)
    assert.Equal(t, "1.0.0", cfg.ServiceVersion)
    assert.Equal(t, ":8080", cfg.HTTPPort)
}

func TestHotUpdates(t *testing.T) {
    // Create test configuration manager
    manager, err := configx.NewManager(ctx, &TestLogger{}, configx.Options{
        ConfigFile:    "test-config.yaml",
        WatchInterval: 100 * time.Millisecond,
    })
    assert.NoError(t, err)
    
    // Load configuration
    var cfg AppConfig
    err = manager.Bind(&cfg)
    assert.NoError(t, err)
    
    // Watch for changes
    updateCount := 0
    err = manager.Watch(ctx, &cfg, func() {
        updateCount++
    })
    assert.NoError(t, err)
    
    // Simulate configuration change
    time.Sleep(200 * time.Millisecond)
    
    // Verify update was detected
    assert.Greater(t, updateCount, 0)
}

type TestLogger struct{}

func (l *TestLogger) With(kv ...any) log.Logger { return l }
func (l *TestLogger) Debug(msg string, kv ...any) {}
func (l *TestLogger) Info(msg string, kv ...any) {}
func (l *TestLogger) Warn(msg string, kv ...any) {}
func (l *TestLogger) Error(err error, msg string, kv ...any) {}
```

## Best Practices

### 1. Use Structured Configuration

```go
type AppConfig struct {
    configx.BaseConfig
    
    // Group related configuration
    Database DatabaseConfig
    Features FeatureConfig
    Business BusinessConfig
}

type DatabaseConfig struct {
    Driver  string `env:"DB_DRIVER" default:"mysql"`
    DSN     string `env:"DB_DSN" default:"user:password@tcp(localhost:3306)/db"`
    MaxIdle int    `env:"DB_MAX_IDLE" default:"10"`
    MaxOpen int    `env:"DB_MAX_OPEN" default:"100"`
}
```

### 2. Validate Configuration

```go
func validateConfig(cfg interface{}) error {
    appCfg, ok := cfg.(*AppConfig)
    if !ok {
        return errors.New("CONFIG_ERROR", "invalid configuration type")
    }
    
    // Validate required fields
    if utils.IsEmpty(appCfg.ServiceName) {
        return errors.New("CONFIG_ERROR", "service name is required")
    }
    
    // Validate formats
    if !utils.IsValidPort(extractPort(appCfg.HTTPPort)) {
        return errors.New("CONFIG_ERROR", "invalid HTTP port")
    }
    
    return nil
}
```

### 3. Use Hot Updates

```go
func main() {
    // Create configuration manager
    manager, err := configx.DefaultManager(ctx, logger)
    if err != nil {
        log.Fatal(err)
    }
    
    // Load configuration
    var cfg AppConfig
    if err := manager.Bind(&cfg); err != nil {
        log.Fatal(err)
    }
    
    // Watch for changes
    if err := manager.Watch(ctx, &cfg, func() {
        logger.Info("Configuration updated")
        updateComponents(cfg)
    }); err != nil {
        log.Fatal(err)
    }
    
    // Start service
    startService(cfg)
}
```

### 4. Handle Configuration Errors

```go
func main() {
    // Create configuration manager
    manager, err := configx.DefaultManager(ctx, logger)
    if err != nil {
        logger.Error(err, "Failed to create configuration manager")
        os.Exit(1)
    }
    
    // Load configuration
    var cfg AppConfig
    if err := manager.Bind(&cfg); err != nil {
        logger.Error(err, "Failed to bind configuration")
        os.Exit(1)
    }
    
    // Validate configuration
    if err := validateConfig(&cfg); err != nil {
        logger.Error(err, "Configuration validation failed")
        os.Exit(1)
    }
    
    // Start service
    startService(cfg)
}
```

## Thread Safety

All functions in this package are safe for concurrent use. The configuration manager is designed to handle concurrent access safely.

## Dependencies

- **Go 1.21+** required
- **Kubernetes client-go** - ConfigMap support (optional)
- **Standard library** - Core functionality

## Version Compatibility

- **Go 1.21+** required
- **API Stability**: Evolving (L3 module)
- **Breaking Changes**: Possible in minor versions

## Contributing

Contributions are welcome! Please see the main project [Contributing Guide](../CONTRIBUTING.md) for details.

## License

This package is part of the EggyByte framework and is licensed under the MIT License.