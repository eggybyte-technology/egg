# Egg CLI Changelog

All notable changes to the `egg` CLI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [CLI-v0.0.4] - 2025-11-03

### Added

- **CLI**: Automatic API configuration in project initialization
  - `egg init` now automatically creates `api/buf.yaml` and `api/buf.gen.yaml` files
  - No need to run `egg api init` separately for new projects
  - Streamlined project setup workflow

- **CLI**: Automatic gen/go module management
  - CLI automatically adds `gen/go` replace directives to all backend services after API generation
  - Automatic workspace integration: `gen/go` is added to `backend/go.work` after code generation
  - No manual go.mod editing required for gen/go module references
  - Works seamlessly when services reference gen/go before it's generated

- **CLI**: Backend-scoped workspace management
  - Go workspace (`backend/go.work`) automatically manages all backend code
  - `gen/go` module automatically integrated into workspace after API generation
  - Root directory remains language-agnostic (no go.mod at root)
  - Service modules use pattern: `<module_prefix>/backend/<service>`
- **CLI**: Version command and version information display
  - `egg version` - Show full version information (CLI version, commit hash, build time, framework version, Go runtime)
  - `egg --version` / `egg -v` - Show short version format
  - Version information includes: CLI version, git commit hash (short), build timestamp (RFC3339 UTC), egg framework version, Go runtime version and platform
  - Version information is automatically generated during release via `cli-release.sh`
- **CLI**: Enhanced `egg doctor` command with version information
  - Displays CLI version, framework version, git commit, and build time at the start of diagnostics
  - Shows detailed version information for all checked tools:
    - Go version (full version string)
    - Docker version (server version)
    - Docker buildx version (if available)
    - buf version (if available)
    - kubectl version (if available)
    - helm version (if available)
  - Improved tool version detection and display for better diagnostics
- **CLI Release**: Automatic version information generation
  - `cli-release.sh` now automatically generates `internal/version/version.go` with version metadata
  - Version information includes: CLI version, git commit hash, build timestamp (RFC3339 UTC), framework version
  - Version information is committed as part of the release process
- **CLI**: Port proxy management for Docker Compose services
  - `egg compose proxy <service-name> <service-port> [--local-port <port>]` - Create port proxy for a single service
  - `egg compose proxy-all` - Automatically create port proxies for all services (HTTP, Health, Metrics ports)
  - `egg compose proxy-stop` - Stop all running port proxy containers
  - Automatic port availability detection and alternative port finding
  - Uses socat-based containers to map Docker network ports to localhost
  - Supports both backend services (HTTP, Health, Metrics) and frontend services (port 3000)
- **CLI**: Added `cli/cmd/egg/` source files to git repository
  - All CLI command implementations now tracked (api.go, build.go, check.go, compose.go, create.go, doctor.go, init.go, kube.go, main.go)
- **CLI**: Docker Compose service testing via Docker internal network
  - Tests now access services using Docker service names instead of localhost
  - Health checks and RPC tests run inside Docker network using `docker compose exec`
  - Supports pattern matching for metrics endpoint validation
- **CLI**: Enhanced test script modularity
  - Removed redundant tests from integration test suite
  - Streamlined test flow for better maintainability
  - Services remain running after tests for manual inspection
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
- **CLI**: Multi-platform Docker image build support with `docker buildx` (linux/amd64, linux/arm64)
- **CLI**: `buildMultiPlatformImage()` function for automated multi-arch builds with push
- **CLI**: Frontend service name validation enforcing underscore usage (Flutter requirement)
- **CLI**: Automatic service name conversion for Docker images (underscores to hyphens)
- **CLI**: Cross-type service name conflict detection (backend vs frontend)
- **CLI**: Comprehensive service name uniqueness validation
- **CLI**: Local dev version support (v0.0.0-dev) with GOPROXY=direct and GOSUMDB=off
- **CLI**: `GoWithEnv()` method in toolrunner for environment-specific command execution
- Added `--buildx` and `--platforms` flags to build command

### Improved

- **CLI**: Enhanced project initialization workflow
  - `egg init` now includes complete API configuration setup
  - Better guidance in command output for next steps
  - Clearer separation between project initialization and API initialization
- **CLI**: Better separation between minimal services (echo) and full services (crud)
- **CLI**: Docker Compose configuration aligned with servicex environment variable standards
- **CLI**: Default project configuration enables database for easier development setup
- **CLI**: Multi-platform Docker builds with automatic platform detection and buildx support
- **CLI**: Service name validation prevents conflicts across all service types (backend/frontend)
- **CLI**: Flutter web build reliability with correct output path handling
- **CLI**: Docker image naming consistency (hyphens for images, underscores for Flutter packages)
- **CLI**: Better error messages for multi-platform builds without push flag
- **CLI**: Development workflow with v0.0.0-dev versions works in both local and Docker environments

### Changed

- **CLI**: Changed verbose flag from `-v` to `-V` to avoid conflict with version flag
  - `egg -v` now shows version information (was verbose flag)
  - `egg --verbose` or `egg -V` enables verbose output
- **CLI**: Improved `egg doctor` output with detailed version information
  - All checked tools now display their version numbers when available
  - Version information section added at the beginning of diagnostics
  - Better visual hierarchy for version and diagnostic information
- **CLI**: `egg compose up` always uses detached mode (`-d`)
  - Removed `--detached` flag (always runs in background)
  - Automatically generates compose.yaml before starting services
  - Improved network configuration with explicit project name
- **CLI**: Removed `.env` file generation from Docker Compose
  - Environment variables are now configured directly in compose.yaml
  - Cleaner deployment structure without unnecessary files
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
- **CLI**: Flutter web build now correctly copies from `build/web` to output directory
- **CLI**: Backend service creation now uses version-based dependencies (v0.0.0-dev) instead of replace directives for Docker compatibility
- **CLI**: Frontend service names in Docker images use hyphens (e.g., `admin-portal`) while source uses underscores (e.g., `admin_portal`)
- **CLI**: Service creation no longer supports `--force` flag; duplicate names are rejected
- **CLI**: Multi-platform builds automatically use `--push` flag (buildx limitation)
- **CLI**: Build commands provide clear guidance when multi-platform requires push
- CLI `build` command now uses Docker buildx by default

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
- **CLI**: Removed `--force` flag from service creation commands (enforces unique service names)
- **CLI**: Removed replace directives for egg modules in generated go.mod files (use v0.0.0-dev versions)

### Fixed

- **CLI Release**: Fixed replace directive removal in CLI release process
  - `cli-release.sh`: Now removes ALL replace directives, not just framework module ones
  - `cli-release.sh`: Parses go.mod to find and remove all egg module replace directives
  - Ensures clean release artifacts without local development paths
- **CLI**: Fixed Docker Compose network name resolution for port proxies
  - Docker Compose network naming convention: `<project-name>_<network-name>`
  - Port proxies now correctly connect to Docker Compose networks
  - All compose commands now use `-p` flag to specify project name for consistent network naming
- **CLI**: Fixed `cli/cmd/` source files not being tracked in git
  - Removed incorrect `egg` pattern from `cli/.gitignore` that was matching `cli/cmd/egg/` directory
  - All CLI source files now properly tracked in git repository
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
- **CLI**: Fixed Docker Compose attempting to build images instead of using pre-built ones
- **CLI**: Fixed database DSN requirement error for echo/ping services
- **CLI**: Fixed service registration failing when database not configured for echo services
- **CLI**: Fixed frontend build copying assets to wrong location
- **CLI**: Fixed Flutter web build output path (now correctly copies from `build/web/`)
- **CLI**: Fixed Docker builds failing with replace directives by using version-based dependencies
- **CLI**: Fixed service name conflicts not being detected across backend/frontend types
- **CLI**: Fixed frontend service creation allowing invalid hyphenated names (now enforces underscores)
- **CLI**: Fixed Docker image names not following naming conventions (now converts underscores to hyphens)
- **CLI**: Fixed multi-platform builds failing without proper --push handling
