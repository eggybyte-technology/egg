# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

### Security

## [0.0.2] - 2025-10-20

### Fixed
- Added go.sum files for all modules to ensure proper dependency checksums
- Go modules now properly resolve with Go proxy and checksum database
- Improved module dependency graph consistency

## [0.0.1] - 2025-10-20

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

## [0.1.0] - 2025-10-17

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
