# Egg - Production-Ready Go Microservices Framework

**A modern, layered Go framework for building Connect-RPC microservices with observability, configuration management, and clean architecture.**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg?style=flat-square)](LICENSE)
[![Version](https://img.shields.io/badge/Version-0.3.1-blue.svg?style=flat-square)](CHANGELOG.md)

---

## Overview

Egg is a comprehensive microservices framework designed for building production-grade Go services with minimal boilerplate. It provides:

- **One-line service startup** with integrated configuration, logging, database, and tracing
- **Egg CLI tool** for project scaffolding and service generation (recommended)
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

Install the Egg CLI tool:

```bash
go install go.eggybyte.com/egg/cli/cmd/egg@latest
```

### Create Your First Service

```bash
# Initialize a new project
egg init my-project

# Create a backend service
cd my-project
egg create service backend --type crud

# Start services
egg build all --local
egg compose up
```

That's it! The CLI generates all necessary files including `main.go`, `go.mod`, Docker configuration, and more.

See [CLI Documentation](cli/README.md) for complete usage guide.

### Manual Setup (Advanced)

If you prefer to use the framework modules directly without the CLI:

```bash
go get go.eggybyte.com/egg/servicex@latest
```

See [servicex/README.md](servicex/README.md) for manual setup examples.

## Key Innovations

### 1. Strict Layered Architecture

Egg enforces a strict layered architecture to prevent circular dependencies and ensure maintainability:

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

**Dependency Rules:**
- A module can only depend on modules in the same or lower layers
- No circular dependencies between modules
- Core modules (L0) have zero external dependencies
- Public interface files are thin (~100-200 lines), complex logic in `internal/`

### 2. CLI-Driven Development

The Egg CLI eliminates boilerplate and enforces best practices:

- **Project scaffolding** - Generate complete project structure
- **Service generation** - Create backend/frontend services with templates
- **Code generation** - Automatic Protobuf code generation with local plugins
- **Build automation** - Multi-platform Docker builds
- **Docker Compose** - Automatic service orchestration
- **Kubernetes** - Helm chart generation

See [CLI Documentation](cli/README.md) for details.

### 3. Production-Ready Defaults

- **Zero-config logging** - Auto-configured from `LOG_LEVEL` environment variable
- **Database auto-detection** - Automatic configuration from `BaseConfig`
- **Hot reload** - Configuration changes without restart
- **Observability** - OpenTelemetry integration out of the box
- **Health checks** - Built-in health endpoints
- **Graceful shutdown** - Proper resource cleanup

## Modules

### Core Modules

- **[core](core/)** - Zero-dependency core types and interfaces
  - `core/errors` - Structured error types with codes
  - `core/identity` - Request metadata and user identity
  - `core/log` - Logger interface

- **[logx](logx/)** - Structured logging (L1)
  - Logfmt and JSON formats
  - Field sorting and colorization
  - Sensitive field masking

- **[configx](configx/)** - Configuration management (L2)
  - Multiple sources (Env, File, K8s ConfigMap)
  - Hot reload with debouncing
  - Struct binding with validation

- **[obsx](obsx/)** - OpenTelemetry provider (L2)
  - Trace and metric provider setup
  - Configurable sampling
  - Resource attributes

- **[httpx](httpx/)** - HTTP utilities (L2)
  - Request binding and validation
  - Security headers middleware
  - CORS support

- **[runtimex](runtimex/)** - Service lifecycle management (L3)
  - Concurrent service startup/shutdown
  - Health check aggregation
  - Multiple server support

- **[connectx](connectx/)** - Connect RPC interceptors (L3)
  - Timeout enforcement
  - Structured logging with correlation
  - Error mapping to Connect codes
  - OpenTelemetry tracing integration

- **[clientx](clientx/)** - Connect client factory (L3)
  - Retry logic with exponential backoff
  - Circuit breaker support
  - Request timeouts

- **[servicex](servicex/)** - One-line service startup (L4)
  - Integrated configuration, logging, database, tracing
  - Automatic database configuration
  - Environment-based log level control
  - Dependency injection container

### Auxiliary Modules

- **[storex](storex/)** - Storage abstraction (GORM-based)
- **[k8sx](k8sx/)** - Kubernetes integration (ConfigMap watching, service discovery)
- **[testingx](testingx/)** - Testing utilities (98.4% coverage)

Each module has its own README with detailed documentation and examples.

## Examples

- **[minimal-connect-service](examples/minimal-connect-service/)** - Minimal Connect-RPC service
- **[user-service](examples/user-service/)** - Complete CRUD service with database
- **[connect-tester](examples/connect-tester/)** - Integration testing tool

See [examples/README.md](examples/README.md) for details.

## Quality & Testing

- **Test Coverage**: 57.27% overall (up from 33.06%)
  - Key modules: `testingx` (98.4%), `httpx` (~100%), `logx` (86.4%)
- **Code Quality**: All modules follow strict standards
  - Comprehensive GoDoc comments (English)
  - Table-driven tests
  - Clean `golangci-lint` output
  - Minimal external dependencies

See [docs/guidance.md](docs/guidance.md) for code quality standards.

## Documentation

- [Architecture Guide](docs/ARCHITECTURE.md) - Detailed architecture documentation
- [Logging Standards](docs/LOGGING.md) - Logging format and practices
- [Code Guidelines](docs/guidance.md) - Code quality standards
- [Module Guide](docs/guide.md) - Module-by-module guide
- [CLI Documentation](cli/README.md) - Complete Egg CLI reference
- [CLI Design](docs/egg-cli.md) - CLI design and implementation details

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Key Contribution Areas

- [x] Comprehensive testing utilities (`testingx` module)
- [ ] Additional storage backends (Redis, MongoDB)
- [ ] Performance benchmarks
- [ ] Additional examples
- [ ] Documentation improvements

## Roadmap

- [x] Core framework modules (L0-L4)
- [x] Connect-RPC integration
- [x] OpenTelemetry observability
- [x] Configuration management with hot reload
- [x] Database integration with GORM
- [x] CLI tool for project scaffolding (`egg` CLI)
- [x] Comprehensive test coverage (57.27% overall)
- [ ] Redis cache integration
- [ ] Message queue integration (Kafka, RabbitMQ)
- [ ] Service mesh integration (Istio, Linkerd)
- [ ] GraphQL support

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
