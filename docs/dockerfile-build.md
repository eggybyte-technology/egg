非常精准。
你提到的两个核心优化方向正是**企业级生产构建体系应做的最终收敛设计**：

1️⃣ `OUT_DIR` 不应只存在于 builder 阶段，而应在 **三个阶段（builder / runtime / export 或最终 runtime）中可见**，
以实现真正的「可配置输出路径」和跨阶段一致性。

2️⃣ `export` 阶段在企业生产流水线中确实可以去掉 —— 因为产物导出属于开发流程，
CI/CD 主要目标是 **生成多架构可运行镜像**，不再需要本地中转层。

所以我们将给出一份真正的 **最终企业级标准版 Dockerfile**，
具备以下特点：

---

## ✅ 企业级规范特征

| 项目                  | 描述                                                 |
| ------------------- | -------------------------------------------------- |
| **多阶段结构**           | `builder` → `runtime`（双阶段，清晰职责）                    |
| **全局 `OUT_DIR` 参数** | 在所有阶段可见、可重定义（默认 `/out`）                            |
| **多平台构建**           | 支持 `linux/amd64`、`linux/arm64` 等                   |
| **工作区结构**           | `backend/go.work` + `gen/go` + `backend/<service>` |
| **输出路径一致**          | `OUT_DIR` 控制生成物路径与拷贝来源路径                           |
| **不包含 export 阶段**   | 专注产镜像（CI/CD 优先）                                    |
| **完全可配置**           | 可通过 `--build-arg` 修改输出路径、端口、服务名等                   |
| **安全与最小化**          | runtime 基于 `alpine`，builder 隔离，CGO 禁用              |
| **BuildKit 缓存优化**   | 精细 COPY 层，减少冗余重构                                   |

---

# 🧱 最终版企业级标准 Dockerfile

```dockerfile
# ==============================================================================
# EggyByte Enterprise Go Service Dockerfile (Builder + Runtime)
# ------------------------------------------------------------------------------
# Version: v1.0.1
# Language: Go 1.25.1
# Base OS: Alpine 3.22
# ------------------------------------------------------------------------------
# Purpose:
#   - Unified, enterprise-grade multi-platform Dockerfile
#   - Supports builder/runtime separation in a single file
#   - Allows fully configurable OUT_DIR across all stages
# ------------------------------------------------------------------------------
# Directory layout:
#   backend/go.work
#   backend/<service>/cmd/server
#   gen/go/   (buf-generated Go code)
# ==============================================================================

# ------------------------------------------------------------------------------
# Global build arguments (visible in all stages)
# ------------------------------------------------------------------------------
ARG GO_VERSION=1.25.1
ARG OUT_DIR=/out                # Unified output directory across all stages
ARG SERVICE_NAME=user           # Default service
ARG HTTP_PORT=8080
ARG HEALTH_PORT=8081
ARG METRICS_PORT=9091

# ------------------------------------------------------------------------------
# 1️⃣ Builder Stage
# ------------------------------------------------------------------------------
FROM ghcr.io/eggybyte-technology/eggybyte-go-builder:go${GO_VERSION}-alpine3.22 AS builder

LABEL org.opencontainers.image.title="EggyByte Builder" \
      org.opencontainers.image.description="Build stage for EggyByte Go microservices" \
      org.opencontainers.image.vendor="EggyByte Technology"

# Inherit global args
ARG SERVICE_NAME
ARG OUT_DIR
ARG HTTP_PORT
ARG HEALTH_PORT
ARG METRICS_PORT
ARG TARGETOS TARGETARCH

WORKDIR /src

# Copy only necessary files to ensure build cache efficiency
COPY gen/go gen/go
COPY backend backend

# Build the service binary
RUN echo "🏗️  Building service '${SERVICE_NAME}' for ${TARGETOS}/${TARGETARCH}" && \
    mkdir -p ${OUT_DIR}/${SERVICE_NAME} && \
    cd backend/${SERVICE_NAME} && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" \
    -o ${OUT_DIR}/${SERVICE_NAME}/server ./cmd/server && \
    echo "✅ Binary built at ${OUT_DIR}/${SERVICE_NAME}/server"

# ------------------------------------------------------------------------------
# 2️⃣ Runtime Stage
# ------------------------------------------------------------------------------
FROM ghcr.io/eggybyte-technology/eggybyte-go-alpine:go${GO_VERSION}-alpine3.22 AS runtime

LABEL org.opencontainers.image.title="EggyByte Go Runtime" \
      org.opencontainers.image.description="Standard runtime for EggyByte Go microservices" \
      org.opencontainers.image.source="https://github.com/eggybyte-technology/egg" \
      org.opencontainers.image.vendor="EggyByte Technology" \
      org.opencontainers.image.licenses="MIT"

# Inherit same args to maintain global consistency
ARG SERVICE_NAME
ARG OUT_DIR
ARG HTTP_PORT
ARG HEALTH_PORT
ARG METRICS_PORT

WORKDIR /app

# Copy the binary from builder using the unified OUT_DIR path
COPY --from=builder ${OUT_DIR}/${SERVICE_NAME}/server /app/app

# Set standard runtime environment
ENV SERVICE_NAME=${SERVICE_NAME} \
    OUT_DIR=${OUT_DIR} \
    HTTP_PORT=${HTTP_PORT} \
    HEALTH_PORT=${HEALTH_PORT} \
    METRICS_PORT=${METRICS_PORT}

# Expose standard EggyByte service ports
EXPOSE ${HTTP_PORT} ${HEALTH_PORT} ${METRICS_PORT}

ENTRYPOINT ["/app/app"]
```

---

# ⚙️ 构建命令（标准企业级流程）

### ✅ 1️⃣ 单平台构建本地镜像

```bash
docker build \
  -f Dockerfile \
  --build-arg SERVICE_NAME=user \
  --build-arg OUT_DIR=/build-out \
  -t ghcr.io/eggybyte-technology/eggybyte-foundation-user:v0.3.0 .
```

### ✅ 2️⃣ 多平台构建并推送镜像

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -f Dockerfile \
  --build-arg SERVICE_NAME=user \
  --build-arg OUT_DIR=/out \
  -t ghcr.io/eggybyte-technology/eggybyte-foundation-user:v0.3.0 \
  --push .
```

### ✅ 3️⃣ 仅构建不推送（CI 本地缓存）

```bash
docker buildx build \
  --platform linux/amd64 \
  -f Dockerfile \
  --build-arg SERVICE_NAME=order \
  -t ghcr.io/eggybyte-technology/eggybyte-foundation-order:dev \
  --load .
```

---

# 📁 输出产物结构（在容器内）

```
/app/app                          ← Runtime binary
/out/<service>/server             ← Builder binary output (可保留调试层)
```

---

# 🧠 设计思路详解

### 1️⃣ **全局 OUT_DIR**

* 在所有阶段统一声明 `ARG OUT_DIR`；
* 保证路径、复制源、日志输出一致；
* 允许开发者通过 `--build-arg OUT_DIR=/workspace/out` 自定义。

### 2️⃣ **无导出阶段**

* CI/CD 不再导出产物；
* 所有镜像通过 runtime 阶段封装；
* 可在运行时 `docker cp` 取出二进制调试（若需要）。

### 3️⃣ **多平台统一**

* 通过 BuildKit 自动注入 `TARGETOS`、`TARGETARCH`；
* 不需人工指定架构；
* 一次 buildx build 可生成多架构镜像。

### 4️⃣ **安全与最小化**

* builder → runtime 全隔离；
* runtime 仅包含 `/app/app`；
* 镜像小（<20MB），启动快；
* 禁用 CGO（`CGO_ENABLED=0`）防止外部依赖。
