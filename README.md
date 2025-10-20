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

#### For Framework Development

```bash
# Clone the repository
git clone https://github.com/eggybyte-technology/egg.git
cd egg

# Sync workspace
go work sync

# Install development tools
make tools

# Build CLI tool
make build-cli

# Run tests
make test
```

#### For Application Development

```bash
# Install the egg CLI tool
go install github.com/eggybyte-technology/egg/cli/cmd/egg@latest

# Or download pre-built binaries from releases
# https://github.com/eggybyte-technology/egg/releases
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

## 🛠️ Egg CLI Tool

The Egg CLI provides a complete development workflow for building microservices.

### Quick Start

```bash
# Install the CLI tool
go install github.com/eggybyte-technology/egg/cli/cmd/egg@latest

# Verify installation
egg doctor

# Initialize a new project
egg init --project-name my-platform \
         --module-prefix github.com/myorg/my-platform \
         --docker-registry ghcr.io/myorg

# Create a backend service
egg create backend user-service

# Create a frontend service
egg create frontend admin_portal --platforms web

# Generate API code
egg api init
egg api generate

# Start local development
egg compose up
```

### Naming Convention for Frontend Services

When creating Flutter frontend services, use **underscores** instead of hyphens to comply with Dart package naming requirements:

```bash
# ✅ Recommended - Use underscores
egg create frontend admin_portal --platforms web
egg create frontend user_dashboard --platforms web

# ⚠️ Acceptable - Will be auto-converted
egg create frontend admin-portal --platforms web
# Automatically converts to: admin_portal
```

Dart requires package names to use only lowercase letters, numbers, and underscores.

For detailed CLI documentation, examples, and all available commands, see **[CLI Documentation](cli/README.md)**.

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

### Framework Development

```bash
# Format code
make fmt

# Run tests
make test

# Run CLI integration tests
make test-cli

# Run linter
make lint

# Build all modules
make build

# Build CLI tool
make build-cli

# Run example
make run-example

# Quality check (fmt + vet + test + lint)
make quality
```

### CLI Tool Testing

```bash
# Run comprehensive CLI integration tests
make test-cli

# Run tests and keep the test project for inspection
make test-cli-keep
```

The CLI integration test validates:
- ✅ Project initialization with custom configuration
- ✅ Backend service generation with local module dependencies
- ✅ Go workspace management (go.work)
- ✅ Frontend service generation (Flutter)
- ✅ Service registration in egg.yaml
- ✅ API configuration and code generation
- ✅ Docker Compose configuration
- ✅ Configuration validation

## 📁 Project Structure

### Framework Repository

```
egg/
├── cli/            # CLI tool
│   ├── cmd/egg/    # Command implementations
│   ├── internal/   # CLI internals
│   │   ├── configschema/  # Configuration schema
│   │   ├── generators/    # Code generators
│   │   ├── templates/     # Service templates
│   │   ├── toolrunner/    # External tool execution
│   │   └── render/        # Manifest renderers
│   └── egg         # Built CLI binary
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
│   ├── guide.md    # Detailed guide
│   ├── egg-cli.md  # CLI documentation
│   └── RELEASING.md # Release guide
├── scripts/        # Automation scripts
│   └── test-cli.sh # CLI integration tests
├── go.work         # Workspace
├── Makefile        # Build scripts
└── .goreleaser.yml # Release configuration
```

### Generated Application Structure

After running `egg init` and creating services:

```
my-platform/
├── api/            # Protobuf API definitions
│   ├── buf.yaml
│   ├── buf.gen.yaml
│   └── myservice/v1/
│       └── service.proto
├── backend/        # Backend services
│   ├── go.work     # Go workspace for all backend services
│   └── user-service/
│       ├── cmd/server/
│       │   └── main.go
│       ├── internal/
│       │   ├── config/
│       │   ├── handler/
│       │   └── service/
│       ├── go.mod
│       └── go.sum
├── frontend/       # Frontend applications
│   └── admin-portal/
│       ├── lib/
│       ├── web/
│       └── pubspec.yaml
├── gen/            # Generated code
│   ├── go/         # Go Connect code
│   ├── dart/       # Dart API clients
│   ├── ts/         # TypeScript types
│   └── openapi/    # OpenAPI specs
├── build/          # Docker build files
│   ├── Dockerfile.backend
│   ├── Dockerfile.frontend
│   └── Dockerfile.eggybyte-go-alpine
├── deploy/         # Deployment manifests
│   └── compose.yaml
└── egg.yaml        # Project configuration
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

Egg uses unified version tags for all modules:

- `v0.0.1` - All modules released together
- `v0.1.0` - Minor version updates
- `v1.0.0` - Major stable release

Usage example:

```bash
# Install CLI tool
go install github.com/eggybyte-technology/egg/cli/cmd@v0.0.1

# Use framework modules
go get github.com/eggybyte-technology/egg/core@v0.0.1
go get github.com/eggybyte-technology/egg/connectx@v0.0.1
```

### Release Process

We use [GoReleaser](https://goreleaser.com/) for automated releases:

```bash
# Test release locally
make release-test

# Create and push tag
git tag -a v0.0.1 -m "Release v0.0.1"
git push origin v0.0.1

# Publish release (requires GITHUB_TOKEN)
export GITHUB_TOKEN=your_token
make release-publish
```

See [RELEASE_QUICKSTART.md](RELEASE_QUICKSTART.md) for quick reference or [docs/RELEASING.md](docs/RELEASING.md) for detailed guide.

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

- [Detailed Guide](docs/guide.md) - Complete framework guide
- [CLI Documentation](cli/README.md) - CLI tool complete reference
- [Dart Naming Guide](docs/DART_NAMING_COMPATIBILITY.md) - Flutter/Dart naming compatibility
- [Example Service](examples/minimal-connect-service) - Minimal Connect service
- [API Documentation](https://pkg.go.dev/github.com/eggybyte-technology/egg) - Go package docs
- [Release Guide](docs/RELEASING.md) - How to release new versions

## 🎯 Use Cases

### Microservices Platform
Build a complete microservices platform with:
- Multiple backend services with Connect
- Web and mobile frontends with Flutter
- Unified observability and configuration
- Kubernetes-native deployment

### API-First Development
- Define APIs with Protobuf
- Generate type-safe clients for multiple languages
- Automatic OpenAPI documentation
- Version control for API evolution

### Cloud Native Applications
- Built-in Kubernetes integration
- ConfigMap hot reload
- Service discovery
- Health checks and metrics

### Monorepo Management
- Multiple services in one repository
- Shared code and dependencies
- Unified build and deployment
- Independent service versioning

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

Thanks to all contributors and the Go community for their support.

---

<div align="center">

**Built with ❤️ by EggyByte Technology**

</div>