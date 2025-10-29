# 🎯 Egg CLI

Command-line interface for the EggyByte Connect-first framework, providing project scaffolding, code generation, and deployment automation.

## Overview

The Egg CLI is a comprehensive tool for managing EggyByte projects from initialization to deployment. It automates project setup, service generation, API code generation with **local protoc plugins**, Docker Compose configuration, and Kubernetes Helm chart generation.

## Key Features

- **Project initialization** - Scaffold new projects with proper structure
- **Service generation** - Create backend (Go) and frontend (Flutter) services
- **Local API code generation** - Use local protoc plugins verified by `egg doctor`
- **Unified Helm charts** - Generate project-level Kubernetes deployment charts
- **Docker Compose** - Automatically generate service orchestration configs
- **Containerized builds** - Build binaries and images using foundation images
- **Health checks** - Validate project configuration and structure
- **Backend-scoped workspace** - Manage Go workspaces automatically
- **Automatic gen/go module management** - CLI handles gen/go replace directives

## Installation

```bash
# Build from source
make build-cli

# Install globally (optional)
sudo cp cli/bin/egg /usr/local/bin/egg
```

## Quick Start

```bash
# 1. Check development environment
egg doctor --install

# 2. Initialize a new project
egg init --project-name myapp --module-prefix github.com/myorg/myapp

# 3. Create backend services
egg create backend user --proto crud --local-modules
egg create backend ping --proto echo --local-modules

# 4. Create frontend
egg create frontend web --platforms web

# 5. Initialize API definitions
egg api init

# 6. Generate code (using local protoc plugins)
egg api generate

# 7. Build services (requires foundation images)
egg build backend user    # Build backend service
egg build frontend web     # Build frontend service
egg build all              # Build all services

# 8. Generate Docker Compose
egg compose generate

# 9. Generate Helm charts
egg kube generate

# 10. Check project health
egg check
```

## Commands

### Environment Management

#### `egg doctor` - Environment diagnostics

Verifies development environment and required tools, including local protoc plugins.

```bash
egg doctor                    # Check environment
egg doctor --install          # Install missing protoc plugins
```

**Checks:**
- Go, Docker, buf, kubectl, helm installations
- Local protoc plugins (protoc-gen-go, protoc-gen-connect-go, protoc-gen-openapiv2, protoc-gen-dart)
- Network connectivity
- File system permissions

**Install plugins:**
```bash
egg doctor --install

# Installs:
# - protoc-gen-go (v1.34.2)
# - protoc-gen-connect-go (v1.16.0)
# - protoc-gen-openapiv2 (v2.24.0)
# - protoc-gen-dart (via dart pub global activate)
```

### Project Management

#### `egg init` - Initialize a new project

Scaffolds a complete project structure with proper directory layout.

```bash
egg init --project-name <name> --module-prefix <prefix> [flags]
```

**Flags:**
- `--project-name` - Project name (default: current directory name)
- `--module-prefix` - Go module prefix (required)
- `--docker-registry` - Docker registry URL (default: ghcr.io/eggybyte-technology)
- `--version` - Project version (default: v1.0.0)

**Example:**
```bash
egg init \
  --project-name ecommerce \
  --module-prefix github.com/myorg/ecommerce \
  --docker-registry ghcr.io/myorg \
  --version v1.0.0
```

#### `egg check` - Validate project configuration

Checks project structure, configuration files, and dependencies.

```bash
egg check
```

### Service Generation

#### `egg create backend` - Create a backend service

Generates a complete Go service with handler, service, repository, and model layers.

```bash
egg create backend <service-name> [flags]
```

**Flags:**
- `--proto` - Proto template type: `echo`, `crud`, or `none` (default: echo)
- `--local-modules` - Use local egg modules (required for development)
- `--force` - Overwrite existing service

**Example:**
```bash
egg create backend user --proto crud --local-modules
egg create backend ping --proto echo --local-modules
```

**Generated structure:**
```
backend/user/
├── go.mod                      # With local egg module replace directives
├── Makefile
├── cmd/server/main.go
├── internal/
│   ├── config/app_config.go
│   ├── handler/handler.go
│   ├── service/service.go
│   ├── repository/repository.go
│   └── model/
│       ├── model.go
│       └── errors.go
```

**Automatic gen/go handling:**
- If `gen/go` exists during service creation, CLI automatically adds replace directive
- After `egg api generate`, all backend services are updated with gen/go replace directive
- No manual go.mod editing required

#### `egg create frontend` - Create a frontend service

Generates a Flutter application with web support.

```bash
egg create frontend <service-name> [flags]
```

**Flags:**
- `--platforms` - Target platforms: `web`, `android`, `ios`, `macos`, `windows`, `linux`

**Example:**
```bash
egg create frontend admin-portal --platforms web
```

### API Management

#### `egg api init` - Initialize API definitions

Creates API directory structure with Buf configuration using local protoc plugins.

```bash
egg api init
```

**Generated files:**
- `api/buf.yaml` - Buf schema configuration
- `api/buf.gen.yaml` - Code generation config (uses local plugins)

#### `egg api generate` - Generate code from protobuf

Generates Go, Dart, and OpenAPI code using **local protoc plugins** (no remote dependencies).

```bash
egg api generate
```

**Workflow:**
1. Run `buf generate` with local plugins
2. Initialize `gen/go/go.mod` module
3. Add `gen/go` to `backend/go.work`
4. Update all backend services with gen/go replace directive
5. Run `go mod tidy` on affected modules

**Generated code:**
- `gen/go/` - Go client/server code (via protoc-gen-go, protoc-gen-connect-go)
- `gen/dart/` - Dart client code (via protoc-gen-dart)
- `gen/openapi/` - OpenAPI docs (via protoc-gen-openapiv2)

**Local plugin configuration:**
```yaml
# api/buf.gen.yaml
version: v2

plugins:
  # Go (protobuf + Connect)
  - local: protoc-gen-go
    out: ../gen/go
    opt:
      - paths=source_relative
  - local: protoc-gen-connect-go
    out: ../gen/go
    opt:
      - paths=source_relative

  # Dart (protobuf + gRPC)
  - local: protoc-gen-dart
    out: ../gen/dart
    opt:
      - grpc

  # OpenAPI v2
  - local: protoc-gen-openapiv2
    out: ../gen/openapi
```

### Build Management

#### `egg build` - Build Docker images

Build backend services and frontend applications using containerized build environments.

```bash
egg build backend <service>       # Build backend service (multi-arch: amd64, arm64)
egg build frontend <service>      # Build frontend service
egg build all                     # Build all services
```

**Backend Build Process:**

1. Compile binary using builder container (multi-arch support)
2. Package binary into runtime image (multi-arch support)

```bash
# Build backend service (default: linux/amd64,linux/arm64)
egg build backend user

# Custom tag and platform
egg build backend user --tag v1.0.0 --platform linux/amd64

# Build and push to registry
egg build backend user --push
```

**Frontend Build Process:**

1. Build Flutter web assets using local Flutter SDK
2. Package assets into nginx image

```bash
# Build frontend service
egg build frontend admin_portal

# Custom tag
egg build frontend admin_portal --tag v1.0.0

# Build and push to registry
egg build frontend admin_portal --push
```

**Build All Services:**

```bash
# Build all backend and frontend services
egg build all

# Build and push all (multi-arch for backend)
egg build all --push
```

**Output Structure:**

```
bin/
├── backend/
│   ├── user/
│   │   └── server         # Compiled Go binary (multi-arch)
│   └── order/
│       └── server
└── frontend/
    ├── admin_portal/      # Flutter web build output
    │   ├── index.html
    │   ├── main.dart.js
    │   └── ...
    └── user_app/
        └── ...
```

**Important:** 
- Backend services are built with multi-architecture support (linux/amd64, linux/arm64) by default
- Local plugins must be installed via `egg doctor --install` before code generation

### Docker Compose

#### `egg compose generate` - Generate Docker Compose configuration

Creates `deploy/compose/compose.yaml` and `.env` files.

```bash
egg compose generate
```

**Output:**
```
deploy/compose/
├── compose.yaml  # Service definitions
└── .env          # Environment variables
```

#### `egg compose up` - Start services

```bash
egg compose up [flags]
```

**Flags:**
- `--detached` - Run in background
- `--build` - Build images before starting

#### `egg compose down` - Stop services

```bash
egg compose down
```

### Kubernetes

#### `egg kube generate` - Generate unified Helm chart

Creates a single project-level Helm chart containing all services.

```bash
egg kube generate
```

**Output:**
```
deploy/helm/<project-name>/
├── Chart.yaml
├── values.yaml
└── templates/
    ├── _helpers.tpl
    ├── backend-deployment.yaml
    ├── backend-service.yaml
    ├── frontend-deployment.yaml
    ├── frontend-service.yaml
    ├── configmaps.yaml
    └── secrets.yaml
```

**Features:**
- Unified project-level chart (not individual service charts)
- Dynamic template rendering based on `egg.yaml` configuration
- Kubernetes-compliant naming (hyphens instead of underscores)

#### `egg kube template` - Render Helm templates

```bash
egg kube template -n <namespace>
```

#### `egg kube apply` - Deploy to cluster

```bash
egg kube apply -n <namespace>
```

#### `egg kube uninstall` - Remove from cluster

```bash
egg kube uninstall -n <namespace>
```

### Build Commands

#### `egg build backend` - Build backend services

```bash
egg build backend <service-name>  # Build single service
egg build backend --all            # Build all services
```

#### `egg build docker` - Build Docker images

```bash
egg build docker <service-name>
```

## Configuration

### Project Configuration (`egg.yaml`)

```yaml
project_name: "ecommerce"
module_prefix: "github.com/myorg/ecommerce"
docker_registry: "ghcr.io/myorg"
version: "v1.0.0"

backend:
  user:
    name: "user"
    ports:
      http: 8080
      health: 8081
      metrics: 9091

frontend:
  admin-portal:
    name: "admin-portal"
    platforms: ["web"]

database:
  enabled: false
  dsn: "mysql://user:pass@tidb.example.com:4000/db"

infrastructure:
  observability:
    enabled: true
  tracing:
    enabled: true
```

## Project Structure

After initialization:

```
project/
├── egg.yaml                    # Project configuration
├── .gitignore
├── api/                        # API definitions
│   ├── buf.yaml
│   ├── buf.gen.yaml           # Local plugin config
│   └── <service>/v1/          # Proto files
├── backend/                   # Backend services
│   ├── go.work                # Go workspace (includes gen/go)
│   └── <service>/            # Service implementations
│       ├── go.mod             # With local replaces
│       └── ...
├── frontend/                  # Frontend services
│   └── <service>/            # Flutter apps
├── gen/                       # Generated code
│   ├── go/                   # Go client/server
│   │   └── go.mod            # Independent module
│   ├── dart/                 # Dart client
│   └── openapi/              # OpenAPI docs
├── build/                     # Build configs
│   ├── Dockerfile.backend
│   ├── Dockerfile.frontend
│   └── nginx.conf
└── deploy/                    # Deployment configs
    ├── compose/               # Docker Compose
    │   ├── compose.yaml
    │   └── .env
    └── helm/                  # Helm charts
        └── <project-name>/   # Unified chart
            ├── Chart.yaml
            ├── values.yaml
            └── templates/
```

## Architecture Highlights

### Backend-Scoped Workspace

- `backend/go.work` manages all Go code in the project
- `gen/go` is automatically added to the workspace after API generation
- Root directory remains language-agnostic (no go.mod)
- Service modules: `<module_prefix>/backend/<service>`
- Generated code module: `<module_prefix>/gen/go`

### Unified Helm Chart

- Single project-level chart (`deploy/helm/<project-name>/`)
- Contains templates for all backend and frontend services
- Service-specific configuration in `values.yaml`
- Dynamic template rendering with project-specific helpers

### Local Plugin Support

- Uses local protoc plugins (verified by `egg doctor`)
- Offline-first development (no remote buf.build dependencies)
- Faster code generation (no network latency)
- Required plugins installed via `egg doctor --install`

**Plugin Installation:**
```bash
# Check environment
egg doctor

# Install missing plugins
egg doctor --install

# Plugins are installed to:
# - Go plugins: ~/go/bin/
# - Dart plugin: via dart pub global activate
```

### Automatic gen/go Module Management

**Problem:** When services reference `gen/go` before it's generated, Go tries to fetch from remote (non-existent repository).

**Solution:** CLI automatically manages gen/go module references:

1. **During service creation:** If `gen/go` exists, add replace directive immediately
2. **After API generation:** Update all existing backend services with gen/go replace directive
3. **Automatic workspace integration:** Add `gen/go` to `backend/go.work`

**No manual go.mod editing required!**

Example workflow:
```bash
# 1. Create services (gen/go doesn't exist yet)
egg create backend user --proto crud --local-modules
egg create backend ping --proto crud --local-modules

# 2. Generate code (gen/go is created)
egg api generate

# CLI automatically:
# - Creates gen/go/go.mod
# - Adds gen/go to backend/go.work
# - Updates user/go.mod with replace directive
# - Updates ping/go.mod with replace directive

# 3. Services can now import gen/go code
cd backend/user && go build ./cmd/server
```

### Docker Compose Integration

- Automatic service discovery from egg.yaml
- Environment variable injection
- Runtime image support (`eggybyte-go-alpine`)
- Output to `deploy/compose/` directory

## Usage Examples

### Complete Workflow

```bash
# 1. Check environment
egg doctor --install

# 2. Initialize project
egg init --project-name shop --module-prefix github.com/myorg/shop

# 3. Create services
egg create backend product --proto crud --local-modules
egg create backend cart --proto crud --local-modules
egg create frontend web --platforms web

# 4. Define APIs
# Edit api/product/v1/product.proto
# Edit api/cart/v1/cart.proto

# 5. Generate code (automatically handles gen/go)
egg api generate

# 6. Generate deployments
egg compose generate
egg kube generate

# 7. Build and run
egg build backend --all
egg compose up
```

### Development Workflow

```bash
# Add new service
egg create backend inventory --proto crud --local-modules

# Update API
# Edit api/inventory/v1/inventory.proto

# Regenerate code (updates all services with gen/go)
egg api generate

# Update deployments
egg compose generate
egg kube generate

# Test locally
egg compose up
```

## Best Practices

### 1. Service Naming

- Use lowercase with hyphens: `user-service`, `order-service`
- No `-service` suffix (CLI validates this)
- Consistent patterns across services

### 2. Proto Templates

- Use `echo` for simple echo/ping services
- Use `crud` for services with Create/Read/Update/Delete operations
- Use `none` to skip proto generation

### 3. Local Development

- Always use `--local-modules` flag for backend services
- Ensures compatibility with local egg framework modules
- CLI automatically manages gen/go module references

### 4. Version Management

- Use semantic versioning: `v1.0.0`, `v1.1.0`, `v2.0.0`
- Update version in egg.yaml for releases

### 5. Plugin Management

- Run `egg doctor` before starting development
- Use `egg doctor --install` to set up local plugins
- No need to manually manage plugin versions

## Troubleshooting

### API Generation Fails

```bash
# Check protoc plugins are installed
egg doctor

# Install missing plugins
egg doctor --install

# Verify plugins are in PATH
which protoc-gen-go
which protoc-gen-connect-go
which protoc-gen-openapiv2
```

### Go Module Issues

```bash
# If services can't find gen/go:
# 1. Run egg api generate to create gen/go module
# 2. CLI automatically adds replace directives
# 3. Run go mod tidy in service directory

cd backend/user
go mod tidy
```

### Docker Build Fails

```bash
# Pull runtime image
docker pull ghcr.io/eggybyte-technology/eggybyte-go-alpine:latest

# Clean Docker cache
docker system prune
```

### Helm Chart Issues

```bash
# Lint chart
helm lint deploy/helm/<project-name>

# Dry-run template
helm template <release-name> deploy/helm/<project-name>
```

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

### Development Setup

```bash
# Clone repository
git clone https://go.eggybyte.com/egg.git
cd egg

# Build CLI
make build-cli

# Run tests (rebuilds CLI automatically)
./scripts/test-cli.sh
```

## License

MIT License - see root LICENSE file for details.
