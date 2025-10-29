非常好，我们现在要完成的是一整套 **标准化的 EggyByte 镜像体系构建链**，涵盖从 builder → runtime → 二进制生成 → 服务镜像的全流程，支持 **go.work monorepo**、**多模块引用**、**外部参数化**。

以下方案完全可落地，并已考虑未来与 `egg CLI` 的自动集成。
所有服务名、路径、输出目录等均通过 **环境变量或 build args** 参数化，保持高度可扩展。

---

# 🧱 1. `Dockerfile.builder` — EggyByte Go Builder 镜像

**镜像名：`ghcr.io/eggybyte-technology/eggybyte-go-builder:go1.25.1-alpine3.22`**

> 用于在 CI 或本地统一构建所有 Go 二进制文件。
> 仅包含编译工具链、依赖和基础构建环境。
> 不做 proto 生成（由开发者自行管理）。

```dockerfile
# ==============================================================================
# EggyByte Go Builder
# Go: 1.25.1
# Base: Alpine 3.22
# Purpose: Unified build environment for all EggyByte Go microservices
# ==============================================================================
FROM golang:1.25.1-alpine3.22

LABEL org.opencontainers.image.title="EggyByte Go Builder" \
      org.opencontainers.image.description="Unified builder for EggyByte Go microservices (supports go.work, multi-module monorepo)." \
      org.opencontainers.image.source="https://github.com/eggybyte-technology/egg" \
      org.opencontainers.image.vendor="EggyByte Technology" \
      org.opencontainers.image.licenses="MIT"

# Install essential build tools (no proto generation here)
RUN apk add --no-cache \
      build-base \
      bash \
      git \
      ca-certificates \
      curl \
      tzdata

# Prepare build environment
ENV CGO_ENABLED=0 \
    GO111MODULE=on \
    GOPATH=/go \
    PATH=$PATH:/go/bin

WORKDIR /src

# Entrypoint kept flexible; egg CLI will provide build commands
ENTRYPOINT ["/bin/bash"]
```

### 🧩 构建指令

```bash
docker build -f Dockerfile.builder \
  -t ghcr.io/eggybyte-technology/eggybyte-go-builder:go1.25.1-alpine3.22 .
```

---

# 🧩 2. `Dockerfile.runtime` — EggyByte Go Runtime 镜像

**镜像名：`ghcr.io/eggybyte-technology/eggybyte-go-alpine:go1.25.1-alpine3.22`**

> 最小化运行时容器，用于运行由 builder 编译的二进制。

```dockerfile
# ==============================================================================
# EggyByte Go Runtime (Minimal Alpine Runtime for Go services)
# Go Runtime: 1.25.1
# Base: Alpine 3.22
# ==============================================================================
FROM alpine:3.22

LABEL org.opencontainers.image.title="EggyByte Go Runtime" \
      org.opencontainers.image.description="Minimal runtime for EggyByte Go microservices (no compiler, just CA and TZ)." \
      org.opencontainers.image.vendor="EggyByte Technology" \
      org.opencontainers.image.licenses="MIT"

RUN apk add --no-cache \
      ca-certificates \
      tzdata && \
    addgroup -S app && adduser -S -G app app

WORKDIR /app

USER app

ENTRYPOINT ["/app/app"]
```

### 🧩 构建指令

```bash
docker build -f Dockerfile.runtime \
  -t ghcr.io/eggybyte-technology/eggybyte-go-alpine:go1.25.1-alpine3.22 .
```

---

# 🧩 3. `Dockerfile.build-binaries` — 构建所有服务的二进制

**用途：由 egg CLI 调用，用于一次性构建所有服务二进制文件**

> 参数化支持：
>
> * `BUILD_DIR`：输出目录（默认 `/build`）
> * `SERVICE_LIST`：服务目录列表（逗号分隔，如 `backend/user,backend/order`）
> * `GEN_MODULE_DIR`：proto 生成的 go 模块路径（默认 `gen/go`）
> * `GO_VERSION`：可选覆盖（默认 1.25.1）

```dockerfile
# ==============================================================================
# EggyByte Multi-Service Binary Builder (Optimized)
# Go: 1.25.1
# Base: ghcr.io/eggybyte-technology/eggybyte-go-builder:go1.25.1-alpine3.22
#
# Project structure expected:
#   /
#   ├── backend/
#   │   ├── go.work
#   │   ├── user/
#   │   ├── order/
#   │   └── ...
#   └── gen/
#       └── go/
#           ├── go.mod
#           └── go.sum
#
# Purpose:
#   - Build all backend microservice binaries from a go.work workspace
#   - Only copy backend/ (with go.work) + gen/go/ (generated proto module)
#   - Output binaries to /build/<service>/server
# ==============================================================================

FROM ghcr.io/eggybyte-technology/eggybyte-go-builder:go1.25.1-alpine3.22 AS builder

# -----------------------------
# Build-time arguments
# -----------------------------
ARG BACKEND_DIR="backend"
ARG GEN_DIR="gen/go"
ARG SERVICE_LIST=""       # Comma-separated list of services, e.g. "user,order,payment"
ARG BUILD_DIR="/build"

# Allow external override of Go environment
ARG GOOS=linux
ARG GOARCH=amd64

# -----------------------------
# Prepare workspace
# -----------------------------
WORKDIR /src

# Copy only go.work (workspace metadata)
COPY ${BACKEND_DIR}/go.work ${BACKEND_DIR}/

# Copy gen/go module (buf generated code)
COPY ${GEN_DIR}/go.mod ${GEN_DIR}/go.sum ${GEN_DIR}/

# Warm up module cache for shared deps
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd ${GEN_DIR} && go mod download

# Copy the full backend and gen/go source
COPY ${BACKEND_DIR} ${BACKEND_DIR}
COPY ${GEN_DIR} ${GEN_DIR}

# Sync workspace (connect backend modules with gen/go)
RUN cd ${BACKEND_DIR} && go work sync || true

# -----------------------------
# Build binaries
# -----------------------------
RUN mkdir -p ${BUILD_DIR}

# We use a small shell loop to build each service dynamically.
# If SERVICE_LIST is empty, build all first-level dirs under backend/.
RUN set -eux; \
    cd ${BACKEND_DIR}; \
    if [ -z "${SERVICE_LIST}" ]; then \
      echo "[INFO] No SERVICE_LIST provided, auto-detecting backend modules..."; \
      SERVICE_LIST=$(find . -mindepth 1 -maxdepth 1 -type d ! -name 'gen' -exec basename {} \;); \
    fi; \
    echo "[INFO] Building services: ${SERVICE_LIST}"; \
    for svc in $(echo ${SERVICE_LIST} | tr ',' ' '); do \
      echo "[BUILD] -> ${svc}"; \
      mkdir -p /bin/${svc}; \
      CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} \
        go build -trimpath -ldflags="-s -w" \
        -o /bin/${svc}/server \
        -C ${svc} ./cmd/server; \
    done

# -----------------------------
# Result
# -----------------------------
# Output binaries: /build/<service>/server
# Example:
#   /build/user/server
#   /build/order/server
# -----------------------------
```

### 🧩 构建指令

```bash
docker buildx build --platform linux/amd64,linux/arm64 \
  -f Dockerfile.build-binaries \
  --build-arg SERVICE_LIST="user,order" \
  -t ghcr.io/eggybyte-technology/eggybyte-build-binaries:multiarch .
```

### 🧩 输出目录说明

容器中 `/bin/<service>/server` 会包含所有二进制，
你可以用 `docker cp` 或 bind mount (`-v $(pwd)/bin:/bin`) 导出结果。

---

# 🧩 4. `Dockerfile.service` — 构建最终服务镜像

**用途：由 egg CLI 自动化打包 runtime 镜像，每个服务独立镜像。**

> 参数化支持：
>
> * `SERVICE_NAME`：服务名（自动传入）
> * `BINARY_PATH`：对应二进制文件路径（默认 `build/${SERVICE_NAME}/server`）
> * `HTTP_PORT`、`HEALTH_PORT`、`METRICS_PORT`

```dockerfile
# ==============================================================================
# EggyByte Backend Service Image (Runtime)
# Purpose: Package a pre-built binary into the standard runtime container.
# ==============================================================================
FROM ghcr.io/eggybyte-technology/eggybyte-go-alpine:go1.25.1-alpine3.22

# Build-time arguments
ARG SERVICE_NAME="user"
ARG BINARY_PATH="build/${SERVICE_NAME}/server"
ARG HTTP_PORT=8080
ARG HEALTH_PORT=8081
ARG METRICS_PORT=9091

WORKDIR /app

# Copy the prebuilt binary
COPY --chmod=755 ${BINARY_PATH} /app/server

# Environment ports for runtime discovery
ENV HTTP_PORT=${HTTP_PORT} \
    HEALTH_PORT=${HEALTH_PORT} \
    METRICS_PORT=${METRICS_PORT}

EXPOSE ${HTTP_PORT} ${HEALTH_PORT} ${METRICS_PORT}

ENTRYPOINT ["/app/server"]
```

### 🧩 构建指令（示例）

```bash
# 为 user 服务打包镜像
docker build -f Dockerfile.service \
  --build-arg SERVICE_NAME=user \
  --build-arg BINARY_PATH=build/backend/user/server \
  -t ghcr.io/eggybyte-technology/eggybyte-foundation-user:v0.3.0 .
```

---

# 🚀 整体构建流程一览

```bash
# Step 1. 构建 builder 镜像（一次性）
docker build -f Dockerfile.builder \
  -t ghcr.io/eggybyte-technology/eggybyte-go-builder:go1.25.1-alpine3.22 .

# Step 2. 构建 runtime 镜像（一次性）
docker build -f Dockerfile.runtime \
  -t ghcr.io/eggybyte-technology/eggybyte-go-alpine:go1.25.1-alpine3.22 .

# Step 3. 构建所有二进制（统一输出）
docker run --rm -v $(pwd):/src -w /src \
  ghcr.io/eggybyte-technology/eggybyte-go-builder:go1.25.1-alpine3.22 \
  bash -c 'go work sync; for s in backend/*; do echo "Building $s"; CGO_ENABLED=0 GOOS=linux go build -C $s -o build/$s/server ./cmd/server; done'

# Step 4. 打包各服务镜像
docker build -f Dockerfile.service \
  --build-arg SERVICE_NAME=user \
  --build-arg BINARY_PATH=build/backend/user/server \
  -t ghcr.io/eggybyte-technology/eggybyte-foundation-user:v0.3.0 .

docker build -f Dockerfile.service \
  --build-arg SERVICE_NAME=order \
  --build-arg BINARY_PATH=build/backend/order/server \
  -t ghcr.io/eggybyte-technology/eggybyte-foundation-order:v0.3.0 .
```

---

# 🧩 与 egg CLI 的集成建议

未来在你的 `egg CLI` 中可实现命令：

```bash
egg build all
```

等价于：

1. 使用 `eggybyte-go-builder` 构建所有服务二进制到 `build/`；
2. 遍历每个 `backend/<service>`；
3. 为每个服务执行：

   ```bash
   docker build -f Dockerfile.service \
     --build-arg SERVICE_NAME=<service> \
     --build-arg BINARY_PATH=build/backend/<service>/server \
     -t ghcr.io/eggybyte-technology/<project>-<service>:<version> .
   ```

这样开发者一条命令即可构建出所有镜像。

---

# ✅ 最终总结

| 镜像                    | 作用      | 构建频率      | 是否参数化                    |
| --------------------- | ------- | --------- | ------------------------ |
| `eggybyte-go-builder` | 构建用工具镜像 | 一次构建、长期复用 | Go 版本可改                  |
| `eggybyte-go-alpine`  | 运行时镜像   | 一次构建、长期复用 | Go/Alpine 版本可改           |
| `build-binaries`      | 批量生成二进制 | 每次构建执行    | 支持 SERVICE_LIST          |
| `service`             | 单服务最终镜像 | 每服务一次     | SERVICE_NAME、BINARY_PATH |
