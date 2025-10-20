# ConfigX Module

<div align="center">

**Unified configuration management for Egg services**

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg)](LICENSE)

</div>

## 📦 Overview

The `configx` module provides unified configuration management for Egg services. It supports environment variables, configuration files, and Kubernetes ConfigMap hot reload with debouncing.

## ✨ Features

- 🔧 **Unified Configuration** - Single interface for all configuration sources
- 🌍 **Environment Variables** - Automatic environment variable binding
- 📁 **File Configuration** - YAML/JSON configuration file support
- ☸️ **Kubernetes ConfigMap** - Hot reload with debouncing
- 🔄 **Hot Reload** - Configuration changes without service restart
- 📝 **Structured Logging** - Context-aware logging
- 🛡️ **Validation** - Configuration validation and defaults
- 🎯 **BaseConfig** - Base configuration class for all services

## 🏗️ Architecture

```
configx/
├── configx.go      # Main configuration interface
├── builders.go     # Configuration builders
├── sources.go      # Configuration sources
└── configx_test.go # Tests
```

## 🚀 Quick Start

### Installation

```bash
go get github.com/eggybyte-technology/egg/configx@latest
```

### Basic Usage

```go
package main

import (
    "context"
    "time"

    "github.com/eggybyte-technology/egg/configx"
    "github.com/eggybyte-technology/egg/core/log"
)

// AppConfig inherits from BaseConfig
type AppConfig struct {
    configx.BaseConfig
    
    // Your business configuration
    FeatureEnabled bool          `env:"FEATURE_ENABLED" default:"false"`
    MaxRetries     int           `env:"MAX_RETRIES" default:"3"`
    Timeout        time.Duration `env:"TIMEOUT" default:"30s"`
    DatabaseURL    string        `env:"DATABASE_URL" required:"true"`
}

func main() {
    // Create logger
    logger := &YourLogger{} // Implement log.Logger interface

    // Create configuration manager
    ctx := context.Background()
    mgr, err := configx.DefaultManager(ctx, logger)
    if err != nil {
        log.Fatal("Failed to create config manager:", err)
    }

    // Bind configuration
    var cfg AppConfig
    if err := mgr.Bind(&cfg); err != nil {
        log.Fatal("Failed to bind configuration:", err)
    }

    // Use configuration
    log.Info("Configuration loaded",
        "service", cfg.ServiceName,
        "version", cfg.ServiceVersion,
        "feature_enabled", cfg.FeatureEnabled,
        "max_retries", cfg.MaxRetries,
    )
}
```

## 📖 API Reference

### Base Configuration

```go
type BaseConfig struct {
    ServiceName    string `env:"SERVICE_NAME" default:"my-service"`
    ServiceVersion string `env:"SERVICE_VERSION" default:"1.0.0"`
    Environment    string `env:"ENV" default:"development"`
    HTTPPort       string `env:"HTTP_PORT" default:":8080"`
    HealthPort     string `env:"HEALTH_PORT" default:":8081"`
    MetricsPort    string `env:"METRICS_PORT" default:":9091"`
    OTLPEndpoint   string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
    ConfigMapName  string `env:"APP_CONFIGMAP_NAME"`
    Namespace      string `env:"NAMESPACE" default:"default"`
}
```

### Configuration Manager

```go
type Manager interface {
    Bind(config interface{}) error
    Watch(ctx context.Context, callback func()) error
    Close() error
}

// DefaultManager creates a default configuration manager
func DefaultManager(ctx context.Context, logger log.Logger) (Manager, error)

// NewManager creates a new configuration manager
func NewManager(ctx context.Context, logger log.Logger, opts ...Option) (Manager, error)
```

### Configuration Sources

```go
type Source interface {
    Load() (map[string]interface{}, error)
    Watch(ctx context.Context, callback func()) error
}

// EnvironmentSource loads configuration from environment variables
func EnvironmentSource() Source

// FileSource loads configuration from a file
func FileSource(path string) Source

// ConfigMapSource loads configuration from Kubernetes ConfigMap
func ConfigMapSource(name, namespace string) Source
```

## 🔧 Configuration Sources

### Environment Variables

```bash
# Service identification
export SERVICE_NAME="my-service"
export SERVICE_VERSION="1.0.0"
export ENV="production"

# Port configuration
export HTTP_PORT=":8080"
export HEALTH_PORT=":8081"
export METRICS_PORT=":9091"

# Business configuration
export FEATURE_ENABLED="true"
export MAX_RETRIES="5"
export TIMEOUT="30s"
export DATABASE_URL="postgres://user:pass@localhost/db"

# OpenTelemetry
export OTEL_EXPORTER_OTLP_ENDPOINT="http://otel-collector:4317"

# Kubernetes configuration
export APP_CONFIGMAP_NAME="my-service-config"
export NAMESPACE="default"
```

### Configuration Files

```yaml
# config.yaml
service_name: "my-service"
service_version: "1.0.0"
environment: "production"

http_port: ":8080"
health_port: ":8081"
metrics_port: ":9091"

feature_enabled: true
max_retries: 5
timeout: "30s"
database_url: "postgres://user:pass@localhost/db"

otel_exporter_otlp_endpoint: "http://otel-collector:4317"
```

### Kubernetes ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-service-config
  namespace: default
data:
  FEATURE_ENABLED: "true"
  MAX_RETRIES: "5"
  TIMEOUT: "30s"
  DATABASE_URL: "postgres://user:pass@localhost/db"
```

## 🛠️ Advanced Usage

### Custom Configuration Manager

```go
// Create custom configuration manager
mgr, err := configx.NewManager(ctx, logger,
    configx.WithSources(
        configx.EnvironmentSource(),
        configx.FileSource("config.yaml"),
        configx.ConfigMapSource("my-service-config", "default"),
    ),
    configx.WithDebounce(5*time.Second),
    configx.WithValidation(true),
)
```

### Configuration Validation

```go
type AppConfig struct {
    configx.BaseConfig
    
    // Required fields
    DatabaseURL string `env:"DATABASE_URL" required:"true"`
    
    // Validation tags
    MaxRetries int `env:"MAX_RETRIES" default:"3" validate:"min=1,max=10"`
    Timeout    time.Duration `env:"TIMEOUT" default:"30s" validate:"min=1s,max=300s"`
    
    // Custom validation
    FeatureEnabled bool `env:"FEATURE_ENABLED" default:"false"`
}

// Custom validation method
func (c *AppConfig) Validate() error {
    if c.FeatureEnabled && c.DatabaseURL == "" {
        return errors.New("database URL is required when feature is enabled")
    }
    return nil
}
```

### Hot Reload

```go
func main() {
    // Create configuration manager
    mgr, err := configx.DefaultManager(ctx, logger)
    if err != nil {
        log.Fatal("Failed to create config manager:", err)
    }

    // Bind configuration
    var cfg AppConfig
    if err := mgr.Bind(&cfg); err != nil {
        log.Fatal("Failed to bind configuration:", err)
    }

    // Watch for configuration changes
    go func() {
        if err := mgr.Watch(ctx, func() {
            log.Info("Configuration reloaded")
            // Handle configuration changes
            handleConfigChange(&cfg)
        }); err != nil {
            log.Error("Failed to watch configuration:", err)
        }
    }()

    // Your application logic
    runApplication(&cfg)
}
```

### Multiple Configuration Sources

```go
// Load configuration from multiple sources
sources := []configx.Source{
    configx.EnvironmentSource(),
    configx.FileSource("config.yaml"),
    configx.FileSource("config.local.yaml"), // Override file
    configx.ConfigMapSource("my-service-config", "default"),
}

mgr, err := configx.NewManager(ctx, logger,
    configx.WithSources(sources...),
    configx.WithMergeStrategy(configx.MergeStrategyOverride),
)
```

## 🔧 Configuration Options

### Manager Options

```go
type Option func(*Manager)

// WithSources sets configuration sources
func WithSources(sources ...Source) Option

// WithDebounce sets debounce duration for hot reload
func WithDebounce(duration time.Duration) Option

// WithValidation enables configuration validation
func WithValidation(enabled bool) Option

// WithMergeStrategy sets merge strategy for multiple sources
func WithMergeStrategy(strategy MergeStrategy) Option
```

### Merge Strategies

```go
type MergeStrategy int

const (
    MergeStrategyOverride MergeStrategy = iota // Later sources override earlier ones
    MergeStrategyMerge                         // Merge nested objects
    MergeStrategyAppend                        // Append arrays
)
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
| ConfigX | Good |

## 🔍 Troubleshooting

### Common Issues

1. **Configuration Not Loading**
   ```go
   // Check if sources are properly configured
   sources := []configx.Source{
       configx.EnvironmentSource(),
       configx.FileSource("config.yaml"),
   }
   ```

2. **Hot Reload Not Working**
   ```go
   // Ensure watch is called
   if err := mgr.Watch(ctx, func() {
       log.Info("Configuration changed")
   }); err != nil {
       log.Error("Watch failed:", err)
   }
   ```

3. **Validation Errors**
   ```go
   // Check validation tags
   type Config struct {
       MaxRetries int `env:"MAX_RETRIES" validate:"min=1,max=10"`
   }
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
