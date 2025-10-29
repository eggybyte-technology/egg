# Egg - Production-Ready Go Microservices Framework

**A modern, layered Go framework for building Connect-RPC microservices with observability, configuration management, and clean architecture.**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg?style=flat-square)](LICENSE)

---

## Overview

Egg is a comprehensive microservices framework designed for building production-grade Go services with minimal boilerplate. It provides:

- **One-line service startup** with integrated configuration, logging, database, and tracing
- **Clean layered architecture** with clear dependency boundaries
- **Connect-RPC first** with automatic interceptor stack
- **Hot configuration reloading** from Kubernetes ConfigMaps
- **Production-ready observability** with OpenTelemetry
- **Type-safe dependency injection** container

## Core Philosophy

1. **Clarity over cleverness** - Explicit, readable code
2. **Layered dependencies** - No circular imports, clear dependency flow
3. **Interface-driven design** - Public API separate from implementation
4. **Production-ready defaults** - Sensible configuration out of the box
5. **CLI-driven development** - Never manually edit `go.mod` or `go.work`

## Quick Start

### Installation

```bash
go get go.eggybyte.com/egg/servicex@latest
```

### Minimal Service

```go
package main

import (
    "context"
    "log/slog"
    "go.eggybyte.com/egg/configx"
    "go.eggybyte.com/egg/logx"
    "go.eggybyte.com/egg/servicex"
)

type AppConfig struct {
    configx.BaseConfig
}

func register(app *servicex.App) error {
    // Register your Connect handlers here
    app.Mux().HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, Egg!"))
    })
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
```

## Recent Improvements

### v0.2.1 - Modern Build System & Unified Logging

**Modernized Makefile with Unified Logging**
- All Makefile commands now use unified `logger.sh` for consistent output formatting
- Improved error handling and cleaner command execution
- New commands for modern development workflow:
  - `make tidy` - Clean and update dependencies for all modules
  - `make coverage` - Generate test coverage report with HTML output
  - `make check` - Quick validation (lint + test)
  - `make quality` - Full quality check (tidy + lint + test + coverage)

**Reorganized Docker Foundation Images**
- Renamed `docker/` to `base-images/` for clearer purpose
- Dedicated Makefile for base image management
- Comprehensive documentation for multi-arch builds

**Enhanced Linter Configuration**
- Fixed `golangci-lint` warnings for `gocritic` settings
- Cleaner lint output without metadata warnings
- Simplified `.golangci.yml` configuration

**Development Workflow:**
```bash
# First-time setup
make setup       # Install tools + init workspace

# Daily development
make tidy        # Clean dependencies
make check       # Quick validation
make coverage    # Check test coverage

# Before release
make quality     # Full quality check
make release VERSION=v0.3.0
```

### v0.2.0 - Simplified Configuration & Enhanced Logging

**Environment-Based Log Level Control**
- Use `LOG_LEVEL` environment variable instead of `WithDebugLogs()`
- Supports: `debug`, `info`, `warn`, `error`
- Automatically applied when using `servicex`

**Simplified Database Configuration**
- `WithAppConfig()` now auto-detects database configuration from `configx.BaseConfig`
- No need for separate `WithDatabase()` call
- Database settings loaded from environment variables after config binding

**Enhanced Logging Helpers**
- Added `Int32()`, `Int64()`, `Float64()`, `String()` to `core/log`
- More convenient structured logging

**Example:**
```go
// Old pattern
servicex.WithConfig(cfg),
servicex.WithDatabase(servicex.FromBaseConfig(&cfg.Database)),
servicex.WithDebugLogs(true),

// New pattern (recommended)
servicex.WithAppConfig(cfg), // Auto-detects database
// Set LOG_LEVEL=debug via environment variable
```

## Architecture

### Layered Design

Egg follows a strict layered architecture to prevent circular dependencies and ensure maintainability:

```
┌─────────────────────────────────────────────────┐
│  L4: Integration Layer                          │
│  ┌─────────────────────────────────────────┐    │
│  │  servicex: One-line service startup     │    │
│  └─────────────────────────────────────────┘    │
└────────────────┬────────────────────────────────┘
                 │
┌────────────────┴────────────────────────────────┐
│  L3: Runtime & Communication Layer              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │runtimex  │  │connectx  │  │clientx   │      │
│  │(lifecycle)│  │(RPC)     │  │(client)  │      │
│  └──────────┘  └──────────┘  └──────────┘      │
└────────────────┬────────────────────────────────┘
                 │
┌────────────────┴────────────────────────────────┐
│  L2: Capability Layer                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │configx   │  │obsx      │  │httpx     │      │
│  │(config)  │  │(tracing) │  │(HTTP)    │      │
│  └──────────┘  └──────────┘  └──────────┘      │
└────────────────┬────────────────────────────────┘
                 │
┌────────────────┴────────────────────────────────┐
│  L1: Foundation Layer                           │
│  ┌─────────────────────────────────────────┐    │
│  │  logx: Structured logging               │    │
│  └─────────────────────────────────────────┘    │
└────────────────┬────────────────────────────────┘
                 │
┌────────────────┴────────────────────────────────┐
│  L0: Core Layer (Zero Dependencies)             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │errors    │  │identity  │  │log       │      │
│  │(types)   │  │(context) │  │(interface)│     │
│  └──────────┘  └──────────┘  └──────────┘      │
└─────────────────────────────────────────────────┘

Auxiliary Modules (can depend on any layer):
┌─────────────────────────────────────────────────┐
│  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │storex    │  │k8sx      │  │testingx  │      │
│  │(storage) │  │(k8s)     │  │(testing) │      │
│  └──────────┘  └──────────┘  └──────────┘      │
└─────────────────────────────────────────────────┘
```

### Dependency Rules

- **Rule 1**: A module can only depend on modules in the same or lower layers
- **Rule 2**: No circular dependencies between modules
- **Rule 3**: Core modules (L0) have zero external dependencies
- **Rule 4**: Public interface files are thin (~100-200 lines), complex logic in `internal/`

## Module Overview

### L4: Integration Layer

#### [servicex](servicex/) - One-Line Service Startup
The highest-level module that integrates all components for microservice initialization.

```go
servicex.Run(ctx,
    servicex.WithService("my-service", "1.0.0"),
    servicex.WithLogger(logger),
    servicex.WithAppConfig(cfg), // Auto-detects database from BaseConfig
    servicex.WithAutoMigrate(&model.User{}),
    servicex.WithRegister(register),
)
```

**Key Features:**
- Integrated configuration, logging, database, tracing
- Automatic database configuration from `BaseConfig`
- Environment-based log level control (`LOG_LEVEL`)
- Connect RPC interceptor stack
- Graceful shutdown with hooks
- Dependency injection container

### L3: Runtime & Communication Layer

#### [runtimex](runtimex/) - Service Lifecycle Management
Manages service startup, shutdown, and health checks.

```go
runtimex.Run(ctx, services, runtimex.Options{
    Logger: logger,
    HTTP:   &runtimex.HTTPOptions{Addr: ":8080", Mux: mux},
    Health: &runtimex.Endpoint{Addr: ":8081"},
})
```

**Key Features:**
- Concurrent service startup/shutdown
- Health check aggregation
- Multiple server support (HTTP, RPC, Health, Metrics)
- Configurable shutdown timeout

#### [connectx](connectx/) - Connect RPC Interceptors
Provides a unified interceptor stack for Connect-RPC services.

```go
interceptors := connectx.DefaultInterceptors(connectx.Options{
    Logger:            logger,
    Otel:              provider,
    SlowRequestMillis: 1000,
})
```

**Key Features:**
- Timeout enforcement with header override
- Structured logging with correlation
- Error mapping to Connect codes
- OpenTelemetry tracing integration

#### [clientx](clientx/) - Connect Client Factory
Creates Connect HTTP clients with retry, circuit breaker, and timeouts.

```go
client := clientx.NewHTTPClient("https://api.example.com",
    clientx.WithTimeout(5*time.Second),
    clientx.WithRetry(3),
)
```

### L2: Capability Layer

#### [configx](configx/) - Configuration Management
Unified configuration with hot reloading from multiple sources.

```go
manager, _ := configx.DefaultManager(ctx, logger)
var cfg AppConfig
manager.Bind(&cfg)
```

**Key Features:**
- Multiple sources (Env, File, K8s ConfigMap)
- Hot reload with debouncing
- Struct binding with validation
- Change notifications

#### [obsx](obsx/) - OpenTelemetry Provider
Simplified OpenTelemetry initialization for tracing and metrics.

```go
provider, _ := obsx.NewProvider(ctx, obsx.Options{
    ServiceName:    "user-service",
    ServiceVersion: "1.0.0",
    OTLPEndpoint:   "otel-collector:4317",
})
```

**Key Features:**
- Trace and metric provider setup
- Configurable sampling
- Resource attributes
- Graceful shutdown

#### [httpx](httpx/) - HTTP Utilities
HTTP helpers for binding, validation, and security.

```go
var req UserRequest
httpx.BindAndValidate(r, &req)

handler := httpx.SecureMiddleware(httpx.DefaultSecurityHeaders())(next)
```

### L1: Foundation Layer

#### [logx](logx/) - Structured Logging
Structured logging based on `log/slog` with logfmt/JSON output.

```go
logger := logx.New(
    logx.WithFormat(logx.FormatLogfmt),
    logx.WithLevel(slog.LevelInfo),
)
logger.Info("user created", "user_id", "u-123")
```

**Key Features:**
- Logfmt and JSON formats
- Field sorting and colorization
- Payload limits and sensitive field masking
- Context-aware logging

### L0: Core Layer

#### [core/errors](core/errors/) - Error Types
Structured error types with codes for API responses.

#### [core/identity](core/identity/) - Identity Context
Request metadata and user identity extraction.

#### [core/log](core/log/) - Logger Interface
Zero-dependency logger interface.

### Auxiliary Modules

#### [storex](storex/) - Storage Abstraction
GORM-based storage with health checks.

```go
store, _ := storex.NewGORMStore(storex.GORMOptions{
    DSN:    "user:pass@tcp(localhost:3306)/mydb",
    Driver: "mysql",
    Logger: logger,
})
```

#### [k8sx](k8sx/) - Kubernetes Integration
ConfigMap watching and service discovery.

#### [testingx](testingx/) - Testing Utilities
Test helpers and utilities (planned).

## Design Patterns

### Interface-Implementation Separation

All major modules follow a clean architecture pattern:

```
module/
├── module.go              # Public API (~100-200 lines)
│   ├── Interface definitions
│   ├── Option functions
│   └── Constructor (delegates to internal)
└── internal/
    ├── implementation.go  # Actual logic
    ├── helpers.go         # Internal helpers
    └── types.go           # Internal types
```

**Benefits:**
- Public API surface is minimal and focused
- Implementation details hidden
- Easy to test and mock
- Clear separation of concerns

### Multi-Stage Initialization

Complex initialization is split into logical stages (servicex example):

```go
1. initializeLogger()       → Setup logging
2. initializeConfig()        → Load configuration
3. initializeDatabase()      → Connect database
4. initializeObservability() → Setup tracing
5. buildApp()                → Create app context
6. startServers()            → Start HTTP servers
7. gracefulShutdown()        → Cleanup resources
```

### Functional Options

All modules use functional options for configuration:

```go
servicex.Run(ctx,
    servicex.WithService("my-service", "1.0.0"),
    servicex.WithLogger(logger),
    servicex.WithAppConfig(cfg), // Auto-detects database
    servicex.WithAutoMigrate(&model.User{}),
    servicex.WithRegister(register),
)
```

**Configuration via Environment:**
```bash
# Service configuration
SERVICE_NAME=my-service
SERVICE_VERSION=1.0.0
ENV=production

# Log level control
LOG_LEVEL=info  # debug, info, warn, error

# Database (auto-detected from BaseConfig)
DB_DRIVER=mysql
DB_DSN=user:pass@tcp(localhost:3306)/mydb
DB_MAX_IDLE=10
DB_MAX_OPEN=100
DB_MAX_LIFETIME=1h

# Ports
HTTP_PORT=8080
HEALTH_PORT=8081
METRICS_PORT=9091
```

## Complete Example

See [examples/user-service](examples/user-service/) for a complete Connect-RPC service with:
- Configuration management
- Database integration with migrations
- Connect RPC handlers
- Health checks
- OpenTelemetry tracing

```bash
cd examples/user-service
make run
```

## Development

### Prerequisites

- Go 1.21+
- Docker and Docker Compose (for examples)
- Make

### Monorepo Structure

The egg framework uses a modular monorepo with sub-projects:

```bash
# Framework modules (root)
make setup          # First-time setup (install tools + init workspace)
make tidy           # Clean and update dependencies for all modules
make test           # Run tests for all modules with race detection
make lint           # Run linter (includes fmt + vet)
make coverage       # Generate test coverage report (HTML + terminal)
make check          # Quick validation (lint + test, no coverage)
make quality        # Full quality check (tidy + lint + test + coverage)
make clean          # Clean test artifacts and coverage files
make tools          # Install required development tools

# Release management
make release VERSION=v0.3.0       # Release all modules with version
make delete-all-tags              # Delete ALL version tags (DANGEROUS)

# CLI tool
cd cli && make build              # Build egg CLI
cd cli && make test-integration   # Run CLI integration tests

# Examples
cd examples && make docker-build  # Build example Docker images
cd examples && make deploy-up     # Start example services
cd examples && make test          # Run integration tests

# Foundation images
cd base-images && make help       # View base image build options
cd base-images && make build-all  # Build builder + runtime images
cd base-images && make push-all   # Build and push to registry
```

### Typical Development Workflow

```bash
# 1. First-time setup
make setup

# 2. Daily development
make tidy          # After go.mod changes
make check         # Before git commit
make coverage      # Check test coverage

# 3. Before release
make quality       # Full quality check

# 4. Release
make release VERSION=v0.3.0
```

### Foundation Images

Build multi-architecture foundation images for Egg services:

```bash
cd base-images

# Build both builder and runtime (for linux/amd64 and linux/arm64)
make build-all

# Build and push to registry
make build-all PUSH=true

# Build for specific platforms
make build-builder DOCKER_PLATFORM=linux/amd64

# Build with specific Go version
make build-all GO_VERSION=1.25.2
```

See [base-images/README.md](base-images/README.md) for detailed documentation.

## Project Structure

```
egg/
├── core/            # L0: Core types and interfaces
│   ├── errors/
│   ├── identity/
│   └── log/
├── logx/            # L1: Structured logging
├── configx/         # L2: Configuration management
├── obsx/            # L2: OpenTelemetry provider
├── httpx/           # L2: HTTP utilities
├── runtimex/        # L3: Lifecycle management
├── connectx/        # L3: Connect interceptors
├── clientx/         # L3: Connect client
├── servicex/        # L4: Service integration
├── storex/          # Auxiliary: Storage
├── k8sx/            # Auxiliary: Kubernetes
├── cli/             # CLI tool for project scaffolding
│   ├── Makefile     # CLI-specific build targets
│   └── README.md    # CLI documentation
├── examples/        # Example services with deployment
│   ├── Makefile     # Example-specific targets
│   ├── deploy/      # Docker Compose configurations
│   └── scripts/     # Integration test scripts
├── base-images/     # Foundation Docker images
│   ├── Makefile              # Image build targets
│   ├── README.md             # Image documentation
│   ├── Dockerfile.builder    # Multi-arch Go builder
│   └── Dockerfile.runtime    # Multi-arch Alpine runtime
├── docs/            # Documentation
│   └── makefile-optimization.md  # Makefile optimization guide
└── scripts/         # Build and utility scripts
    └── logger.sh    # Unified logging for Makefile and scripts
```

## Code Quality Standards

All modules follow strict quality standards:

1. **Documentation**: All exported symbols have GoDoc comments in English
2. **Testing**: Comprehensive test coverage with table-driven tests
3. **Linting**: Clean `golangci-lint` output
4. **File Size**: Public interface files < 300 lines, implementation < 500 lines
5. **Dependencies**: Minimal external dependencies, managed via `go work`

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Key Contribution Areas

- [ ] Additional storage backends (Redis, MongoDB)
- [ ] More comprehensive testing utilities
- [ ] Performance benchmarks
- [ ] Additional examples
- [ ] Documentation improvements

## Roadmap

- [x] Core framework modules (L0-L4)
- [x] Connect-RPC integration
- [x] OpenTelemetry observability
- [x] Configuration management with hot reload
- [x] Database integration with GORM
- [ ] Redis cache integration
- [ ] Message queue integration (Kafka, RabbitMQ)
- [ ] Service mesh integration (Istio, Linkerd)
- [ ] CLI tool for project scaffolding
- [ ] GraphQL support

## Documentation

- [Architecture Guide](docs/ARCHITECTURE.md) - Detailed architecture documentation
- [Logging Standards](docs/LOGGING.md) - Logging format and practices
- [Code Guidelines](docs/guidance.md) - Code quality standards
- [Module Guide](docs/guide.md) - Module-by-module guide

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Built with ❤️ by the EggyByte team.

Special thanks to:
- [Connect-RPC](https://connectrpc.com/) for the excellent RPC framework
- [OpenTelemetry](https://opentelemetry.io/) for observability standards
- The Go community for inspiration and best practices

---

**Made with Go 🚀 | Built for Production 🏭 | Designed for Scale 📈**
