# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.3-alpha.2] - 2025-11-07

### Changed

- **servicex**: Standardized client usage pattern across all services
  - Services now use injected clients exclusively (no temporary client creation)
  - Simplified `ValidateInternalToken` to demonstrate client capability by calling greet service
  - Removed redundant `expectedInternalToken` and `greetServiceURL` fields from service structs
  - Service constructors now accept only necessary dependencies (reduced parameters)
- **servicex**: Consolidated handler wrapper functions for cleaner API
  - Removed `WrapHandler` and `WrapHandlerWithToken` (redundant wrappers)
  - `CallService` now directly includes logging logic
  - `CallServiceWithToken` wraps `CallService` with `WithInternalToken` for consistent composition
- **servicex**: Simplified DI container API
  - `Provide()` and `Resolve()` are now convenience wrappers around `ProvideTyped()`/`ResolveTyped()`
  - Improved error messages and type safety for dependency injection
  - Reduced API surface while maintaining backward compatibility
- **servicex**: Added `RegisterServices()` helper to consolidate common registration patterns
  - Combines `ProvideCommonConstructors()` and `ProvideMany()` into single call
  - Reduces boilerplate in service registration functions
  - Automatically registers logger and database constructors
- **examples/user-service**: Updated to demonstrate standardized patterns
  - Simplified `registerServices()` function using new helpers
  - Cleaner handler registration with direct pattern instead of unused generic helper
  - Reduced main.go from ~180 lines to ~175 lines with improved clarity

### Removed

- **servicex**: Removed unused `RegisterConnectService` generic helper
  - Not compatible with Connect's interface-based handler pattern
  - Direct registration pattern is clearer and more idiomatic
- **servicex**: Removed `CreateOptionalClient` helper (redundant)
  - `RegisterOptionalClients` provides better batch registration
- **examples/user-service**: Removed redundant service fields
  - Removed `expectedInternalToken` field (direct validation no longer used)
  - Removed `greetServiceURL` field (client encapsulates URL)
  - Removed `secureCompare` helper function (no longer needed)

### Fixed

- **examples/connect-tester**: Updated `ValidateInternalToken` test to match new behavior
  - Test now validates client capability instead of token comparison
  - Adjusted expected RPC call count (23 instead of 24)
  - Removed `ValidateInternalToken_Invalid` test (no longer applicable)

## [0.3.3-alpha.1] - 2025-11-06

### Added

- **core/identity**: Added `RequireInternalToken()` function for service-to-service authentication
  - Validates internal token from context against expected token using constant-time comparison
  - Prevents timing attacks with `crypto/subtle.ConstantTimeCompare`
  - Returns `CodeUnauthenticated` error if token is missing or invalid
  - Supports method-level token validation for fine-grained access control
- **configx**: Added `Security.InternalToken` field to `BaseConfig` for internal token configuration
  - Automatically loaded from `INTERNAL_TOKEN` environment variable
  - Integrated into service initialization flow via `servicex`
- **servicex**: Added `App.Config()` method to access configuration struct in registration function
  - Eliminates need to pass configuration as parameter to `registerServices`
  - Simplifies service initialization code
- **servicex**: Added `App.RegisterConnectHandler()` convenience method
  - Automatically injects configured interceptors
  - Reduces boilerplate code for Connect handler registration
  - Logs handler registration for observability
- **servicex**: Added `CallService()` and `CallServiceWithToken()` helper functions for handler simplification
  - Automatically handles Connect request/response conversion (req.Msg → service call → connect.NewResponse)
  - Provides automatic debug logging for all handler methods
  - Reduces handler code from ~10 lines to 1 line per method
  - `CallServiceWithToken()` includes internal token validation for admin operations
- **servicex**: Added `RegisterOptionalClients()` for batch client registration
  - Supports registering multiple optional service clients in one call
  - Automatically extracts URLs from config struct using reflection (via field names)
  - Unified logging and error handling for all clients
  - Eliminates repetitive client creation code
  - Extensible design: easily add new clients by adding entries to the map
- **servicex**: Added `ProvideMany()` helper for bulk constructor registration
  - Registers multiple constructors with unified error handling
  - Stops on first error with descriptive error messages
  - Reduces boilerplate in service registration
- **servicex**: Added `ProvideCommonConstructors()` helper for standard dependencies
  - Automatically registers logger and database constructors
  - Reduces common initialization boilerplate
- **servicex**: Added `ResolveAndRegister()` helper for dependency resolution and handler registration
  - Combines dependency resolution and Connect handler registration in one call
  - Simplifies service initialization flow
- **clientx**: Added internal token support with `WithInternalToken()` option
  - Automatically adds `X-Internal-Token` header to all outgoing requests
  - Integrated into `NewConnectClient()` for seamless service-to-service authentication
  - Configurable header name via `WithInternalTokenHeader()`
- **examples/user-service**: Added comprehensive clientx usage example
  - Demonstrates service-to-service communication pattern
  - Shows client initialization with internal token injection
  - Includes `internal/client` package with production-ready client wrapper
  - Added `GetGreeting()` method demonstrating inter-service calls
- **examples/user-service**: Added `ValidateInternalToken` RPC method
  - Validates internal tokens by calling greet service with provided token
  - Demonstrates service-to-service token validation pattern
  - Supports both valid and invalid token test scenarios
- **examples/connect-tester**: Enhanced test coverage with comprehensive endpoint testing
  - Added multiple ListUsers pagination scenarios (different page sizes, normalization, capping)
  - Added UpdateUser error scenarios (non-existent ID, duplicate email)
  - Added DeleteUser error scenarios (non-existent ID)
  - Added AdminResetAllUsers scenarios (no token, no confirm, with token and confirm)
  - Added ValidateInternalToken tests (valid token, invalid token)
  - Enhanced error scenario testing (empty ID, invalid email format)
  - Reduced redundant test output for cleaner test logs
  - Test coverage increased from 12 to 24 test cases

### Changed

- **servicex**: Simplified `registerServices` function signature
  - No longer requires configuration parameter
  - Configuration accessed via `app.Config()` method
  - Makes main.go initialization code cleaner and more maintainable
- **servicex**: Refactored internal structure following egg standards
  - Moved `handlers.go` and `registration.go` to `internal/` directory
  - Module root now contains only `servicex.go` and `servicex_test.go`
  - All implementation logic properly encapsulated in `internal/` package
- **examples/user-service**: Refactored to use new servicex convenience methods
  - Uses `app.MustDB()` instead of manual nil check
  - Uses `app.RegisterConnectHandler()` for simplified handler registration
  - Uses `RegisterOptionalClients()` for batch client registration
  - Uses `ProvideMany()` for bulk constructor registration
  - Handler methods simplified using `CallService()` and `CallServiceWithToken()`
  - Handler code reduced from ~150 lines to ~35 lines (77% reduction)
  - Improved code organization and readability
- **examples/user-service**: Simplified handler implementations
  - All CRUD methods now use `CallService()` helper (1 line per method)
  - Admin methods use `CallServiceWithToken()` for automatic token validation
  - Eliminated repetitive error handling and response conversion code
- **examples/user-service**: Simplified main.go client registration
  - Replaced manual client creation with `RegisterOptionalClients()`
  - Automatic URL extraction from config (no manual URLGetter functions)
  - Easier to extend with additional clients in the future
- **examples/connect-tester**: Reduced redundant test output
  - Removed verbose success logging for individual test cases
  - Only logs failures and critical test scenarios
  - Test summary still provides complete coverage information
  - Cleaner test output for better readability

### Fixed

- **configx**: Fixed configuration validation not being called after environment variable binding
  - `BindToStruct()` now automatically calls `Validate()` method if configuration struct implements `Validator` interface
  - Allows configuration structs to parse structured data from raw environment variables (e.g., parsing `OSS_BUCKETS="bucket1:region1,bucket2:region2"`)
  - Fixes issue where services using complex configuration parsing (like `eggybyte-oss`) would fail at runtime instead of at configuration load time
  - Configuration validation errors now have clear error messages: `"configuration validation failed: <reason>"`

## [0.3.2] - 2025-11-03

### Fixed

- **Release Scripts**: Fixed module dependency management in release process
  - `release.sh`: Now correctly removes ALL replace directives (including those parsed from go.mod)
  - `release.sh`: Updates ALL egg module dependencies to release version, not just already-released ones
  - `release.sh`: Ensures all modules use consistent version dependencies during release
  
- **Workspace Reinitialization**: Fixed replace directive handling in workspace setup
  - `reinit-workspace.sh`: Fixed `local` keyword error that caused script failure (bash `local` only works in functions)
  - `reinit-workspace.sh`: Added comprehensive validation to ensure relative paths are used (never absolute paths)
  - `reinit-workspace.sh`: Improved error handling and logging for replace directive addition
  - `reinit-workspace.sh`: Now correctly processes all modules including CLI and examples
  - `reinit-workspace.sh`: Added detailed logging showing each replace directive being added

### Changed

- **Release Process**: Improved large file detection
  - Created standalone `scripts/check-large-files.sh` script for unified large file detection
  - `make git-large-files` now uses `--check-only` flag (no interactive prompt, just display results)
  - Release scripts (`release.sh`, `cli-release.sh`) now check only files that will be added by `git add`
  - Uses `--check-staged` flag to detect large files in modified/untracked files (excludes `.gitignore` ignored files)
  - Prevents false positives from build artifacts and ignored files
  - All scripts use unified `logger.sh` for consistent logging output

## [0.3.1] - 2025-10-31

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

## [0.3.0] - 2025-10-31

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
- **Build System**: Improved lint command reliability
  - Makefile lint target now correctly handles empty output scenarios
  - Fixed false-positive failures in multi-module lint runs

### Removed

### Fixed

- **Makefile**: Fixed lint command logic to handle empty grep output correctly
  - Lint now correctly passes when all modules have no errors
  - Fixed false-positive failures caused by grep exit code when no matches found

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
- **examples**: Dedicated build script (`examples/scripts/build-examples.sh`) for building service binaries and Docker images
- **examples**: Comprehensive test script (`examples/scripts/test-examples.sh`) with infrastructure management and health checks
- **examples**: Docker Compose infrastructure setup (MySQL) for local development and testing
- **examples**: Enhanced `connect-tester` with comprehensive metrics endpoint validation (RPC, Runtime, Process, Database metrics)
- **examples**: `examples/bin/` directory for compiled binaries (Git-tracked with `.gitkeep`)

### Changed

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

- **Makefile**: Removed `make build` (not needed for library modules)
- **Makefile**: Removed `make fmt` (redundant with lint)
- **Makefile**: Removed `make vet` (redundant with lint)
- **Makefile**: Removed `make publish-modules` (merged into release)
- **Makefile**: Removed `make security` from quality check (moved to optional/manual execution)
- **Makefile**: Removed all Docker-related targets from root (moved to base-images/)
- **examples/user-service**: Removed ~150 lines of mock repository implementation
- **examples/user-service**: Removed fallback to in-memory storage (production best practice)

### Fixed

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
