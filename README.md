# Egg Framework

<div align="center">

**A modern, modular Go microservices framework designed for Cloud Native environments**

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/eggybyte-technology/egg)](https://goreportcard.com/report/github.com/eggybyte-technology/egg)
[![Coverage](https://img.shields.io/badge/Coverage-85%25-brightgreen.svg)](https://github.com/eggybyte-technology/egg)

</div>

## ✨ Features

- 🚀 **Connect-First Architecture** - Unified interceptor stack with zero business intrusion
- 🔧 **Unified Configuration Management** - Environment variables, files, and K8s ConfigMap hot reload
- 📊 **Complete Observability** - OpenTelemetry integration with unified logging, tracing, and metrics
- 🔐 **Identity Injection & Propagation** - Automatic user identity extraction from request headers
- 🎯 **Single Port Strategy** - HTTP/Connect/gRPC-Web unified port with separate health/metrics ports
- ☸️ **Kubernetes Native** - ConfigMap name-based watching, service discovery, and Secret contracts
- 🗄️ **Database Adapters** - GORM integration supporting MySQL, PostgreSQL, and SQLite
- 📦 **Monorepo Architecture** - Independent modules with clear dependencies and subdirectory tag releases

## 🏗️ Architecture

Egg follows a layered architecture with clear module responsibilities:

```
┌─────────────────────────────────────────────────┐
│              Application Layer                  │
│         (Your Business Logic)                   │
└─────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────┐
│          Transport Layer (L3)                   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐     │
│  │ connectx │  │  configx │  │   obsx   │     │
│  └──────────┘  └──────────┘  └──────────┘     │
│  ┌──────────┐  ┌──────────┐                   │
│  │   k8sx   │  │  storex  │                   │
│  └──────────┘  └──────────┘                   │
└─────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────┐
│           Runtime Layer (L2)                    │
│              ┌──────────┐                       │
│              │ runtimex │                       │
│              └──────────┘                       │
└─────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────┐
│            Core Layer (L1)                      │
│  ┌─────┐  ┌────────┐  ┌──────────┐  ┌───────┐│
│  │ log │  │ errors │  │ identity │  │ utils ││
│  └─────┘  └────────┘  └──────────┘  └───────┘│
└─────────────────────────────────────────────────┘
```

## 📦 Modules

### Core Layer (L1) - Zero Dependencies

#### `core`
- **`log`** - Structured logging interface compatible with slog philosophy
- **`errors`** - Layered error handling with error codes and wrapping
- **`identity`** - User identity and request metadata container
- **`utils`** - Common utilities for retry, time, slices, etc.

### Runtime Layer (L2) - Runtime Management

#### `runtimex`
Lifecycle orchestration, unified port strategy, and health/metrics endpoint management.

### Transport & Infrastructure Layer (L3) - Transport & Infrastructure

#### `connectx`
- Connect protocol binding
- Unified interceptors: recovery, logging, tracing, metrics, identity injection, error mapping
- Identity extraction from Higress request headers

#### `configx`
- Unified configuration management: environment variables, files, K8s ConfigMap
- Hot reload support with debouncing
- BaseConfig base class inherited by all services

#### `obsx`
- OpenTelemetry Tracing and Metrics initialization
- OTLP exporter support
- Runtime metrics collection

#### `k8sx`
- ConfigMap name-based watching (supports multiple ConfigMaps)
- Service discovery (Headless / ClusterIP)
- Secret contracts (injection via env + secretKeyRef)

#### `storex`
- Storage interface definitions
- GORM adapters: MySQL, PostgreSQL, SQLite
- Connection registration and health probes

## 🚀 Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/eggybyte-technology/egg.git
cd egg

# Sync workspace
go work sync

# Install development tools
make tools

# Run tests
make test
```

### Create Your First Service

```go
package main

import (
    "context"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/eggybyte-technology/egg/configx"
    "github.com/eggybyte-technology/egg/connectx"
    "github.com/eggybyte-technology/egg/core/log"
    "github.com/eggybyte-technology/egg/obsx"
    "github.com/eggybyte-technology/egg/runtimex"
)

// AppConfig inherits from BaseConfig
type AppConfig struct {
    configx.BaseConfig
    // Your business configuration
}

func main() {
    logger := &YourLogger{} // Implement log.Logger interface
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // 1. Configuration management
    mgr, _ := configx.DefaultManager(ctx, logger)
    var cfg AppConfig
    _ = mgr.Bind(&cfg)

    // 2. Observability
    otel, _ := obsx.NewProvider(ctx, obsx.Options{
        ServiceName:    cfg.ServiceName,
        ServiceVersion: cfg.ServiceVersion,
        OTLPEndpoint:   cfg.OTLPEndpoint,
    })
    defer otel.Shutdown(ctx)

    // 3. Connect routing + interceptors
    mux := http.NewServeMux()
    ints := connectx.DefaultInterceptors(connectx.Options{
        Logger: logger,
        Otel:   otel,
    })
    // Register your Connect handlers
    // ...

    // 4. Runtime
    _ = runtimex.Run(ctx, nil, runtimex.Options{
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
}
```

See the complete example at [examples/minimal-connect-service](examples/minimal-connect-service).

## ⚙️ Configuration Management

### Base Configuration (BaseConfig)

All services should inherit from `configx.BaseConfig`:

```go
type AppConfig struct {
    configx.BaseConfig
    
    // Your business configuration
    FeatureEnabled bool  `env:"FEATURE_ENABLED" default:"false"`
    MaxRetries     int   `env:"MAX_RETRIES" default:"3"`
}
```

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

# Observability
export OTEL_EXPORTER_OTLP_ENDPOINT="http://otel-collector:4317"

# Dynamic configuration (K8s mode)
export APP_CONFIGMAP_NAME="my-service-config"
export NAMESPACE="default"
```

### ConfigMap Hot Reload

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-service-config
data:
  FEATURE_ENABLED: "true"
  MAX_RETRIES: "5"
```

Configuration changes are automatically reloaded with debouncing support.

## 📊 Observability

### Logging

Unified log fields:

- `ts`, `level`, `service`, `version`, `env`, `instance`
- `trace_id`, `span_id`, `req_id`
- `rpc_system`, `rpc_service`, `rpc_method`
- `status`, `latency_ms`, `remote_ip`, `user_agent`

### Tracing

Uses OpenTelemetry with automatic tracing for all Connect requests.

### Metrics

Recommended metric naming:

- `rpc.server.duration` (Histogram, ms)
- `rpc.server.requests` (Counter, labels: code, service, method)
- `rpc.server.payload_bytes` (UpDownCounter, labels: direction=in|out)

## 🛠️ Development Tools

```bash
# Format code
make fmt

# Run tests
make test

# Run linter
make lint

# Build all modules
make build

# Run example
make run-example

# Quality check (fmt + vet + test + lint)
make quality
```

## 📁 Project Structure

```
egg/
├── core/           # L1: Zero-dependency core interfaces
│   ├── log/        # Logging interface
│   ├── errors/     # Error handling
│   ├── identity/   # Identity container
│   └── utils/      # Common utilities
├── runtimex/       # L2: Runtime management
├── connectx/       # L3: Connect binding
├── configx/        # L3: Configuration management
├── obsx/           # L3: Observability
├── k8sx/           # L3: Kubernetes integration
├── storex/         # L3: Storage adapters
├── examples/       # Example services
│   └── minimal-connect-service/
├── docs/           # Documentation
│   └── guide.md    # Detailed guide
├── go.work         # Workspace
├── Makefile        # Build scripts
└── .golangci.yml   # Linter configuration
```

## 📈 Test Coverage

| Module | Coverage |
|--------|----------|
| core/log | 100% |
| core/errors | 91.7% |
| core/identity | 100% |
| core/utils | 94.3% |
| runtimex | 58.1% |
| connectx | 92.9% |
| configx | Good |
| obsx | Good |
| k8sx | Good |
| storex | Good |

## 🏷️ Versioning & Releases

Egg uses Monorepo subdirectory tag strategy:

- `core/v1.0.0`
- `runtimex/v1.0.0`
- `connectx/v1.2.0`
- `obsx/v1.1.0`

Usage example:

```bash
go get github.com/eggybyte-technology/egg/core@core/v1.0.0
go get github.com/eggybyte-technology/egg/connectx@connectx/v1.2.0
```

For major version v2+, module paths need `/v2` suffix.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Create a Pull Request

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## ✅ Quality Standards

- **Code Style**: `gofmt` + `goimports`
- **Static Analysis**: `go vet` + `golangci-lint`
- **Testing**: Unit test coverage > 80%
- **Documentation**: All exported symbols must have GoDoc comments
- **Security**: `govulncheck` scanning

## 🚀 Production Readiness Checklist

- [ ] Implement `log.Logger` interface (recommend using `slog`)
- [ ] Configure OpenTelemetry exporters
- [ ] Set up reasonable health check logic
- [ ] Configure Prometheus metrics collection
- [ ] Configure RBAC in K8s (if using ConfigMap/service discovery)
- [ ] Set reasonable resource limits (CPU/Memory)
- [ ] Configure log levels and sensitive information filtering
- [ ] Enable TLS (production environment)
- [ ] Configure graceful shutdown timeout
- [ ] Monitor key metrics and alerts

## 📚 Resources

- [Detailed Guide](docs/guide.md)
- [Example Service](examples/minimal-connect-service)
- [API Documentation](https://pkg.go.dev/github.com/eggybyte-technology/egg)
- [Architecture Design](docs/ARCHITECTURE.md)
- [Release Notes](docs/RELEASING.md)

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

Thanks to all contributors and the Go community for their support.

---

<div align="center">

**Built with ❤️ by EggyByte Technology**

</div>