# Egg CLI

<div align="center">

**A Connect-first, Kubernetes-native modern project management tool for EggyByte platform**

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)](https://github.com/eggybyte-technology/egg)

</div>

## ✨ Features

- 🚀 **Project Initialization** - Quickly create project skeleton and configuration
- 🏗️ **Service Generation** - Auto-generate backend (Go) and frontend (Flutter) services
- 📡 **API Management** - Use buf to manage protobuf definitions, generate multi-language code (Go, Dart, TypeScript, OpenAPI)
- 🐳 **Local Development** - Docker Compose integration with one-click service and database startup
- ☸️ **Kubernetes Deployment** - Auto-generate Helm charts and deploy to Kubernetes clusters
- 🏭 **Image Building** - Multi-platform Docker image building and pushing
- ✅ **Configuration Validation** - Comprehensive project structure and configuration validation

## 📦 Installation

### From Source

```bash
git clone https://github.com/eggybyte-technology/egg.git
cd egg/cli
go build -o egg ./cmd/main.go
sudo mv egg /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/eggybyte-technology/egg/cli/cmd@latest
```

## 🚀 Quick Start

### 1. Initialize Project

```bash
mkdir my-platform
cd my-platform
egg init --project-name my-platform
```

### 2. Create Backend Service

```bash
egg create backend user-service
```

### 3. Initialize API Definitions

```bash
egg api init
```

Add your `.proto` files in the `api/` directory, then generate code:

```bash
egg api generate
```

### 4. Local Development

Start all services:

```bash
egg compose up
```

Stop services:

```bash
egg compose down
```

View logs:

```bash
egg compose logs --service user-service
```

### 5. Deploy to Kubernetes

Generate Helm charts:

```bash
egg kube template -n production
```

Apply to cluster:

```bash
egg kube apply -n production
```

### 6. Build and Push Images

Build images:

```bash
egg build --version v1.0.0
```

Build and push:

```bash
egg build --version v1.0.0 --push
```

Build specific services only:

```bash
egg build --subset user-service,admin-portal
```

### 7. Check Project

```bash
egg check
```

## 📖 Command Reference

### Global Options

- `--verbose, -v` - Enable verbose output
- `--non-interactive` - Disable interactive prompts
- `--json` - Output in JSON format

### `init` - Initialize Project

```bash
egg init [flags]
```

**Options:**
- `--project-name` - Project name (default: current directory name)
- `--module-prefix` - Go module prefix
- `--docker-registry` - Docker image registry address
- `--version` - Project version

### `create` - Create Services

#### Create Backend Service

```bash
egg create backend <name>
```

**Generates:**
- Go module and workspace configuration
- Connect-only service structure
- Configuration management
- Health check and metrics endpoints

#### Create Frontend Service

```bash
egg create frontend <name> [flags]
```

**Options:**
- `--platforms` - Target platforms (web, android, ios)

### `api` - API Management

#### Initialize API Definitions

```bash
egg api init
```

#### Generate Code

```bash
egg api generate
```

**Generates:**
- Go code (protobuf + Connect)
- Dart code (Flutter)
- TypeScript type definitions
- OpenAPI specifications

### `compose` - Docker Compose Management

#### Start Services

```bash
egg compose up [flags]
```

**Options:**
- `--detached` - Run in background

#### Stop Services

```bash
egg compose down
```

#### View Logs

```bash
egg compose logs [flags]
```

**Options:**
- `--service` - Filter specific service
- `--follow, -f` - Follow log output

### `kube` - Kubernetes Deployment

#### Generate Helm Templates

```bash
egg kube template [flags]
```

**Options:**
- `--namespace, -n` - Kubernetes namespace

#### Apply to Cluster

```bash
egg kube apply [flags]
```

**Options:**
- `--namespace, -n` - Kubernetes namespace

#### Uninstall

```bash
egg kube uninstall [flags]
```

**Options:**
- `--namespace, -n` - Kubernetes namespace

### `build` - Build Images

```bash
egg build [flags]
```

**Options:**
- `--push` - Push to image registry
- `--version` - Image version tag
- `--subset` - Build only specified services (comma-separated)

### `check` - Check Project

```bash
egg check
```

**Checks:**
- Project structure integrity
- Configuration validity
- Port conflicts
- Expression references
- Service configuration

## ⚙️ Configuration File (egg.yaml)

The project configuration file `egg.yaml` is the single source of truth for the project, containing:

- Project metadata
- Build configuration
- Environment variables
- Service definitions
- Kubernetes resources
- Database configuration

See [egg-cli.md](../docs/egg-cli.md) for example configuration.

## 🔗 Expression System

Egg supports configuration reference expressions:

- `${cfg:resource}` - Inject ConfigMap name (Kubernetes)
- `${cfgv:resource:key}` - Inject ConfigMap value (Compose)
- `${sec:resource:key}` - Inject Secret reference (Kubernetes)
- `${svc:name@type}` - Inject service reference (type: clusterip or headless)

## 🏗️ Architecture Design

Egg CLI follows these design principles:

1. **Connect-first** - Default to Connect protocol, expose only HTTP port
2. **Kubernetes-native** - Native support for K8s ConfigMap, Secret, and service discovery
3. **CLI-driven** - All operations through CLI commands, no manual editing of generated files
4. **Configuration as Code** - Single configuration file manages entire project
5. **Production-grade Quality** - Complete documentation, error handling, and test coverage

## 🛠️ Development

### Run Tests

```bash
go test ./...
```

### Build

```bash
go build -o egg ./cmd/main.go
```

### Contributing

Please refer to [CONTRIBUTING.md](../../CONTRIBUTING.md).

## 📄 License

See [LICENSE](../../LICENSE).

---

<div align="center">

**Built with ❤️ by EggyByte Technology**

</div>

