# 1. 目标与总体原则（DX）

1. **一条命令拉起一套服务**：最少配置（仅 `egg.yaml`）。
2. **Connect-first / 单端口**：对外仅暴露一个 HTTP(h2c) 端口承载 Connect / gRPC-Web；健康、指标端口独立。
3. **名称法注入**（K8s）：只把 ConfigMap **名称**注入环境，应用内动态监听；Secret 用 `secretKeyRef`。
4. **官方工具优先**：所有 `go work|mod`、`buf generate`、`docker buildx`、`helm`、`kubectl` 均以命令驱动。
5. **内建可观测性**：健康检查、指标、日志、Tracing 拦截器默认打开。
6. **命名硬约束**：服务名不得包含 `-service` 尾缀；镜像名严格 `项目名-服务名`。
7. **幂等生成**：重复执行不会覆盖已有关键文件；提供 `egg check` 进行一致性校验与修复建议。

---

# 2. 命名与镜像规范（强制）

## 2.1 服务命名

* 允许：`user`、`order`、`billing-api`
* 禁止：`user-service`、`order_service`（`egg create backend` 将直接报错并给出修正建议）

## 2.2 镜像命名

* **规则**：`image_name = <project_name>-<service_name>`（全小写、`/`→`-`、空格→`-`、去重连字符）
* **最终镜像引用**：`<docker_registry>/<image_name>:<tag>`

  * 例：`docker_registry = ghcr.io/eggybyte-technology`，项目 `eggybyte-foundation`、服务 `user`
  * → `ghcr.io/eggybyte-technology/eggybyte-foundation-user:dev`

## 2.3 Tag 策略（默认）

* `:dev`（本地构建）
* `:vX.Y.Z`（发布）
* `:<git-sha-7>`（可选：CI 增量追踪）
  `egg build --version v0.2.0 --push` 会同时打 `v0.2.0` 与 `<git-sha-7>`。

---

# 3. 目录结构与生成物

## 3.1 CLI 仓库（已存在）

```
egg/                                  # go.eggybyte.com/egg
└─ cli/
   ├─ cmd/egg/
   └─ internal/{configschema,projectfs,toolrunner,generators,render,ref,lint,ui}
```

## 3.2 由 CLI 生成的项目

```
<project_root>/                       # 由 `egg init <project_name>` 生成
├─ egg.yaml                           # 唯一事实源
├─ api/                               # proto 与 buf
│  ├─ buf.yaml
│  └─ buf.gen.yaml
├─ gen/                               # buf 输出
│  ├─ go/
│  ├─ dart/
│  └─ openapi/
├─ backend/
│  └─ user/                           # ← 注意：无 -service 后缀
│     ├─ go.mod
│     ├─ cmd/server/main.go
│     └─ internal/{config,handler,service,repository}
├─ frontend/
│  └─ admin-portal/                   # 可选（Flutter web）
├─ build/                             # Dockerfile 模板
└─ deploy/                            # 渲染产物（compose.yaml、helm/*）
```

---

# 4. CLI 命令与行为（详细说明）

```
egg
├─ init <project-name>
│    - 在全新文件夹中初始化（不污染当前目录）
│    - 写入 egg.yaml / api/{buf.yaml,buf.gen.yaml} / build 模板
│    - go work init（空），等待后续 add
│
├─ create
│  ├─ backend <name> [flags]
│  │    --proto [none|echo|crud]   # 默认 echo
│  │    --pkg <proto.package>      # 默认由 module_prefix+服务域推导
│  │    --api-dir <path>           # 默认 ./api
│  │  行为：
│  │    - 校验 <name> 不得含 -service
│  │    - 在 backend/<name>/ 生成骨架（Connect-only 单端口）
│  │    - 在 api/<name>/v1 生成示例 proto（可选）
│  │    - 执行：
│  │       go work use ./backend/<name>
│  │       (如需) go mod init <module_prefix>/backend/<name>
│  │    - 更新 egg.yaml：backend.<name> 节点（无需手写端口）
│  │    - buf generate → gen/go|dart|openapi
│  │
│  └─ frontend <name> --platforms web
│       - 可选生成 Flutter Web 骨架；镜像名规则同样使用 `<project>-<frontend>`
│
├─ api
│  ├─ init                          # 最小 buf.yaml/buf.gen.yaml（存在则跳过）
│  └─ generate [--targets ...]      # 默认 go,dart,openapi；可选 ts(pb types)
│
├─ compose
│  ├─ up [--detached]
│  ├─ down
│  └─ logs [--service <name>]
│
├─ kube
│  ├─ template [-n <ns>]            # Helm 渲染
│  ├─ apply [-n <ns>]
│  └─ uninstall [-n <ns>]
│
├─ build [--push] [--version vX.Y.Z] [--subset svc1,svc2]
│    - docker buildx bake，平台来自 egg.yaml: build.platforms
│    - 镜像名自动 `<docker_registry>/<project>-<service>:<tag>`
│
├─ check                             # 结构、命名、端口、引用、表达式校验
└─ version
```

---

# 5. `egg.yaml` 规范 v0.2（含样例）

```yaml
config_version: "1.0"

project_name: "eggybyte-foundation"               # ← 用于镜像前缀“项目名-服务名”
module_prefix: "go.eggybyte.com/eggybyte-foundation"
docker_registry: "ghcr.io/eggybyte-technology"

build:
  platforms: ["linux/amd64","linux/arm64"]

env:
  global:
    LOG_LEVEL: "info"
  backend:
    DATABASE_DSN: "user:pass@tcp(mysql:3306)/app?charset=utf8mb4&parseTime=True"  # Compose 下默认

backend_defaults:
  ports: { http: 8080, health: 8081, metrics: 9091 }  # 未覆盖则继承

kubernetes:
  resources:
    configmaps:
      global-config:
        FEATURE_A: "on"
    secrets:
      jwtkey:
        KEY: "super-secret"

backend:
  user:                                         # ← 无 -service 后缀
    # image_name 自动：eggybyte-foundation-user（无需手写）
    kubernetes:
      service:
        clusterIP: { name: "user" }
        headless:  { name: "user-headless", publishNotReadyAddresses: true }
    env:
      common:
        SERVICE_NAME: "user"
      docker:
        RATE_LIMIT_QPS: "${cfgv:global-config:FEATURE_A}"     # 仅 Compose 生效
      kubernetes:
        APP_CONFIGMAP_NAME: "${cfg:global-config}"            # 名称法（应用动态监听）
        JWT_TOKEN:         "${sec:jwtkey:KEY}"
        PROXY_TARGET:      "${svc:user@headless}"

frontend:
  admin-portal:
    platforms: ["web"]
    # 前端镜像名也可复用规则：eggybyte-foundation-admin-portal

database:
  enabled: false                 # true → compose 自动附加 mysql:9.4
  image: "mysql:9.4"
  port: 3306
  root_password: "rootpass"
  database: "app"
  user: "user"
  password: "pass"
```

### 5.1 表达式与环境注入规则

* `${cfg:<cm>}`：仅 K8s，注入 **ConfigMap 名称**；应用内据此热读取。
* `${cfgv:<cm>:<key>}`：仅 Compose 可展开具体值；K8s **禁止**。
* `${sec:<secret>:<key>}`：K8s → `valueFrom.secretKeyRef`；Compose 默认不直注密文。
* `${svc:<name>@clusterip|headless}`：注入服务名（类型提示供渲染与应用解析 SRV）。

---

# 6. 后端骨架（生成要点）

## 6.1 Config

```go
// backend/user/internal/config/app_config.go
package config

import "go.eggybyte.com/egg/configx"

type AppConfig struct {
  configx.BaseConfig                 // HTTP/HEALTH/METRICS/APP_CONFIGMAP_NAME...
  RateLimitQPS int `env:"RATE_LIMIT_QPS" default:"200"`
}
```

## 6.2 Main（单端口 H2C + 健康/指标）

```go
// backend/user/cmd/server/main.go
package main

import (
  "context"
  "net/http"
  "time"

  "go.eggybyte.com/egg/configx"
  "go.eggybyte.com/egg/connectx"
  "go.eggybyte.com/egg/runtimex"

  userv1 "go.eggybyte.com/eggybyte-foundation/gen/go/user/v1"
  "github.com/bufbuild/connect-go"
  "go.eggybyte.com/eggybyte-foundation/gen/go/user/v1/userv1connect"

  "go.eggybyte.com/eggybyte-foundation/backend/user/internal/config"
  "go.eggybyte.com/eggybyte-foundation/backend/user/internal/handler"
)

func main() {
  ctx := context.Background()
  var cfg config.AppConfig
  configx.MustLoad(&cfg)

  mux := http.NewServeMux()
  ints := connectx.DefaultInterceptors(connectx.Options{SlowRequestMillis: 1000})

  h := handler.NewUserHandler("user")
  path, svc := userv1connect.NewUserServiceHandler(h, connect.WithInterceptors(ints...))
  mux.Handle(path, svc)

  _ = runtimex.Run(ctx, nil, runtimex.Options{
    HTTP:    &runtimex.HTTPOptions{Addr: ":" + cfg.HTTPPort, H2C: true, Mux: mux},
    Health:  &runtimex.Endpoint{Addr: ":" + cfg.HealthPort},
    Metrics: &runtimex.Endpoint{Addr: ":" + cfg.MetricsPort},
    ShutdownTimeout: 15 * time.Second,
  })
}
```

## 6.3 Handler（示例 echo）

```go
// backend/user/internal/handler/user.go
package handler

import (
  "context"
  "time"

  "github.com/bufbuild/connect-go"
  userv1 "go.eggybyte.com/eggybyte-foundation/gen/go/user/v1"
)

type UserHandler struct{ serviceName string }

func NewUserHandler(serviceName string) *UserHandler { return &UserHandler{serviceName: serviceName} }

func (h *UserHandler) Ping(ctx context.Context, req *connect.Request[userv1.PingRequest]) (*connect.Response[userv1.PingResponse], error) {
  resp := &userv1.PingResponse{Message: req.Msg.GetMessage(), Service: h.serviceName, TsUnix: time.Now().Unix()}
  return connect.NewResponse(resp), nil
}
```

---

# 7. Proto 模板与 buf 生成

## 7.1 默认 `--proto echo`

```
api/user/v1/user.proto
```

```proto
syntax = "proto3";
package eggybyte_foundation.user.v1;
option go_package = "go.eggybyte.com/eggybyte-foundation/gen/go/user/v1;userv1";

service UserService {
  rpc Ping(PingRequest) returns (PingResponse);
}
message PingRequest { string message = 1; }
message PingResponse { string message = 1; string service = 2; int64 ts_unix = 3; }
```

## 7.2 可选 `--proto crud`

* `GetUser`、`ListUsers`、`CreateUser`、`UpdateUser`、`DeleteUser`，含标准分页。

## 7.3 `egg api init` / `egg api generate`

* `buf.yaml` 最小：`version: v2`、`name:` 指向你的仓库空间
* `buf.gen.yaml` 默认插件：

  * `buf.build/protocolbuffers/go`、`buf.build/connectrpc/go` → `gen/go`
  * `buf.build/protocolbuffers/dart` → `gen/dart`
  * `buf.build/grpc-ecosystem/openapiv2` → `gen/openapi`
  * 可选：`buf.build/bufbuild/es`（只出 TS pb types）

---

# 8. Compose 渲染（关键规则）

* **端口**：服务未显式定义 → 继承 `backend_defaults.ports`。
* **环境**：展开 `${cfgv:cm:key}`；禁止 `${cfg:}` 与 `${sec:}`。
* **数据库**：`database.enabled=true` → 自动附加 `mysql:9.4`，注入 `DATABASE_DSN`，并 `depends_on`。
* **镜像**：自动使用 `<docker_registry>/<project>-<service>:dev`。
* **示例片段**：

```yaml
services:
  user:
    image: ghcr.io/eggybyte-technology/eggybyte-foundation-user:dev
    container_name: user
    environment:
      HTTP_PORT: "8080"
      HEALTH_PORT: "8081"
      METRICS_PORT: "9091"
      RATE_LIMIT_QPS: "${FEATURE_A}"    # 由 compose 预先从 config 展开或 .env 提供
    ports: ["8080:8080","8081:8081","9091:9091"]
    depends_on: [mysql]                 # 若 database.enabled=true
```

---

# 9. Helm/Kubernetes 渲染（关键规则）

* **Service**：同时生成 `ClusterIP` 与 `Headless`（可选），名称取自 `kubernetes.service.*.name`。
* **Env 注入**：

  * `${cfg:cm}` → 注入 **名称**（如 `APP_CONFIGMAP_NAME=global-config`），应用据此动态监听。
  * `${sec:secret:key}` → `env.valueFrom.secretKeyRef`。
  * `${svc:name@headless}` → 注入服务名或 FQDN 片段（供客户端 SRV 解析）。
* **Probe**：liveness/readiness 指向 `HEALTH_PORT`；metrics 在 `METRICS_PORT`。
* **镜像**：统一 `<docker_registry>/<project>-<service>:<tag>`。
* **资源**：`kubernetes.resources.{configmaps,secrets}` 自动生成/更新（ConfigMap 仅 K/V；Secret base64）。

---

# 10. 构建与发布

* `egg build --version v0.2.0 --push`

  * **buildx** 多平台（来自 `build.platforms`）
  * Tag：`v0.2.0` + `git-sha-7`（可选）
  * 镜像：每个服务各自 `<project>-<service>`
* `egg build --subset user,order` 仅构建指定服务。

---

# 11. 校验（`egg check`）

* 服务名不得以 `-service` 结尾（报错并建议 `user`）。
* 端口冲突/端口与默认值重复（建议删除重复定义，保持最简）。
* `${cfgv}` 在 K8s 目标被检测到（报错）。
* 未声明的 `configmaps/secrets` 被引用（报错）。
* `buf` 包/`go_package` 与 `module_prefix` 不一致（报错+修复建议）。
* 镜像名是否按 `<project>-<service>` 规范生成（强检）。

---

# 12. 典型工作流（5 分钟）

```bash
egg init eggybyte-foundation && cd eggybyte-foundation

egg create backend user               # echo proto + handler + go.work use
egg api init && egg api generate

egg compose up --detached             # /health /metrics /connect RPC 可用

egg build --version v0.2.0 --push     # ghcr.io/.../eggybyte-foundation-user:v0.2.0

egg kube template -n dev | kubectl apply -f -
```

---

# 13. 兼容与迁移

* 已有以 `-service` 结尾的项目：

  * `egg check` 提示并提供 `egg migrate rename --from user-service --to user` 的文件/路径/包名重写方案（不强制自动执行）。
* 旧字段（如 `grpc` 多端口）继续读取但**发出 warning**，渲染按 Connect 单端口落地。

---

# 14. CLI 内部实现要点

* `configschema`：解析/默认值填充（端口继承、镜像名计算）、强校验（命名/表达式/引用/端口）。
* `projectfs`：幂等写入骨架；**绝不**直接改 `go.work` 内容，统一走命令：

  * `go work init`（如未初始化）
  * `go work use ./backend/<name>`
  * `cd backend/<name> && go mod init <module_prefix>/backend/<name>`
* `toolrunner`：封装 `go|buf|docker|helm|kubectl|flutter`，输出统一、带修复建议。
* `generators`：`Backend.Create`（校名→骨架→proto→注册→buf generate）、`API.Init|Generate`、`Frontend.Create`。
* `render`：Compose（允许 `${cfgv}` 解值）、Helm（名称法/secretKeyRef/SVC 类型）。
* `lint/check`：一处产出所有告警与错误，支持 `--fix` 对无损项自动修复。
* `ui`：标准与 `--json` 机器可读模式。

---

# 15. 设计小结（与你的要求逐条对齐）

* ✅ **后端服务不以 `-service` 结尾**，CLI 强校验与迁移建议。
* ✅ **镜像名自动：`项目名-服务名`**，统一构建、Compose、Helm 三端引用。
* ✅ **最小配置可运行**：`egg create backend user` 默认生成 Echo Proto + Handler，`egg compose up` 即可打通链路。
* ✅ **Connect-first / 单端口**、**K8s 名称法注入**、**MySQL 9.4 一键联调**、**buf 统一生成**、**go work 命令式维护** 全部按 v0.2.0 规范落地。