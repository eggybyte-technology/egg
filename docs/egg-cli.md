# egg 命令行工具（egg/cli）——完整设计与实现方案

> 目标：在**不改变既有工作方法与指令语义**的前提下，提供一套 Connect-first、Kubernetes-native、Monorepo 友好的 CLI。CLI 与 **egg 通用库**同仓（monorepo），CLI 源码放置于 `egg/cli`；通过命令行生成的项目为**外部工程**（单独目录），两者**严格区分**。

---

## 1. 两套目录结构（必须区分）

### 1.1 仓库（egg monorepo）结构（含 CLI 与通用库）

```
egg/                           # Git 仓库根（模块名：github.com/eggybyte-technology/egg）
├─ go.work                     # 仅仓库内部模块聚合（不向生成项目泄露）
├─ core/                       # 通用库（示例：基础工具与接口）
├─ runtimex/                   # 统一进程生命周期、HTTP/H2C、健康/指标
├─ connectx/                   # Connect 拦截器栈（日志/追踪/身份/错误态）
├─ configx/                    # 强类型配置绑定、K8s 名称法动态覆盖
├─ obsx/ k8sx/ storex/ ...     # 其他通用模块
└─ cli/                        # ← 命令行工具的**唯一**位置
   ├─ cmd/egg/                 # main 包（`egg` 可执行文件入口）
   ├─ internal/
   │  ├─ configschema/         # egg.yaml 的 schema、加载与默认值填充
   │  ├─ projectfs/            # 脚手架写入（保证幂等），最小模板（非大段代码拷贝）
   │  ├─ toolrunner/           # go/buf/flutter/docker/kubectl/helm 调用器（exec）
   │  ├─ generators/           # api/backend/frontend 生成器
   │  ├─ render/
   │  │  ├─ compose/           # docker-compose 渲染（含 database→mysql:9.4 附加）
   │  │  └─ helm/              # Helm values 与模板渲染（ConfigMap/Secret 名称法）
   │  ├─ ref/                  # ${cfg|cfgv|sec|svc:...} 表达式解析
   │  ├─ lint/                 # 结构/端口/表达式/依赖检查
   │  └─ ui/                   # 统一日志、进度条、诊断与修复建议
   └─ testdata/                # golden 测试用例（渲染结果、骨架工程等）
```

### 1.2 通过 CLI 生成的**项目**结构（外部）

```
my-platform/                   # 用户项目根（由 `egg init` 生成）
├─ egg.yaml                    # 顶层配置（唯一事实源）
├─ api/                        # proto 与 buf 配置
│  ├─ buf.yaml
│  └─ buf.gen.yaml             # 生成 Go/Dart/TS(pb)/OpenAPI，TS 仅 bufbuild/es（pb types）
├─ backend/                    # Go 服务与 go.work（项目内）
│  ├─ go.work                  # 只引用 backend/* 子模块
│  └─ user-service/            # 由 `egg create backend` 生成的微服务
│     ├─ go.mod
│     ├─ cmd/server/main.go    # Connect-only 单 HTTP 端口 + 健康/指标
│     └─ internal/{config,handler,service,...}
├─ frontend/
│  └─ admin-portal/            # `egg create frontend` 可选生成（Flutter web）
├─ gen/
│  ├─ go/                      # buf 生成的 Go 代码（Protobuf 消息和 Connect 客户端/服务端）
│  ├─ dart/                    # buf 生成的 Dart 代码（Protobuf 消息和 Connect 客户端/服务端）
│  └─ openapi/                 # OpenAPI/Swagger 文档
├─ build/                      # Dockerfile 模板与构建脚本
└─ deploy/                     # 渲染输出：compose.yaml / helm/values.yaml 等
```

---

## 2. 核心约束与默认行为

* **Connect-first**：对外只开放 **HTTP**（h2/h2c，承载 Connect/gRPC-Web）；**健康/指标**独立端口。
* **端口继承**：`backend.<svc>.ports` 可省略，将自动继承 `backend_defaults.ports`；只有自定义时才在服务级覆盖。
* **K8s「名称法」**：ConfigMap 在容器环境只注入**名称**；Secret 使用 `valueFrom.secretKeyRef`；`svc:` 表达式指明 `clusterip|headless`。
* **数据库联调**：`database.enabled=true` 时，`egg compose up` 自动拉起 `mysql:9.4` 并注入典型 `DATABASE_DSN`。
* **官方工具优先**：不直接改写 `go.mod/go.work` 等文件，统一通过 `go`/`buf`/`flutter`/`helm`/`kubectl`/`docker` 命令完成。
* **完全保留既有 YAML 能力**：资源段（ConfigMap/Secret）、双 Service（clusterIP + headless）、三层 env 合并、表达式引擎、Compose/K8s 差异注入等一项不缺。

---

## 3. `egg.yaml` 规范（摘要）

```yaml
config_version: "1.0"
project_name: "my-platform"
version: "v1.0.0"
module_prefix: "github.com/eggybyte-technology/my-platform"
docker_registry: "ghcr.io/eggybyte-technology"

build:
  platforms: ["linux/amd64","linux/arm64"]
  go_runtime_image: "eggybyte-go-alpine"

env:
  global:
    LOG_LEVEL: "info"
    KUBERNETES_NAMESPACE: "prod"
  backend:
    DATABASE_DSN: "user:pass@tcp(mysql:3306)/app?charset=utf8mb4&parseTime=True"
  frontend:
    FLUTTER_BASE_HREF: "/"

backend_defaults:
  ports: { http: 8080, health: 8081, metrics: 9091 }   # 服务级默认值（可被覆盖）

kubernetes:
  resources:
    configmaps:
      global-config:
        FEATURE_A: "on"
    secrets:
      jwtkey:
        KEY: "super-secret"

backend:
  user-service:
    image_name: "user-service"
    # ports: { http: 18080, health: 18081, metrics: 19091 }  # 仅需自定义时才写
    kubernetes:
      service:
        clusterIP: { name: "user-svc" }
        headless:  { name: "user-svc-headless", publishNotReadyAddresses: true }
    env:
      common:
        SERVICE_NAME: "user-service"
      docker:
        RATE_LIMIT_QPS: "${cfgv:global-config:FEATURE_A}"   # Compose 可解值
      kubernetes:
        APP_CONFIGMAP_NAME: "${cfg:global-config}"          # 名称法
        JWT_TOKEN:         "${sec:jwtkey:KEY}"              # secretKeyRef
        PROXY_TARGET:      "${svc:user-service@headless}"   # 发现策略

frontend:
  admin-portal:
    platforms: ["web"]
    image_name: "admin-portal"

database:
  enabled: false                 # true → compose 自动附加 mysql:9.4
  image: "mysql:9.4"
  port: 3306
  root_password: "rootpass"
  database: "app"
  user: "user"
  password: "pass"
```

**表达式规则（摘要）**

* `${cfg:<cm>}`：注入 **ConfigMap 名称**（K8s），Compose 禁用；
* `${cfgv:<cm>:<key>}`：注入值（仅本地 Compose 可解）；
* `${sec:<secret>:<key>}`：K8s 渲染为 `secretKeyRef`（Compose 默认不直接注入密文）；
* `${svc:<name>@clusterip|headless}`：注入服务名称与类型提示（应用侧据此选择解析策略）。

---

## 4. 命令体系（与既有语义一致）

```
egg
├─ init                            # 初始化外部项目骨架（api/backend/frontend/gen/build/deploy/egg.yaml）
├─ create
│  ├─ backend <name>               # 生成后端微服务骨架（Connect-only；服务级端口默认继承）
│  └─ frontend <name> --platforms web
├─ api
│  ├─ init                         # 写入最小 buf.yaml / buf.gen.yaml
│  └─ generate                     # buf generate → Go/Dart/TS(pb)/OpenAPI
├─ compose
│  ├─ up [--detached]              # 渲染 + 启动；database.enabled=true → 附加 mysql:9.4
│  ├─ down                         # 停止并清理
│  └─ logs [--service <name>]      # 聚合日志
├─ kube
│  ├─ template [-n <ns>]           # Helm 渲染到 deploy/helm/*
│  ├─ apply [-n <ns>]              # kubectl apply
│  └─ uninstall [-n <ns>]          # 卸载
├─ build [--push] [--version vX.Y.Z] [--subset svc1,svc2]
└─ check                           # 结构/端口/表达式/数据库配置等一致性校验
```

**关键行为差异点**

* `create backend`：不会写死端口；默认由 `backend_defaults.ports` 注入 `HTTP_PORT/HEALTH_PORT/METRICS_PORT`。
* `api generate`：生成 Go/Dart/OpenAPI 三种代码；Go 和 Dart 都包含完整的 Protobuf 消息和 Connect 客户端/服务端代码。
* `compose up`：若 `database.enabled=true`，自动注入 `mysql:9.4` 服务与依赖关系；后端服务无显式 `DATABASE_DSN` 时仍以 `env.backend.DATABASE_DSN` 为准（建议在模板中给出指向 `mysql:3306` 的默认值，便于即开即用）。
* `kube *`：严格名称法与 secretKeyRef；双 Service（clusterIP/headless）按 YAML 渲染。

---

## 5. CLI 实现结构（egg/cli）

### 5.1 命令入口（Cobra 建议）

* `cli/cmd/egg/main.go`：解析全局 flag（`--verbose`、`--non-interactive` 等），挂载子命令。

### 5.2 内部模块职责

* `internal/configschema`

  * `Load(path) (Config, Diagnostics)`：读取 `egg.yaml`，校验 schema，填充默认值（含端口继承）。
  * `Validate(Config)`：检查端口冲突、服务名/镜像名合法性、表达式引用存在性、数据库字段完备性等。

* `internal/projectfs`

  * 幂等写入骨架（存在则跳过/合并），**只写最小模板**：`api/*` 的 buf 配置、`backend/go.work`、最小 `main.go` 等。
  * **绝不**直接编辑 `go.mod/go.work` 内容，全部通过 `toolrunner` 执行官方命令达成。

* `internal/toolrunner`

  * `Go`：`go mod init|tidy|get`，`go work init|use`；
  * `Buf`：`buf config init|generate`；
  * `Flutter`：`flutter create`、`flutter build web`；
  * `Docker`：build/push；`Helm`、`Kubectl`：template/apply。
  * 统一日志与错误处理（含命令回显、失败建议、可重试提示）。

* `internal/generators`

  * `API.Init()`：写入最小 `buf.yaml / buf.gen.yaml`；
  * `API.Generate()`：执行 `buf generate`，输出到 `gen/*`；生成 Go/Dart/OpenAPI 三种代码；
  * `Backend.Create(name)`：`go mod init` + `go work use` + 连接 `egg` 库（`runtimex/connectx/configx/obsx`），生成 Connect-only 服务骨架；
  * `Frontend.Create(name, platforms)`：`flutter create`（平台默认 web）。

* `internal/render/compose`

  * `Render(config)`：端口注入（`svc.ports ?? defaults.ports`）、Compose 环境变量合并、表达式解析（Compose 侧限制：禁止 `${cfg:}` 直接注值，`${cfgv:}` 允许，本地开发 Secret 默认不注）；
  * `AttachMySQL(config)`：`database.enabled=true` → 追加 `mysql:9.4` 服务（`MYSQL_*` 环境、健康检查、3306 端口映射），并为后端服务追加 `depends_on: [mysql]`。

* `internal/render/helm`

  * `Render(config)`：将 `${cfg:}` 渲染为名称字符串；`${sec:}` 渲染为 `valueFrom.secretKeyRef`；`${svc:}` 注入 `<NAME>/<KIND>` 标识并输出相应 Service；
  * 双 Service（clusterIP/headless）按 YAML 渲染；Pod/Service 端口与注入的 `HTTP/HEALTH/METRICS` 保持一致；生成至 `deploy/helm/<svc>/templates/*` 与 `values.yaml`。

* `internal/ref`

  * 统一实现四类表达式的词法/语义解析与目标环境（Compose/K8s）策略。
  * 对非法或不支持的用法（如 Compose 中使用 `${cfg:}`）给出**清晰错误**与迁移建议。

* `internal/lint`

  * 规则：端口继承一致性（服务级与默认值相同则提示可删除）、`grpc` 字段兼容性提示（Connect-only，忽略并给 warning）、数据库必填项检查、资源引用存在性校验、Service 命名规范校验等。

* `internal/ui`

  * 统一风格化输出（步骤标题、完成打勾、告警/错误）、`--json` 机器可读模式（可选）。

---

## 6. 服务骨架（由 `create backend` 生成，最小示意）

```go
// internal/config/app_config.go
package config

import "github.com/eggybyte-technology/egg/configx"

type AppConfig struct {
  configx.BaseConfig             // 含 HTTP/Health/Metrics/APP_CONFIGMAP_NAME 等基线键
  RateLimitQPS int `env:"RATE_LIMIT_QPS" default:"200"`
}
```

```go
// cmd/server/main.go
package main

import (
  "net/http"
  "time"
  "context"

  "github.com/eggybyte-technology/egg/runtimex"
  "github.com/eggybyte-technology/egg/connectx"
  "github.com/eggybyte-technology/egg/configx"
)

func main() {
  ctx := context.Background()
  var cfg config.AppConfig
  configx.MustLoad(&cfg) // 读取 env + 名称法覆盖

  mux := http.NewServeMux()
  ints := connectx.DefaultInterceptors(connectx.Options{SlowRequestMillis: 1000})
  // 绑定你的 Connect handler：path, h := yourapi.NewServiceHandler(..., connect.WithInterceptors(ints...))
  // mux.Handle(path, h)

  _ = runtimex.Run(ctx, nil, runtimex.Options{
    HTTP:    &runtimex.HTTPOptions{Addr: ":" + cfg.HTTPPort, H2C: true, Mux: mux},
    Health:  &runtimex.Endpoint{Addr: ":" + cfg.HealthPort},
    Metrics: &runtimex.Endpoint{Addr: ":" + cfg.MetricsPort},
    ShutdownTimeout: 15 * time.Second,
  })
}
```

> **结果**：对外仅一个 HTTP 端口；健康/指标独立端口；配置与观测基线开箱即用。

---

## 7. 渲染细节与策略对齐

* **端口注入**：`ports := svc.ports ?? backend_defaults.ports` → 注入 `HTTP_PORT/HEALTH_PORT/METRICS_PORT` 三变量，并用于生成 Service/ContainerPorts。
* **Compose**：禁止 `${cfg:}` 直接注入；`${cfgv:cm:key}` 允许；`${sec:}` 默认不直接注入（建议 `.env.local` 方式）；`${svc:}` 注入 `<NAME>/<KIND>` 标记（供应用侧解析 headless/clusterIP）。
* **Helm/K8s**：`${cfg:}` 注入**名称**；`${sec:}` → `secretKeyRef`；`${svc:}` 写入名称与类型；按类型渲染目标 Service（headless 开启 `publishNotReadyAddresses`，解析 SRV；clusterIP 解析 A 记录）。
* **旧字段兼容**：若服务级 YAML 仍存在 `grpc` 端口字段 → 忽略并给出“Connect-only”的**警告**（不失败，降低迁移成本）。

---

## 8. API 生成（buf）

* `api/buf.yaml`：`version: v1` + 需要的 `deps`。
* `api/buf.gen.yaml`：

  * **Go**：`buf.build/protocolbuffers/go` 和 `buf.build/connectrpc/go`，`out: ../gen/go`；
  * **Dart**：`buf.build/protocolbuffers/dart` 和 `buf.build/connectrpc/dart`，`out: ../gen/dart`；
  * **OpenAPI**：`buf.build/grpc-ecosystem/openapiv2`，`out: ../gen/openapi`。
* `egg api init/generate` 封装 `buf`，并对产物完成**幂等**更新与最小化写入。

---

## 9. 数据库（Compose 集成）

* 开启 `database.enabled=true` → `compose` 渲染时自动附加：

  * `mysql:9.4` 服务、`ports: ["3306:3306"]`、`MYSQL_*` 环境、健康检查；默认**不持久化**（可后续在 `egg.yaml` 中开启 volume 字段）。
  * 为后端服务追加 `depends_on: [mysql]`；如 `env.backend.DATABASE_DSN` 未覆盖，示例模板中给出指向 `mysql:3306` 的 DSN，保证“起步即通”。

---

## 10. 质量保障与测试

* **lint**：

  * 端口省略建议（与默认值相同则提示可移除）；
  * `grpc` 字段兼容提示；
  * 数据库必填项校验；
  * 资源/表达式引用存在性；
  * Service 命名/端口一致性。
* **测试**：

  * golden 测试（compose/helm 渲染结果、最小工程骨架、buf 产物路径）；
  * integration：`egg init → create backend → api init/generate → compose template → kube template`；
  * e2e（可选）：在 CI 中对示例工程执行一遍 `compose up -d`，比对健康检查与端口开放。

---

## 11. 版本与发布

* CLI 二进制 `egg`：`go build ./cli/cmd/egg`；
* 版本号随仓库 Tag（`vX.Y.Z`），`egg version` 输出仓库 Tag + commit；
* 配置版本 `config_version` 支持向后兼容检查，必要时提供 `egg migrate`（未来扩展）。

---

## 12. 典型工作流（端到端）

1. `egg init` → 生成外部项目骨架与 `egg.yaml`（含 `backend_defaults.ports`、`database.enabled=false`）。
2. `egg create backend user-service` → 生成 Connect-only 服务骨架；服务级**不写端口**（走继承）。
3. `egg api init && egg api generate` → 一次产出 **Go/Dart/OpenAPI**。
4. （可选）编辑 `egg.yaml`：`database.enabled=true`。
5. `egg compose up` → 启动 user-service + `mysql:9.4`。
6. `egg kube template -n prod && egg kube apply -n prod` → 渲染/发布。

---

### 小结

* **代码放置**：CLI 严格在 `egg/cli`，与通用库并存、职责清晰；
* **项目生成**：外部目录是“用户工程”，结构稳定，`egg.yaml` 为单一事实源；
* **一致体验**：保留既有 YAML 能力与指令语义；
* **现代基线**：Connect-only、K8s 名称法、buf 一次生成 Go/Dart/OpenAPI、Compose 一键数据库联调；
* **实现可落地**：模块划分清晰，全部用官方工具链操作，幂等与可测试性完备。

如果你需要，我可以把 `egg/cli` 的命令树（Cobra）、`configschema` 与 `render/compose|helm` 的最小实现骨架一次性生成，直接在你的仓库里开一个 MR 版本。
