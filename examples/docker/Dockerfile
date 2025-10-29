# ==============================================================================
# EggyByte Example Services Image
# ==============================================================================
#
# Purpose:
#   Package pre-compiled example service binaries into minimal runtime containers.
#   Used by examples/scripts/build-examples.sh for building demo services.
#
# Base Image:
#   ghcr.io/eggybyte-technology/eggybyte-go-alpine:go1.25.1-alpine3.22
#   (minimal Alpine runtime with CA certificates and timezone data)
#
# Build Process:
#   1. Compile binary locally: CGO_ENABLED=0 go build -o examples/bin/<service>
#   2. Build image: docker build -f examples/docker/Dockerfile --build-arg SERVICE_NAME=<service> .
#
# Build Args:
#   - SERVICE_NAME: Name of the service (e.g., "minimal-connect-service", "user-service")
#   - BINARY_PATH: Path to pre-built binary (default: examples/bin/${SERVICE_NAME})
#   - HTTP_PORT: Main HTTP service port (default: 8080)
#   - HEALTH_PORT: Health check endpoint port (default: 8081)
#   - METRICS_PORT: Prometheus metrics port (default: 9091)
#
# Runtime Environment:
#   - HTTP_PORT: Service listens on this port for requests
#   - HEALTH_PORT: Health check endpoint (GET /health)
#   - METRICS_PORT: Prometheus metrics endpoint (GET /metrics)
#
# Security:
#   - Runs as non-root user 'app' (inherited from base image)
#   - Static binary with no runtime dependencies
#   - Minimal attack surface (no shell, no package manager)
#
# Usage Example:
#   docker build -f examples/docker/Dockerfile \
#     --build-arg SERVICE_NAME=user-service \
#     --build-arg BINARY_PATH=examples/bin/user-service \
#     --build-arg HTTP_PORT=8082 \
#     --build-arg HEALTH_PORT=8083 \
#     --build-arg METRICS_PORT=9092 \
#     -t user-service:latest .
#
# ==============================================================================

# Base image supports multi-platform (amd64/arm64)
# Platform is automatically selected by Docker buildx based on --platform flag
# No need to explicitly set --platform in FROM as buildx handles this automatically
FROM ghcr.io/eggybyte-technology/eggybyte-go-alpine:go1.25.1-alpine3.22

# Build-time arguments with defaults
ARG SERVICE_NAME="service"
ARG BINARY_PATH="examples/bin/${SERVICE_NAME}"
ARG HTTP_PORT=8080
ARG HEALTH_PORT=8081
ARG METRICS_PORT=9091

# Image metadata for example services
LABEL org.opencontainers.image.title="egg-example-${SERVICE_NAME}" \
      org.opencontainers.image.description="EggyByte framework example service: ${SERVICE_NAME}" \
      org.opencontainers.image.source="https://github.com/eggybyte/egg" \
      org.opencontainers.image.vendor="EggyByte Technology" \
      org.opencontainers.image.licenses="MIT"

# Set working directory (inherited from base, explicit for clarity)
WORKDIR /app

# Copy pre-built binary with execute permissions
# Binary must be compiled with CGO_ENABLED=0 for static linking
COPY --chmod=755 ${BINARY_PATH} /app/app

# Configure runtime environment variables
# Services should read these to determine which ports to bind
ENV HTTP_PORT=${HTTP_PORT} \
    HEALTH_PORT=${HEALTH_PORT} \
    METRICS_PORT=${METRICS_PORT}

# Expose service ports for documentation and inter-container communication
# Note: These don't actually publish ports; use -p flag at runtime
EXPOSE ${HTTP_PORT} ${HEALTH_PORT} ${METRICS_PORT}

# Run the service binary
# Inherits ENTRYPOINT ["/app/app"] from base image
# Override with CMD if service requires arguments

