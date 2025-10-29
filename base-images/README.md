# Egg Framework Base Images

This directory contains Dockerfiles and build configurations for the Egg framework's foundation images.

## Overview

The base images provide standardized build and runtime environments for all Egg-based services. They ensure consistent Go versions, Alpine Linux versions, and optimized image sizes across the ecosystem.

## Images

### 1. Builder Image (`eggybyte-go-builder`)

**Purpose**: Compilation environment for building Go applications

**Base**: `golang:1.25.1-alpine3.22` (configurable)

**Features**:
- Full Go toolchain for compilation
- Build essentials (gcc, musl-dev, etc.)
- Common build tools
- Optimized for fast builds

**Usage**: Multi-stage Dockerfile build stage
```dockerfile
FROM ghcr.io/eggybyte-technology/eggybyte-go-builder:go1.25.1-alpine3.22 AS builder
WORKDIR /build
COPY . .
RUN go build -o app ./cmd/server
```

### 2. Runtime Image (`eggybyte-go-alpine`)

**Purpose**: Minimal runtime environment for running Go binaries

**Base**: `alpine:3.22` (configurable)

**Features**:
- Minimal Alpine Linux
- CA certificates for HTTPS
- Timezone data
- Non-root user for security
- Extremely small image size (~10MB)

**Usage**: Multi-stage Dockerfile runtime stage
```dockerfile
FROM ghcr.io/eggybyte-technology/eggybyte-go-alpine:go1.25.1-alpine3.22
COPY --from=builder /build/app /app
ENTRYPOINT ["/app"]
```

## Building Images

### Quick Start

```bash
# Build both images locally
make build-all

# Build and push to registry (multi-arch)
make build-all PUSH=true

# View all options
make help
```

### Configuration

Customize build via environment variables:

| Variable          | Default                              | Description                     |
|-------------------|--------------------------------------|---------------------------------|
| `DOCKER_REGISTRY` | `ghcr.io/eggybyte-technology`        | Container registry              |
| `GO_VERSION`      | `1.25.1`                             | Go version                      |
| `ALPINE_VERSION`  | `3.22`                               | Alpine Linux version            |
| `DOCKER_PLATFORM` | `linux/amd64,linux/arm64`            | Target architectures            |
| `PUSH`            | `false`                              | Push to registry after build    |

### Examples

```bash
# Build with specific Go version
make build-builder GO_VERSION=1.25.2

# Build for single platform
make build-runtime DOCKER_PLATFORM=linux/amd64

# Build and push to custom registry
make push-all DOCKER_REGISTRY=myregistry.io/myorg
```

## Multi-Architecture Support

The images support multiple architectures via Docker Buildx:

- `linux/amd64` (x86_64)
- `linux/arm64` (aarch64)

**Note**: Multi-platform builds require `PUSH=true` to push directly to registry. Local builds default to `linux/amd64` for compatibility.

## Image Tags

Each image has two tags:

1. **Versioned tag**: `go{GO_VERSION}-alpine{ALPINE_VERSION}`
   - Example: `go1.25.1-alpine3.22`
   - Use for reproducible builds

2. **Latest tag**: `latest`
   - Always points to the most recent build
   - Use for development only

## Maintenance

### Updating Go Version

1. Update `GO_VERSION` in Makefile or pass as parameter
2. Rebuild images: `make build-all PUSH=true`
3. Update service Dockerfiles to reference new tag

### Updating Alpine Version

1. Update `ALPINE_VERSION` in Makefile
2. Test builds locally: `make build-all`
3. Push to registry: `make push-all`

## Security

- **Non-root user**: Runtime image runs as user `appuser` (UID 1000)
- **Minimal attack surface**: Runtime contains only essential binaries
- **Regular updates**: Base images should be rebuilt monthly for security patches

## Troubleshooting

### Build fails with "multiple platforms not supported"

**Issue**: Local builds don't support multi-platform by default

**Solution**: Either:
- Use `PUSH=true` to build and push
- Set `DOCKER_PLATFORM=linux/amd64` for single platform

### Permission denied when running container

**Issue**: Binary not executable or wrong user

**Solution**: Ensure binary has execute permissions in builder stage:
```dockerfile
RUN chmod +x /build/app
```

## Related Documentation

- [Dockerfile Best Practices](../docs/dockerfile-rules.md)
- [Examples](../examples/README.md)
- [Egg Framework Documentation](../README.md)

## License

This package is part of the EggyByte framework and is licensed under the MIT License.
See the root LICENSE file for details.

