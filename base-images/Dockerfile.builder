# ==============================================================================
# EggyByte Go Builder Image
# ==============================================================================
# 
# Purpose:
#   Unified build environment for compiling all EggyByte Go microservices.
#   Supports go.work workspace-based monorepo projects with multiple modules.
#   This image provides the complete Go toolchain and essential build utilities.
#
# Base Image:
#   golang:1.25.1-alpine3.22 (official Go image with Alpine Linux)
#
# Key Features:
#   - Full Go 1.25.1 compiler toolchain
#   - Essential build tools (gcc, make, git)
#   - CA certificates for secure HTTPS connections
#   - Timezone data for accurate time handling
#   - No protocol buffer generation tools (managed separately in development)
#
# Usage:
#   This image is intended for compilation only. Use it to build Go binaries
#   that will be packaged into the minimal runtime image (eggybyte-go-alpine).
#
#   Example:
#     docker run --rm -v $PWD:/src -w /src \
#       ghcr.io/eggybyte-technology/eggybyte-go-builder:go1.25.1-alpine3.22 \
#       go build -o bin/service ./cmd/server
#
# Build Args:
#   None (base image version is pinned)
#
# Security:
#   - Runs as root (build-time only, not for production)
#   - Includes CA certificates for secure package downloads
#   - No sensitive data should be baked into this image
#
# Maintenance:
#   - Go version: 1.25.1 (update base image tag to upgrade)
#   - Alpine version: 3.22 (matches base image)
#   - Review and update dependencies quarterly
#
# ==============================================================================

FROM golang:1.25.1-alpine3.22

# Image metadata (OCI standard labels)
LABEL org.opencontainers.image.title="EggyByte Go Builder" \
      org.opencontainers.image.description="Unified builder for EggyByte Go microservices (supports go.work, multi-module monorepo)." \
      org.opencontainers.image.source="https://github.com/eggybyte-technology/egg" \
      org.opencontainers.image.vendor="EggyByte Technology" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="go1.25.1-alpine3.22"

# Install essential build tools
# - build-base: gcc, g++, make and other compilation tools
# - bash: shell for build scripts
# - git: version control for go mod operations
# - ca-certificates: trusted CA roots for HTTPS
# - curl: HTTP client for downloading dependencies
# - tzdata: timezone database for time-aware builds
RUN apk add --no-cache \
      build-base \
      bash \
      git \
      ca-certificates \
      curl \
      tzdata

# Configure Go build environment
# CGO_ENABLED=0: Disable CGO for static binaries (overridable at build time)
# GO111MODULE=on: Enable Go modules (default in Go 1.16+, explicit for clarity)
# GOPATH=/go: Default Go workspace path
# PATH: Include Go binary directory for installed tools
ENV CGO_ENABLED=0 \
    GO111MODULE=on \
    GOPATH=/go \
    PATH=$PATH:/go/bin

# Set working directory for build operations
# Egg CLI will mount the project source into /src
WORKDIR /src

# Default entrypoint: bash shell for flexible command execution
# Egg CLI will override this with specific build commands
ENTRYPOINT ["/bin/bash"]

