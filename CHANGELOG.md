# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Makefile**: New `make tidy` command to clean and update dependencies for all modules
- **Makefile**: New `make coverage` command to generate test coverage reports (HTML + terminal)
- **Makefile**: New `make check` command for quick validation (lint + test, no coverage)
- **Makefile**: New `make quality` command for full quality check (tidy + lint + test + coverage)
- **base-images**: Dedicated Makefile for foundation image management (builder + runtime)
- **base-images**: Comprehensive README.md with multi-arch build documentation
- **docs**: `makefile-optimization.md` guide documenting build system improvements
- **configx**: `LogLevel` field in `BaseConfig` for unified log level configuration
- **logx**: `ParseLevel()` function to convert log level strings to `slog.Level`
- **logx**: `NewFromEnv()` convenience function for zero-config logger creation from `LOG_LEVEL` environment variable
- **servicex**: `BaseConfigProvider` interface for modern, type-safe configuration handling
- **servicex**: Automatic default logger creation from `LOG_LEVEL` environment variable
- **examples**: `GetBaseConfig()` method in AppConfig implementing BaseConfigProvider interface
- **examples**: Production-ready examples with complete GoDoc documentation

### Changed

- **Makefile**: All commands now use unified `scripts/logger.sh` for consistent output formatting
- **Makefile**: Removed redundant `make build` (library modules don't need build)
- **Makefile**: Removed redundant `make fmt` and `make vet` (already included in lint)
- **Makefile**: Removed `make publish-modules` (merged into `make release`)
- **Makefile**: Moved Docker foundation image builds to `base-images/Makefile`
- **Makefile**: Shell loops now use `source $(LOGGER)` for proper function execution
- **Makefile**: Improved error handling with better exit codes and failure tracking
- **Project Structure**: Renamed `docker/` to `base-images/` for clearer purpose
- **scripts/logger.sh**: Changed from `echo -e` to `printf` for better portability and reliability
- **scripts/logger.sh**: Improved formatting functions to avoid shell interpretation issues
- **.golangci.yml**: Simplified `gocritic` configuration to remove redundant disabled-checks
- **.golangci.yml**: Removed warnings about already-disabled checks (`rangeValCopy`, `hugeParam`)
- **.gitignore**: Added `.coverage/` directory and `coverage.out` file patterns
- **servicex**: Default logger now uses console format with colors for better development experience
- **servicex**: Logger initialization reads `LOG_LEVEL` from BaseConfig (via configx) for unified configuration management
- **servicex**: All configuration now unified through configx (no direct environment variable reads)
- **examples/user-service**: Removed mock repository fallback; database is now mandatory for production compliance
- **examples/user-service**: Enhanced GoDoc documentation across all internal packages (config, model, repository, service, handler)
- **examples/user-service**: Simplified `main.go` to use servicex auto-configuration (no manual logger creation)
- **examples/minimal-connect-service**: Moved `main.go` to `cmd/server/` for standard project structure
- **examples/minimal-connect-service**: Simplified `main.go` to use servicex auto-configuration
- **build**: Updated `build.sh` to correctly build both services from `cmd/server` directory

### Improved

- **Build System**: Unified logging format across Makefile and shell scripts using `logger.sh`
- **Build System**: Reduced root Makefile from 367 lines to 231 lines (37% reduction)
- **Build System**: Clearer separation of concerns (framework vs. base images vs. examples)
- **Developer Experience**: Consistent colored output with proper formatting across all commands
- **Developer Experience**: Better error messages with clear success/failure indicators
- **Code Quality**: No more `/bin/sh: @echo: command not found` errors in Makefile execution
- **Code Quality**: Cleaner `golangci-lint` output without metadata warnings
- **Documentation**: All internal packages now have comprehensive English GoDoc comments
- **Documentation**: Added detailed parameter, return value, concurrency, and performance notes to public APIs
- **Documentation**: Service layer, repository layer, and model layer fully documented with usage examples
- **Simplification**: Services can now start with zero logger configuration while maintaining full customization support
- **Architecture**: Unified configuration management - all settings flow through configx (BaseConfig), not direct environment reads
- **Architecture**: Replaced reflection-based config extraction with interface-based approach (50+ lines removed, type-safe)
- **Consistency**: Logger configuration follows the same pattern as other framework components (config-first approach)
- **Code Quality**: Eliminated reflection usage in servicex configuration handling for better maintainability and compile-time safety

### Removed

- **Makefile**: Removed `make build` (not needed for library modules)
- **Makefile**: Removed `make fmt` (redundant with lint)
- **Makefile**: Removed `make vet` (redundant with lint)
- **Makefile**: Removed `make publish-modules` (merged into release)
- **Makefile**: Removed `make security` from quality check (moved to optional/manual execution)
- **Makefile**: Removed all Docker-related targets from root (moved to base-images/)
- **examples/user-service**: Removed ~150 lines of mock repository implementation
- **examples/user-service**: Removed fallback to in-memory storage (production best practice)

### Fixed

- **Makefile**: Fixed shell script execution errors (`@echo: command not found`) by using `source $(LOGGER)`
- **Makefile**: Fixed golangci-lint warnings about gocritic configuration
- **Makefile**: Fixed subshell navigation using `(cd module && command)` pattern
- **scripts/logger.sh**: Fixed `-e` flag being printed in output by switching to `printf`
- **.golangci.yml**: Fixed unnecessary disabled-checks configuration for gocritic
- **connectx**: Added panic protection in metrics interceptor when accessing response body
- **connectx**: Enhanced recovery interceptor logging for better panic debugging
- **connectx**: Protected response size recording from potential nil pointer dereferences in error scenarios

### Security

## [0.2.0-beta.2] - 2025-10-26

### Added

- **obsx**: Runtime metrics collection (goroutines, GC, heap, stack)
- **obsx**: Process metrics collection (CPU, RSS, uptime, start time)
- **obsx**: Database connection pool metrics (open, in-use, idle, waits)
- **obsx**: `EnableRuntimeMetrics()` method for Go runtime observability
- **obsx**: `EnableProcessMetrics()` method for process-level observability
- **obsx**: `RegisterDBMetrics()` method for database connection pool monitoring
- **obsx**: `RegisterGORMMetrics()` method for GORM database monitoring
- **connectx**: Metrics interceptor with Prometheus-compatible RPC metrics
- **connectx**: Exemplar support for trace-metric correlation
- **clientx**: Client-side RPC metrics collection
- **servicex**: `WithMetricsConfig()` option for fine-grained metrics control
- **servicex**: `MetricsConfig` struct for selective metric enablement
- **examples**: Runtime and process metrics enabled in minimal-connect-service and user-service
- **docs**: Comprehensive metrics implementation guide (METRICS_IMPLEMENTATION.md)

### Changed

- **obsx**: Refactored module structure following egg standards (all logic moved to `internal/`)
- **obsx**: Module now exports only public API in `obsx.go`, implementation in `internal/`
- **obsx**: Removed `otel_scope_*` labels from Prometheus metrics (30%+ cardinality reduction)
- **connectx**: Standardized metric names with proper units (`_seconds`, `_bytes`, `_total`)
- **connectx**: Updated histogram buckets to industry-standard ranges
- **connectx**: Split `rpc_procedure` into `rpc_service` + `rpc_method` labels
- **servicex**: Fixed initialization to enable metrics even when tracing is disabled
- **servicex**: Enhanced observability initialization with conditional metric enablement
- **examples/connect-tester**: Added runtime/process metrics validation
- **Module organization rule**: Updated to require single export file per module root

### Fixed

- **servicex**: Fixed bug where `EnableTracing=false` prevented metrics initialization
- **obsx**: Corrected MeterProvider type signatures in internal implementation

## [0.2.0] - 2025-10-25

### Added

- **servicex**: New `WithAppConfig()` option for automatic database configuration detection from `configx.BaseConfig`
- **servicex**: Environment-based log level control via `LOG_LEVEL` environment variable (supports: debug, info, warn, error)
- **servicex**: Automatic database configuration extraction after environment variable binding
- **servicex**: Database DSN masking in logs for security
- **core/log**: New helper functions `Int32()`, `Int64()`, `Float64()`, and `String()` for structured logging
- **examples/connect-tester**: Enhanced test coverage with multi-language greetings, batch operations, and comprehensive error scenarios
- **examples/connect-tester**: Detailed test output with colored results and metrics
- **scripts/test-examples.sh**: Improved test flow with explicit infrastructure management
- **scripts/build.sh**: Docker daemon status check to prevent build failures
- **Documentation**: Comprehensive README updates across all modules with servicex integration examples
- **Documentation**: Migration guides from old patterns to new simplified patterns

### Changed

- **servicex**: `WithAppConfig()` now combines configuration binding and database auto-detection in one call
- **servicex**: Logger initialization now prioritizes `LOG_LEVEL` environment variable over `WithDebugLogs()`
- **servicex**: Database configuration is extracted from `BaseConfig` after `configx.Manager.Bind()` completes
- **servicex**: Improved initialization logging with database configuration preview (masked DSN)
- **examples/minimal-connect-service**: Updated to use `LOG_LEVEL` environment variable
- **examples/user-service**: Refactored to use `WithAppConfig()` for simplified database configuration
- **examples/user-service**: Enhanced main.go with comprehensive English documentation
- **examples/connect-tester**: Complete rewrite with structured test suites and detailed reporting
- **scripts/test-examples.sh**: Removed `--remove-orphans` flag to preserve infrastructure services
- **scripts/test-examples.sh**: Enhanced infrastructure health checks (MySQL, Jaeger, OTEL Collector)
- **All READMEs**: Updated with modern patterns, environment variable documentation, and best practices

### Deprecated

- **servicex**: `WithDebugLogs()` is now deprecated in favor of `LOG_LEVEL` environment variable
- **servicex**: Separate `WithConfig()` + `WithDatabase()` pattern deprecated in favor of `WithAppConfig()`

### Fixed

- **servicex**: Fixed database configuration not being detected when using `WithAppConfig()` with `BaseConfig`
- **servicex**: Fixed log level not being applied from environment variable
- **servicex**: Fixed infrastructure services being shut down during example tests
- **scripts/test-examples.sh**: Fixed Docker daemon accessibility check with diagnostic messages
- **scripts/test-examples.sh**: Fixed infrastructure services being incorrectly stopped after tests
- **examples/connect-tester**: Fixed linter errors with undefined log helper functions
- **examples/connect-tester**: Fixed variable scoping issues in test functions

### Documentation

- **Root README**: Added "Recent Improvements" section highlighting v0.2.0 features
- **servicex/README**: Added "Database Configuration Auto-Detection" and "Log Level Control" sections
- **servicex/README**: Added comprehensive troubleshooting guide
- **configx/README**: Added "Integration with servicex" section with auto-detection explanation
- **logx/README**: Added "Log Level Configuration" section with servicex integration
- **core/log/README**: Updated with new helper functions and usage examples
- **All module READMEs**: Added servicex integration examples showing recommended patterns
- **All example READMEs**: Updated with LOG_LEVEL usage and modern configuration patterns

## [0.1.0] - 2025-10-20

### Added
- Initial release of Egg framework
- Core module with zero-dependency interfaces
- Runtime management with unified port strategy
- Connect integration with unified interceptors
- OpenTelemetry observability support
- Kubernetes integration for ConfigMap watching and service discovery
- Storage abstraction interfaces
- Docker buildx support for multi-platform image builds (amd64, arm64)
- Production integration test script (`scripts/test-cli-production.sh`)
- Go workspace (`go.work`) support for local development

### Changed
- CLI `build` command now uses Docker buildx by default
- Added `--buildx` and `--platforms` flags to build command

### Fixed
- **Critical**: Removed `replace` directives from all module `go.mod` files
- **Critical**: Updated all internal dependencies to use proper version `v0.0.1`
- Fixed remote dependency resolution for external users
- Module dependencies now work correctly without local repository

## [0.0.1] - 2025-10-17

### Added
- Core logging interface compatible with slog concepts
- Structured error handling with error codes
- User identity and request metadata context management
- Utility functions for retry logic and common operations
- Runtime lifecycle management with graceful shutdown
- HTTP/2 and HTTP/2 Cleartext support
- Connect interceptors for tracing, logging, and error handling
- OpenTelemetry provider initialization
- Kubernetes ConfigMap watching
- Service discovery for headless and ClusterIP services
- Storage interface definitions and health checks

### Changed

### Deprecated

### Removed

### Fixed

### Security

---

## Legend

- **Added** for new features
- **Changed** for changes in existing functionality
- **Deprecated** for soon-to-be removed features
- **Removed** for now removed features
- **Fixed** for any bug fixes
- **Security** for vulnerability fixes
