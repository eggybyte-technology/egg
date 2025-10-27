# 🎯 CLI Package

The `cli` package provides a command-line interface for the EggyByte framework.

## Overview

This package offers a comprehensive CLI tool for project management, service generation, and deployment automation. It's designed to streamline the development workflow and provide a consistent experience across different projects.

## Features

- **Project initialization** - Create new EggyByte projects
- **Service generation** - Generate backend and frontend services
- **API management** - Protobuf API definition and generation
- **Docker Compose** - Generate and manage Docker Compose configurations
- **Kubernetes** - Generate Kubernetes manifests
- **Health checks** - Project validation and health checks

## Quick Start

```bash
# Build the CLI
make build-cli

# Initialize a new project
./cli/egg init --project-name my-project --module-prefix github.com/myorg/my-project

# Create a backend service
./cli/egg create backend user-service

# Generate Docker Compose configuration
./cli/egg compose generate

# Check project health
./cli/egg check
```

## Commands

### Project Management

#### `init` - Initialize a new project

```bash
./cli/egg init [flags]
```

**Flags:**
- `--project-name` - Project name (required)
- `--module-prefix` - Go module prefix (required)
- `--docker-registry` - Docker registry URL
- `--version` - Project version

**Example:**
```bash
./cli/egg init \
  --project-name user-service \
  --module-prefix github.com/myorg/user-service \
  --docker-registry ghcr.io/myorg \
  --version v1.0.0
```

#### `check` - Check project health

```bash
./cli/egg check
```

Validates the project structure and configuration.

### Service Generation

#### `create backend` - Create a backend service

```bash
./cli/egg create backend <service-name> [flags]
```

**Flags:**
- `--local-modules` - Use local modules instead of remote dependencies

**Example:**
```bash
./cli/egg create backend user-service --local-modules
```

#### `create frontend` - Create a frontend service

```bash
./cli/egg create frontend <service-name> [flags]
```

**Flags:**
- `--platforms` - Target platforms (web, mobile, desktop)

**Example:**
```bash
./cli/egg create frontend admin-portal --platforms web
```

### API Management

#### `api init` - Initialize API definitions

```bash
./cli/egg api init
```

Creates the API directory structure with Buf configuration.

#### `api generate` - Generate code from protobuf

```bash
./cli/egg api generate
```

Generates Go code from protobuf definitions.

### Docker Compose

#### `compose generate` - Generate Docker Compose configuration

```bash
./cli/egg compose generate
```

Generates `docker-compose.yaml` based on project configuration.

#### `compose up` - Start services

```bash
./cli/egg compose up [flags]
```

**Flags:**
- `--detached` - Run in detached mode
- `--build` - Build images before starting

**Example:**
```bash
./cli/egg compose up --detached --build
```

#### `compose down` - Stop services

```bash
./cli/egg compose down
```

### Kubernetes

#### `kube generate` - Generate Kubernetes manifests

```bash
./cli/egg kube generate
```

Generates Kubernetes manifests for deployment.

### Health and Diagnostics

#### `doctor` - Check development environment

```bash
./cli/egg doctor
```

Checks the development environment and dependencies.

## Configuration

### Project Configuration (`egg.yaml`)

```yaml
project_name: "user-service"
module_prefix: "github.com/myorg/user-service"
docker_registry: "ghcr.io/myorg"
version: "v1.0.0"

backend:
  user-service:
    name: "user-service"
    version: "v1.0.0"
    ports:
      http: 8080
      health: 8081
      metrics: 9091

frontend:
  admin-portal:
    name: "admin-portal"
    version: "v1.0.0"
    platforms: ["web"]

api:
  enabled: true
  version: "v1"
```

### Environment Variables

```bash
# CLI configuration
EGG_CONFIG_FILE=egg.yaml
EGG_LOG_LEVEL=info
EGG_OUTPUT_FORMAT=text

# Docker configuration
DOCKER_REGISTRY=ghcr.io/myorg
DOCKER_TAG=latest

# Kubernetes configuration
KUBECONFIG=/path/to/kubeconfig
KUBERNETES_NAMESPACE=default
```

## Usage Examples

### Complete Project Setup

```bash
# 1. Initialize project
./cli/egg init \
  --project-name ecommerce \
  --module-prefix github.com/myorg/ecommerce \
  --docker-registry ghcr.io/myorg \
  --version v1.0.0

# 2. Create backend services
./cli/egg create backend user-service --local-modules
./cli/egg create backend product-service --local-modules
./cli/egg create backend order-service --local-modules

# 3. Create frontend
./cli/egg create frontend web-portal --platforms web

# 4. Initialize API
./cli/egg api init

# 5. Generate Docker Compose
./cli/egg compose generate

# 6. Check project health
./cli/egg check
```

### Service Development Workflow

```bash
# 1. Create new service
./cli/egg create backend payment-service --local-modules

# 2. Add API definitions
# Edit api/payment/v1/payment.proto

# 3. Generate code
./cli/egg api generate

# 4. Update Docker Compose
./cli/egg compose generate

# 5. Build and test
./scripts/build.sh service examples/payment-service payment-service
./scripts/test.sh examples
```

### Deployment Workflow

```bash
# 1. Generate Kubernetes manifests
./cli/egg kube generate

# 2. Build all services
./scripts/build.sh all

# 3. Deploy with Docker Compose
./cli/egg compose up --detached --build

# 4. Check service health
./scripts/deploy.sh health
```

## Project Structure

After running `egg init`, the following structure is created:

```
project/
├── egg.yaml                 # Project configuration
├── .gitignore              # Git ignore file
├── api/                    # API definitions
│   ├── buf.yaml           # Buf configuration
│   ├── buf.gen.yaml       # Code generation config
│   └── <service>/         # Service API definitions
├── backend/               # Backend services
│   ├── go.work            # Go workspace
│   └── <service>/         # Service implementations
├── frontend/               # Frontend services
│   └── <service>/         # Service implementations
├── build/                 # Build configurations
│   ├── Dockerfile.backend
│   ├── Dockerfile.frontend
│   └── Dockerfile.eggybyte-go-alpine
├── deploy/                # Deployment configurations
│   ├── docker-compose.yaml
│   ├── otel-collector-config.yaml
│   └── k8s/               # Kubernetes manifests
└── scripts/               # Build and deployment scripts
    ├── build.sh
    ├── deploy.sh
    └── test.sh
```

## Backend Service Structure

When creating a backend service, the following structure is generated:

```
backend/<service-name>/
├── go.mod                  # Go module file
├── go.sum                  # Go dependencies
├── cmd/                    # Application entrypoints
│   └── server/
│       └── main.go        # Main application
├── internal/               # Internal packages
│   ├── config/            # Configuration
│   │   └── app_config.go
│   ├── handler/           # HTTP handlers
│   │   └── handler.go
│   ├── service/           # Business logic
│   │   └── service.go
│   └── repository/        # Data access
│       └── repository.go
└── gen/                   # Generated code
    └── go/
        └── <service>/
            └── v1/
                ├── <service>.pb.go
                └── <service>v1connect/
                    └── <service>.connect.go
```

## Frontend Service Structure

When creating a frontend service, the following structure is generated:

```
frontend/<service-name>/
├── pubspec.yaml           # Dart dependencies
├── lib/                   # Dart source code
│   └── main.dart         # Main application
├── web/                   # Web-specific files
│   ├── index.html        # HTML entry point
│   └── manifest.json     # Web app manifest
└── test/                  # Tests
    └── widget_test.dart
```

## API Definition Structure

When initializing API definitions, the following structure is created:

```
api/
├── buf.yaml              # Buf configuration
├── buf.gen.yaml          # Code generation config
└── <service>/            # Service API definitions
    └── v1/
        └── <service>.proto
```

## Testing

```bash
# Test CLI functionality
./scripts/test.sh cli

# Test production workflows
./scripts/test.sh production

# Test example services
./scripts/test.sh examples
```

## Best Practices

### 1. Use Consistent Naming

```bash
# Good: Consistent naming
./cli/egg create backend user-service
./cli/egg create backend product-service
./cli/egg create backend order-service

# Avoid: Inconsistent naming
./cli/egg create backend userService
./cli/egg create backend product_service
./cli/egg create backend OrderService
```

### 2. Version Management

```bash
# Use semantic versioning
./cli/egg init --version v1.0.0
./cli/egg init --version v1.1.0
./cli/egg init --version v2.0.0
```

### 3. Module Organization

```bash
# Use consistent module prefixes
./cli/egg init --module-prefix github.com/myorg/user-service
./cli/egg init --module-prefix github.com/myorg/product-service
./cli/egg init --module-prefix github.com/myorg/order-service
```

### 4. Docker Registry

```bash
# Use consistent Docker registry
./cli/egg init --docker-registry ghcr.io/myorg
./cli/egg init --docker-registry docker.io/myorg
./cli/egg init --docker-registry registry.example.com/myorg
```

## Troubleshooting

### Common Issues

#### 1. Go Module Issues

```bash
# Error: module not found
# Solution: Use --local-modules flag
./cli/egg create backend user-service --local-modules
```

#### 2. Docker Build Issues

```bash
# Error: Docker build fails
# Solution: Check Docker daemon and build context
docker system prune
./scripts/build.sh clean
./scripts/build.sh all
```

#### 3. Configuration Issues

```bash
# Error: Invalid configuration
# Solution: Validate configuration
./cli/egg check
```

#### 4. Port Conflicts

```bash
# Error: Port already in use
# Solution: Check port usage and update configuration
lsof -i :8080
./cli/egg compose down
./cli/egg compose up
```

### Debug Mode

```bash
# Enable debug logging
EGG_LOG_LEVEL=debug ./cli/egg <command>

# Verbose output
./cli/egg <command> --verbose
```

## Contributing

Contributions are welcome! Please see the main project [Contributing Guide](../CONTRIBUTING.md) for details.

### Development Setup

```bash
# Clone repository
git clone https://go.eggybyte.com/egg.git
cd egg

# Build CLI
make build-cli

# Run tests
./scripts/test.sh cli
```

### Adding New Commands

1. Create command file in `cli/cmd/egg/`
2. Implement command logic
3. Add tests
4. Update documentation

## License

This package is part of the EggyByte framework and is licensed under the MIT License.