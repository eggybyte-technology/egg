# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.1] - 2025-01-31

### Changed

- **Build System**: Unified `.gitignore` to root directory only
  - Removed all subdirectory `.gitignore` files
  - Simplified rules using global patterns (bin/, build/, tmp/, logs/, etc.)
  - All directories with standard names (bin, build, gen, tmp, logs) are automatically ignored regardless of location
- **Release Process**: Simplified CLI release command
  - Short parameters: `make cli-release CLI=v1.0.0 FW=v0.3.0`
  - Still supports full parameters: `make cli-release VERSION=v1.0.0 FRAMEWORK_VERSION=v0.3.0`
- **Release Process**: Added large file check before release
  - Automatically checks for files >1MB in git-tracked files
  - Prompts for confirmation if large files are found
  - Helps prevent accidental commits of large binaries or artifacts

### Fixed

- **CLI**: Fixed `cli/cmd/` source files not being tracked in git
  - Removed incorrect `egg` pattern from `cli/.gitignore` that was matching `cli/cmd/egg/` directory
  - All CLI source files now properly tracked in git repository

### Added

- **CLI**: Added `cli/cmd/egg/` source files to git repository
  - All CLI command implementations now tracked (api.go, build.go, check.go, compose.go, create.go, doctor.go, init.go, kube.go, main.go)

## [0.3.0] - 2025-01-31

### Added

- **Testing**: Comprehensive test coverage improvements across all modules
  - `testingx`: Test utilities with 98.4% coverage (MockLogger, CaptureLogger, context helpers)
  - `logx/internal`: Handler tests with 92.3% coverage (formatting, masking, concurrency)
  - `httpx/internal`: Middleware tests with 100% coverage (security headers, CORS)
  - `obsx/internal`: Provider and metrics tests with 69.8% coverage (OpenTelemetry integration)
  - `runtimex/internal`: Health check tests with 20% coverage (registry, checkers)
  - `storex/internal`: GORM adapter tests with 59.4% coverage (connection pooling, registry)
  - `configx/internal`: Manager, sources, and validator tests with 66.4% coverage
  - `connectx/internal`: Interceptor helper function tests with 23.3% coverage
  - `clientx/internal`: Retry transport tests with 95.8% coverage (retry logic, circuit breaker)
- **Testing**: Overall test coverage increased from 33.06% to 57.27% (+24.21 percentage points, +73.3% improvement)
- **Testing**: Added comprehensive test suites for internal implementations
  - Table-driven tests following Go best practices
  - Concurrent safety verification
  - Error handling and edge case coverage
  - Mock implementations for external dependencies

### Improved

- **Testing**: Significantly improved test coverage across core modules
  - `testingx`: From 0% to 98.4% coverage (foundational testing utilities)
  - `logx`: From 20.1% to 86.4% coverage (logging handler and formatting)
  - `httpx`: From 51.4% to ~100% coverage (HTTP middleware)
  - `storex`: From 9.4% to 63.2% coverage (database adapters)
  - `configx`: From 17.6% to 64.6% coverage (configuration management)
  - `obsx`: From 5.5% to 68.8% coverage (observability)
  - `clientx`: From 19.3% to 47.0% coverage (HTTP client with retry)
  - `runtimex`: From 14.3% to 28.6% coverage (runtime management)
- **Testing**: Enhanced test reliability and maintainability
  - Proper mock implementations for external dependencies
  - Table-driven test patterns for comprehensive coverage
  - Concurrent safety verification in all test suites
  - Edge case and error path testing

### Added

- **CLI**: Docker Compose service testing via Docker internal network
  - Tests now access services using Docker service names instead of localhost
  - Health checks and RPC tests run inside Docker network using `docker compose exec`
  - Supports pattern matching for metrics endpoint validation
- **CLI**: Enhanced test script modularity
  - Removed redundant tests from integration test suite
  - Streamlined test flow for better maintainability
  - Services remain running after tests for manual inspection

### Changed

- **CLI**: Docker Compose templates no longer expose ports to localhost
  - Services are accessed via Docker internal network only
  - Removed port mappings from compose.yaml generation
  - Improved security by preventing accidental local port exposure
- **CLI**: Unified logging format across CLI and shell scripts
  - CLI `ui` package now matches `logger.sh` output format
  - Removed redundant status text (SUCCESS, ERROR) from logger output
  - Info and debug messages without prefixes for cleaner output
  - Full-line coloring for better readability
  - Section headers use simple white bullet point (•)
  - Enhanced contrast for command and section outputs
- **CLI**: Improved `egg doctor` command output
  - Removed redundant prefixes for cleaner display
  - Better visual hierarchy with standardized formatting
- **CLI**: Optimized `egg check` command output
  - Added summary display at the beginning
  - Consistent formatting for errors, warnings, and info messages
  - Better organized results grouping

### Improved

- **Scripts**: Enhanced logger.sh with cleaner output format
  - Success messages: `[✓] message` (no "SUCCESS:" prefix)
  - Error messages: `[✗] message` (no "ERROR:" prefix)
  - Info messages: `message` (white, no prefix)
  - Debug messages: `message` (magenta, no prefix)
  - Warning messages: `[!] message` (yellow)
  - Section headers: `• Section Name` (white)
  - Commands: `[CMD] command` (bright cyan, enhanced contrast)
- **Testing**: Docker Compose integration tests now use Docker network DNS
  - Health checks: `http://service-name:port/path`
  - RPC tests: `http://service-name:port/rpc-path`
  - Metrics tests: `http://service-name:port/metrics` with pattern matching
  - Tests execute curl commands inside Docker containers via `docker compose exec`
- **CLI**: Improved code quality and linter compliance
  - All linter errors resolved (unlambda, wsl, errcheck)
  - Proper error handling annotations for CLI output functions
  - Cleaner code structure following Go best practices
- **Build System**: Improved lint command reliability
  - Makefile lint target now correctly handles empty output scenarios
  - Fixed false-positive failures in multi-module lint runs

### Removed

- **CLI**: Removed redundant integration tests from test-cli.sh
  - Test 7: Runtime image check (no longer built locally)
  - Test 9: Build Docker Image (merged into Test 8)
  - Test 10: Docker Compose Validation (covered by Test 12)
  - Test 13: Validate egg.yaml structure (covered by Test 2)
  - Test 15: API Code Generation Verification (covered by Test 5)
  - Test 16: Service Compilation (covered by Test 8)
  - Test 17: Syntax Validation (covered by build process)
- **CLI**: Removed Docker Compose port mappings from templates
  - Backend services no longer expose HTTP/Health/Metrics ports
  - Frontend services no longer expose web ports
  - Database services no longer expose database ports
  - Services accessed via Docker network only

### Fixed

- **CLI**: Fixed Docker Compose test failures caused by removed port mappings
  - Tests now correctly access services via Docker internal network
  - Health checks work with Docker service names instead of localhost
  - RPC tests execute inside Docker containers for proper network access
- **CLI**: Fixed `egg doctor` output showing redundant prefixes (`[✓] [+] Go`)
- **CLI**: Fixed `egg check` output format inconsistencies
- **CLI**: Fixed code quality issues identified by linter
  - Simplified template function in `templates.go` (removed unnecessary lambda wrapper)
  - Fixed case block formatting in `ui.go` (added proper newlines per wsl linter)
  - Added proper error handling annotations for stdout/stderr writes in CLI context
  - Simplified single-case select statement in `compose.go`
- **Makefile**: Fixed lint command logic to handle empty grep output correctly
  - Lint now correctly passes when all modules have no errors
  - Fixed false-positive failures caused by grep exit code when no matches found

### Added

- **CLI**: Support for optional database dependency based on proto template type
  - `echo` template services can start without database (no DB_DSN required)
  - `crud` template services require database (DB_DSN mandatory)
  - Handler templates adapt based on template type (Ping only vs full CRUD)
- **CLI**: Conditional file generation - only CRUD templates generate model/repository/service files
- **CLI**: Enhanced Docker Compose configuration with servicex-standard environment variables
  - `SERVICE_NAME`, `SERVICE_VERSION`, `APP_ENV`, `LOG_LEVEL` automatically configured
  - Database configuration (`DB_DSN`, `DB_DRIVER`) included when database enabled
  - Health checks and restart policies added to all services
- **CLI**: `--local` flag for `build backend` and `build frontend` commands
  - Builds for local platform only (no push)
  - Automatically detects platform (linux/amd64 or linux/arm64)
- **CLI**: Default database enabled in new projects (`database.enabled: true`)

### Changed

- **CLI**: Docker Compose now uses pre-built images instead of building during compose
  - Services reference images built via `egg build all --local` or `egg build all --push`
  - Image names follow pattern: `<docker_registry>/<project_name>-<service_name>:<version>`
- **CLI**: `egg build all` defaults to multi-platform build with push
  - Use `--local` flag to build for local platform only (no push)
- **CLI**: Frontend build process simplified
  - Flutter web assets now copied directly from `frontend/<service>/build/web`
  - Removed intermediate copy step to `bin/frontend/<service>`
- **CLI**: `ping` service default uses `echo` template (simplest Ping RPC only)
- **CLI**: Backend service templates now conditionally generate code based on proto type
  - Echo services: handler only (no database dependency)
  - CRUD services: full stack (handler/service/repository/model with database)
- **CLI**: Test integration script defaults to keeping test directory (`--keep` behavior)
  - Use `--remove` flag to clean up test directory after completion

### Fixed

- **CLI**: Fixed Docker Compose attempting to build images instead of using pre-built ones
- **CLI**: Fixed database DSN requirement error for echo/ping services
- **CLI**: Fixed service registration failing when database not configured for echo services
- **CLI**: Fixed frontend build copying assets to wrong location

### Improved

- **CLI**: Better separation between minimal services (echo) and full services (crud)
- **CLI**: Docker Compose configuration aligned with servicex environment variable standards
- **CLI**: Default project configuration enables database for easier development setup

### Added

- **CLI**: Multi-platform Docker image build support with `docker buildx` (linux/amd64, linux/arm64)
- **CLI**: `buildMultiPlatformImage()` function for automated multi-arch builds with push
- **CLI**: Frontend service name validation enforcing underscore usage (Flutter requirement)
- **CLI**: Automatic service name conversion for Docker images (underscores to hyphens)
- **CLI**: Cross-type service name conflict detection (backend vs frontend)
- **CLI**: Comprehensive service name uniqueness validation
- **CLI**: Local dev version support (v0.0.0-dev) with GOPROXY=direct and GOSUMDB=off
- **CLI**: `GoWithEnv()` method in toolrunner for environment-specific command execution
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
- **examples**: Dedicated build script (`examples/scripts/build-examples.sh`) for building service binaries and Docker images
- **examples**: Comprehensive test script (`examples/scripts/test-examples.sh`) with infrastructure management and health checks
- **examples**: Docker Compose infrastructure setup (MySQL) for local development and testing
- **examples**: Enhanced `connect-tester` with comprehensive metrics endpoint validation (RPC, Runtime, Process, Database metrics)
- **examples**: `examples/bin/` directory for compiled binaries (Git-tracked with `.gitkeep`)

### Changed

- **CLI**: Flutter web build now correctly copies from `build/web` to output directory
- **CLI**: Backend service creation now uses version-based dependencies (v0.0.0-dev) instead of replace directives for Docker compatibility
- **CLI**: Frontend service names in Docker images use hyphens (e.g., `admin-portal`) while source uses underscores (e.g., `admin_portal`)
- **CLI**: Service creation no longer supports `--force` flag; duplicate names are rejected
- **CLI**: Multi-platform builds automatically use `--push` flag (buildx limitation)
- **CLI**: Build commands provide clear guidance when multi-platform requires push
- **toolrunner**: Error formatting improved with proper wrapping and cleaner output
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
- **examples**: Refactored build system to use dedicated `build-examples.sh` script instead of root `build.sh`
- **examples**: Simplified infrastructure to MySQL-only (removed Jaeger and OTEL Collector; tracing disabled, metrics-only)
- **examples**: Standardized Docker Compose configuration with proper servicex environment variables
- **examples**: Updated Docker Compose files to remove obsolete `version` field (Compose V2)
- **examples**: Improved port cleanup script to only check necessary ports (MySQL + application services)

### Improved

- **CLI**: Multi-platform Docker builds with automatic platform detection and buildx support
- **CLI**: Service name validation prevents conflicts across all service types (backend/frontend)
- **CLI**: Flutter web build reliability with correct output path handling
- **CLI**: Docker image naming consistency (hyphens for images, underscores for Flutter packages)
- **CLI**: Better error messages for multi-platform builds without push flag
- **CLI**: Development workflow with v0.0.0-dev versions works in both local and Docker environments
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
- **Testing**: All framework modules now pass `go test -race` with zero data race warnings
- **Testing**: Enhanced test reliability with proper synchronization primitives in concurrent test code
- **Testing**: Improved test isolation by using random port allocation to prevent conflicts during parallel execution

### Removed

- **CLI**: Removed `--force` flag from service creation commands (enforces unique service names)
- **CLI**: Removed replace directives for egg modules in generated go.mod files (use v0.0.0-dev versions)
- **Makefile**: Removed `make build` (not needed for library modules)
- **Makefile**: Removed `make fmt` (redundant with lint)
- **Makefile**: Removed `make vet` (redundant with lint)
- **Makefile**: Removed `make publish-modules` (merged into release)
- **Makefile**: Removed `make security` from quality check (moved to optional/manual execution)
- **Makefile**: Removed all Docker-related targets from root (moved to base-images/)
- **examples/user-service**: Removed ~150 lines of mock repository implementation
- **examples/user-service**: Removed fallback to in-memory storage (production best practice)

### Fixed

- **CLI**: Fixed Flutter web build output path (now correctly copies from `build/web/`)
- **CLI**: Fixed Docker builds failing with replace directives by using version-based dependencies
- **CLI**: Fixed service name conflicts not being detected across backend/frontend types
- **CLI**: Fixed frontend service creation allowing invalid hyphenated names (now enforces underscores)
- **CLI**: Fixed Docker image names not following naming conventions (now converts underscores to hyphens)
- **CLI**: Fixed multi-platform builds failing without proper --push handling
- **toolrunner**: Fixed error message formatting with proper fmt.Errorf wrapping
- **Makefile**: Fixed shell script execution errors (`@echo: command not found`) by using `source $(LOGGER)`
- **Makefile**: Fixed golangci-lint warnings about gocritic configuration
- **Makefile**: Fixed subshell navigation using `(cd module && command)` pattern
- **scripts/logger.sh**: Fixed `-e` flag being printed in output by switching to `printf`
- **.golangci.yml**: Fixed unnecessary disabled-checks configuration for gocritic
- **connectx**: Added panic protection in metrics interceptor when accessing response body
- **connectx**: Enhanced recovery interceptor logging for better panic debugging
- **connectx**: Protected response size recording from potential nil pointer dereferences in error scenarios
- **examples**: Fixed Docker platform mismatch warnings by using `docker buildx` with explicit `--platform` for multi-arch base images
- **examples**: Fixed MySQL 9.4 configuration (removed obsolete `--default-authentication-plugin` parameter)
- **examples**: Fixed Docker Compose orphan container warnings (expected behavior, documented in comments)
- **examples**: Fixed database status detection logic to be more robust and informative
- **examples**: Fixed Docker build to automatically select correct platform variant (arm64/amd64) matching binary architecture
- **core/identity**: Fixed nil value handling - `WithUser(ctx, nil)` and `WithMeta(ctx, nil)` now return context unchanged instead of storing nil
- **core/identity**: Fixed `UserFrom()` and `MetaFrom()` to correctly return `false` when stored value is nil
- **runtimex**: Fixed data race in test logger by adding mutex protection for concurrent log writes
- **runtimex**: Fixed data race in mock service by adding mutex protection for state access
- **servicex**: Fixed shutdown hooks not being called - hooks registered via `AddShutdownHook()` are now properly copied back to internal App
- **servicex**: Fixed port conflicts in integration tests by using random port allocation (`HTTP_PORT=0`) for parallel test execution

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
