# egg v0.1.0——完整架构与落地设计方案（最终版）

> 目标：以“最少样板代码 + 最强默认安全/观测 + 明确依赖边界”的方式，为 Go/Connect/Kubernetes 微服务提供一个**可用、可爱、可信**的通用库体系。本文为最终版设计文档，直接用于落地与团队协作。

---

## 1. 设计宗旨与约束

### 1.1 设计宗旨

* **工程化可落地**：所有能力均能以最少 API 调用启用，并有明确的默认值与回退策略。
* **单向依赖**：模块遵守自下而上的单向依赖（DAG），杜绝环依赖与跨层耦合。
* **生产就绪**：内置超时/熔断/重试、严格配置校验、健康检查聚合、默认安全头、结构化错误详情。
* **观测一致**：日志/指标/追踪三位一体，字段命名统一，便于中台运营。
* **K8s 友好**：ConfigMap“名称法”动态配置、Secret 仅通过 `secretKeyRef` 注入、健康/就绪探针清晰可用。

### 1.2 全局约束

* Go 1.23+；`golangci-lint`、`go test -race -cover`、`gorelease`、`govulncheck` 必须通过。
* 日志采用**单行 logfmt**；仅 `level` 字段着色；字段**按 key 字母序**排序，确保 diff 稳定。
* 默认**单端口**承载 HTTP/Connect/gRPC-Web（h2/h2c），健康/指标为独立端口。
* 鉴权交给网关（如 Higress）进行，服务内只做**权限判定**（基于上下文 `identity`）。

---

## 2. 分层与依赖（L0→L4）

```
L0  core         // 零依赖：接口、错误、身份、上下文键、轻量工具
│
L1  logx         // 日志实现：单行 logfmt、仅 level 彩色、字段排序/截断
│
L2  configx      // 多源配置 & 热更（K8s 名称法） + 严格校验
L2  obsx         // OTel/Prom 观测初始化（Tracer/Meter/Exporter）
L2  httpx        // HTTP 工具：Bind & Validate、默认安全头、404/405
│
L3  runtimex     // 运行时：统一端口(h2/h2c)、健康聚合、/debug/vars
L3  connectx     // Connect 绑定/拦截器：超时、日志、追踪/指标、错误映射
L3  clientx      // Connect 客户端工厂：重试、熔断、超时、指标、幂等
│
L4  servicex     // 集成器：单函数启动 + 函数式选项，轻量 DI，关闭栈
```

**依赖规则**：仅允许指向更低层；如 `servicex → {connectx,runtimex} → {configx,obsx,logx} → core`。

---

## 3. 仓库与工作区结构（Monorepo + 多模块）

```
github.com/eggybyte-technology/egg
├─ go.work
├─ docs/
│  ├─ ARCHITECTURE.md
│  └─ LOGGING.md
├─ core/        (module: egg/core)
├─ logx/        (module: egg/logx)
├─ configx/     (module: egg/configx)
├─ obsx/        (module: egg/obsx)
├─ httpx/       (module: egg/httpx)
├─ runtimex/    (module: egg/runtimex)
├─ connectx/    (module: egg/connectx)
├─ clientx/     (module: egg/clientx)
├─ servicex/    (module: egg/servicex)
├─ storex/      (module: egg/storex)         # 可选示范（事务/健康）
├─ testingx/    (module: egg/testingx)
└─ examples/
   ├─ minimal-connect-service/
   └─ user-service/
```

> 根 `go.work` 仅 `use` 各子模块与 examples；按子模块打 tag，对外 `go get github.com/.../servicex@servicex/v0.1.0`。

---

## 4. 核心规范（“默认即最佳实践”）

### 4.1 日志规范（logfmt + 仅 level 彩色 + 排序）

* 单行 logfmt，如：

  ```
  level=INFO msg="user created" req_id="r-123" user_id="u-9" cost_ms=12
  ```
* 仅 `level` 彩色；CLI/本地默认彩色，容器/采集可关闭。
* 除 `level`/`msg` 外，其余字段按 **key 字母序**排序，保证日志 diff 稳定。
* 大 payload 仅记录大小与哈希（可选），避免泄露敏感信息。

### 4.2 端口与协议

* HTTP/Connect/gRPC-Web 默认单端口（h2/h2c 自动协商）。
* `/healthz` 与 `/metrics` 固定独立端口，避免自检/观测阻塞业务端口。
* `/debug/vars` 提供编译信息、启动时间与配置快照（敏感字段脱敏）。

### 4.3 配置

* 基线：**环境变量 > 文件（可选） > K8s ConfigMap 名称法**（覆盖“动态键”）。
* 校验：集成 `validator`，启动与热更均进行严格校验，失败则拒绝生效并保留旧快照。
* Secret：仅通过 `env.valueFrom.secretKeyRef` 注入，不由库直接读取。

### 4.4 身份与权限

* 网关完成鉴权与身份注入（请求头）；服务端从上下文读取 `identity.UserInfo`，仅做**权限判定**。
* 上下文键使用**非导出的强类型**，杜绝冲突。

### 4.5 关闭顺序

* 按 LIFO 关闭栈：先停 listener/拒绝新请求 → 等待飞行请求 → 关闭下游依赖 → Flush 观测 → 释放日志。

---

## 5. 模块规格与 API

### 5.1 `core`（零依赖，稳定契约）

* **errors**：

  ```go
  type Code string
  const (
    CodeInvalid Code = "INVALID_ARGUMENT"
    CodeNotFound Code = "NOT_FOUND"
    CodeUnauth  Code = "UNAUTHENTICATED"
    CodeDenied  Code = "PERMISSION_DENIED"
    CodeExhaust Code = "RESOURCE_EXHAUSTED"
    CodeInternal Code = "INTERNAL"
    CodeUnavailable Code = "UNAVAILABLE"
  )

  type E struct {
    Code    Code
    Msg     string
    Err     error
    Op      string
    Details []any // 可放 PB message 作为结构化细节
  }

  func Wrap(err error) *ErrorBuilder // Fluent 构建器：WithOp/WithCode/WithDetails/Build
  ```
* **identity**：

  ```go
  type UserInfo struct { ID string; Roles []string; Internal bool }
  func WithUser(ctx context.Context, u *UserInfo) context.Context
  func UserFrom(ctx context.Context) (*UserInfo, bool)
  func HasRole(ctx context.Context, role string) bool
  ```
* **log**：

  ```go
  type Logger interface {
    With(kv ...any) Logger
    Debug(msg string, kv ...any)
    Info(msg string, kv ...any)
    Warn(msg string, kv ...any)
    Error(msg string, kv ...any)
  }
  ```

### 5.2 `logx`（日志实现）

* `New(opts ...Option) Logger`；可选项：`WithFormat("logfmt"|"json")/WithColor(bool)/WithLevel(...)`。
* 字段排序、单行保证、敏感字段掩码、payload 截断计量（可选）。
* `FromContext(ctx)`：注入 `trace_id/request_id/user_id` 等。

### 5.3 `configx`（配置聚合与热更 + 严格校验）

* `Manager`：

  ```go
  type Manager interface {
    Bind(target any, opts ...BindOption) error
    OnUpdate(func(snapshot any)) // 触发后校验→部分热更
    Snapshot() any               // 只读快照（脱敏）
  }
  ```
* 选项：`WithValidator(*validator.Validate)`、`WithDynamicKeys(map[string]struct{})`、`WithLogger(log.Logger)`。
* K8s 名称法：仅注入 `*_CONFIGMAP_NAME`，由库 watch 对应 ConfigMap 的动态键并热更。
* `BaseConfig`：`HTTP_PORT/HEALTH_PORT/METRICS_PORT/OTLP_ENDPOINT/LOG_*` 等。

### 5.4 `obsx`（观测初始化）

* `NewProvider(ctx, opts...) (TracerProvider, MeterProvider, Shutdown)`；
* 指标前缀：`rpc.server.* / rpc.client.* / process.*`；
* 选项：`WithOTLPEndpoint/WithSamplerRatio/WithServiceName/WithVersion`。

### 5.5 `httpx`（HTTP 工具）

* `BindAndValidate(r *http.Request, dst any) error`（JSON → struct + validator）。
* 默认安全头中间件：`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Strict-Transport-Security`（可配置）。
* 统一 `NotFound`/`MethodNotAllowed` JSON 响应；可被 `runtimex.Options` 覆盖。

### 5.6 `runtimex`（运行时）

* `Run(ctx, httpCfg, healthCfg, metricsCfg, mux, opts...) error`；
* 选项：`WithH2C(true)`、`WithSecureHeaders(true)`、`WithReadHeaderTimeout/WithIdleTimeout`；
* **健康聚合**：

  ```go
  type HealthChecker interface {
    Name() string
    Check(ctx context.Context) error
  }
  func RegisterHealthChecker(h HealthChecker)
  ```
* `/healthz` 聚合：任一子检查失败 → 503（供 Readiness 使用）。
* `/debug/vars`：构建信息/启动时间/配置快照（脱敏）。

### 5.7 `connectx`（服务端拦截器族与绑定）

* `DefaultInterceptors(opts...) []connect.Interceptor`：

  1. **RecoveryInterceptor**：捕获 panic 并转换为 INTERNAL 错误，记录日志。
  2. **TimeoutInterceptor（服务级配置）**：使用 `Options.DefaultTimeoutMs` 作为默认超时；
     * 支持请求头 `X-RPC-Timeout-Ms` **下调**（不得超过服务端上限）。
     * 在 handler 前使用 `context.WithTimeout`。
     * **注意**：超时配置在微服务级别（非 Protobuf MethodOptions 扩展）。
  3. **IdentityInterceptor**：从请求头提取用户身份与元数据，注入 context。
  4. **ErrorMappingInterceptor**：`core/errors.Code ↔ connect.Code`，支持完整错误码映射（包括 `RESOURCE_EXHAUSTED`、`UNIMPLEMENTED` 等）。
  5. **LoggingInterceptor**：上下文感知 logger，打印 `service/method/status/latency/payload_bytes` 等；支持慢请求告警。
* `HeaderMapping`：解析 `X-Request-Id`, `X-User-Id`, `X-User-Roles`, `X-Real-IP` 等注入 `identity`。
* `Bind(mux, path, handler)`：注册工具。

**配置示例**：

```go
interceptors := connectx.DefaultInterceptors(connectx.Options{
  Logger:            logger,
  Otel:              otelProvider,
  DefaultTimeoutMs:  30000, // 30秒默认超时
  EnableTimeout:     true,
  SlowRequestMillis: 1000,  // 慢请求阈值
})
```

### 5.8 `clientx`（客户端工厂）

* `NewHTTPClient(baseURL string, opts...)` 返回已注入拦截器的 Connect 客户端：

  * **RetryInterceptor**：指数退避+抖动，仅对幂等/可判定错误重试（如 `UNAVAILABLE/503`）；
  * **CircuitBreakerInterceptor**：按服务维度统计失败率/半开窗口（可用 `sony/gobreaker`）；
  * **TimeoutInterceptor**：客户端侧缺省上限，支持方法级 override；
  * **MetricsInterceptor**：`rpc.client.*` 指标；
  * 幂等性：支持 `X-Idempotency-Key`/方法配置识别。

### 5.9 `servicex`（集成器 + 轻量 DI + 关闭栈）

* **单函数启动（函数式选项）**：

  ```go
  err := servicex.Run(ctx,
    servicex.WithService("greeter", "0.1.0"),
    servicex.WithConfig(&AppConfig{}),
    servicex.WithTracing(true),
    servicex.WithMetrics(true),
    servicex.WithRegister(func(app *servicex.App) error {
      // 依赖注入
      app.Provide(repository.NewUserRepo)
      app.Provide(service.NewUserService)
      app.Provide(handler.NewUserHandler)
      var h *handler.UserHandler
      _ = app.Resolve(&h)

      path, ih := greeterv1connect.NewGreeterHandler(
        h, connect.WithInterceptors(app.Interceptors()...),
      )
      app.Mux().Handle(path, ih)
      return nil
    }),
  )
  ```
* **DI 容器**：`Provide(constructor)` / `Resolve(&T)`，按照构造函数参数自动拓扑构建与缓存。
* **关闭栈**：`AddShutdownHook(fn)`；`Run` 退出时 LIFO 依次执行，带超时控制。
* **拦截器出厂链**：`app.Interceptors()` 返回平台链，业务可追加。

### 5.10 `storex`（可选：事务助手与健康）

* `WithTransaction(ctx, fn)` 自动 commit/rollback，并把 tx 放入 ctx，仓储从 ctx 取用。
* `HealthChecker` 实现：DB ping 超时/失败即报错。

### 5.11 `testingx`（测试助手）

* `NewMockLogger(t)`：捕获日志并断言；
* `NewContextWithIdentity(t, userInfo)`：快速构造带身份 ctx；
* `AssertError(t, err, expectedCode)`：校验 `core/errors.Code`；
* `configtest.MockManager`：注入期望配置与热更事件；
* `StartTestServer(tb, registerFn)`：内存自启服务端做集成测试。

---

## 6. 统一标准（日志/错误/指标/安全/性能）

### 6.1 错误码与映射（示例表）

| core/errors.Code   | Connect Code      | HTTP Status |
| ------------------ | ----------------- | ----------- |
| INVALID_ARGUMENT   | InvalidArgument   | 400         |
| NOT_FOUND          | NotFound          | 404         |
| UNAUTHENTICATED    | Unauthenticated   | 401         |
| PERMISSION_DENIED  | PermissionDenied  | 403         |
| RESOURCE_EXHAUSTED | ResourceExhausted | 429         |
| UNAVAILABLE        | Unavailable       | 503         |
| INTERNAL           | Internal          | 500         |

> `*errors.E.Details` 将作为 Connect ErrorDetails（PB message）返回；服务端日志打印 `op/code/trace_id` 等关键字段。

### 6.2 指标命名（示例）

* 服务器：`rpc.server.requests_total{service,method,code}`、`rpc.server.duration_ms_bucket`、`rpc.server.payload_bytes`；
* 客户端：`rpc.client.requests_total`、`rpc.client.duration_ms`、`rpc.client.retries_total`；
* 资源：`process.cpu.percent`、`process.mem.bytes`。

### 6.3 默认安全

* 默认启用安全头；在 Ingress/HSTS 已统一时可关闭或调整；
* 日志/指标避免敏感数据；大字段做长度阈值截断并记录 `payload_bytes`。

### 6.4 性能建议

* `ReadHeaderTimeout/IdleTimeout` 结合网关超时设置；
* h2c 在内网默认开启；
* 客户端重试/熔断仅对幂等或可识别的瞬时错误开放。

---

## 7. Kubernetes 实践

### 7.1 环境变量与 ConfigMap/Secret

* 仅注入 `APP_CONFIGMAP_NAME=my-svc-dyn`；ConfigMap 只包含“动态键”（如 `RATE_LIMIT_QPS`, `SLOW_REQUEST_MILLIS`）。
* Secret 通过 `env.valueFrom.secretKeyRef` 注入，例如 DB 密码。
* 端口：`HTTP_PORT`、`HEALTH_PORT`、`METRICS_PORT` 来源于 `BaseConfig`。

**Deployment 片段**：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata: { name: greeter }
spec:
  template:
    spec:
      containers:
      - name: greeter
        image: ghcr.io/eggybyte-technology/greeter:0.1.0
        env:
        - name: HTTP_PORT
          value: "8080"
        - name: HEALTH_PORT
          value: "8081"
        - name: METRICS_PORT
          value: "9091"
        - name: APP_CONFIGMAP_NAME
          value: "greeter-dyn"
        - name: DB_URL
          valueFrom:
            secretKeyRef:
              name: greeter-secret
              key: db_url
```

### 7.2 探针与服务

* `readinessProbe` 命中 `/healthz` 聚合；`livenessProbe` 命中轻量自检；
* Service：业务端口（ClusterIP），健康/指标内部访问；网关接入承载外部流量。

---

## 8. 从零到“一行起服”示例

### 8.1 Proto（Per-RPC 超时）

```proto
syntax = "proto3";
package demo.v1;

import "google/protobuf/descriptor.proto";
extend google.protobuf.MethodOptions { int64 timeout_ms = 50001; }

message SayRequest { string name = 1; }
message SayResponse { string text = 1; }

service Greeter {
  rpc Say(SayRequest) returns (SayResponse) { option (timeout_ms) = 250; }
}
```

### 8.2 main.go（servicex + DI + 出厂拦截器）

```go
type AppConfig struct {
  configx.BaseConfig
  SlowRequestMillis int64  `env:"SLOW_REQUEST_MILLIS" validate:"min=0,max=60000"`
  DatabaseURL       string `env:"DB_URL" validate:"required,url"`
}

func main() {
  ctx := context.Background()
  _ = servicex.Run(ctx,
    servicex.WithService("greeter", "0.1.0"),
    servicex.WithConfig(&AppConfig{}),
    servicex.WithTracing(true),
    servicex.WithMetrics(true),
    servicex.WithRegister(func(app *servicex.App) error {
      // 依赖注入（示例）
      app.Provide(repository.NewGormUserRepository)
      app.Provide(service.NewUserService)
      app.Provide(handler.NewUserHandler)
      var h *handler.UserHandler
      _ = app.Resolve(&h)

      path, ih := demov1connect.NewGreeterHandler(
        h, connect.WithInterceptors(app.Interceptors()...),
      )
      app.Mux().Handle(path, ih)
      return nil
    }),
  )
}
```

---

## 9. CI/CD 与质量门槛

* **Lint/Tests**：`golangci-lint run`、`go test -race -cover`（≥80%）、`go vet`。
* **兼容性**：`gorelease` 检查导出 API 变更；`govulncheck` 供应链扫描。
* **模块发版**：按子模块打 tag（`core/v0.1.0`、`servicex/v0.1.0` 等）；在 CI 中基于 `path` 变更选择性构建与发布。
* **示例与文档**：`examples/` 可跑通；`docs/ARCHITECTURE.md`、`docs/LOGGING.md` 与模块 `README.md` 同步更新。

---

## 10. 开发守则（团队统一）

1. **接口分级**：导出符号标注 `// Stable`/`// Experimental`；稳定面除非重大问题不得破坏兼容。
2. **日志口径**：所有服务必须使用 `logx`；业务日志通过 `logx.FromContext(ctx)` 取 logger；严禁打印敏感数据。
3. **错误处理**：统一使用 `core/errors`；对外返回 Connect 错误并附加结构化 Details；对内日志必须含 `op/code/trace_id`。
4. **配置热更**：仅动态键允许热更；热更必须通过 `configx`，并在回调中有界重建相关对象（如限流器）。
5. **拦截器链**：优先使用 `connectx.DefaultInterceptors()`；业务可使用 `connectx.Chain()` 追加。
6. **超时/重试/熔断**：方法级超时为强制；客户端重试/熔断仅对幂等/瞬时错误启用；禁止级联重试导致放大。
7. **观测埋点**：出现慢请求（如 >1s）必须记录 `payload_bytes/latency` 并打点；异常路径必须带错误码与原因。
8. **关闭顺序**：所有资源注册 `AddShutdownHook`；严禁在 `defer` 中静默吞错。

---

## 11. 附录

### 11.1 常用环境变量（示例）

* `HTTP_PORT=8080`
* `HEALTH_PORT=8081`
* `METRICS_PORT=9091`
* `APP_CONFIGMAP_NAME=my-svc-dyn`
* `OTLP_ENDPOINT=http://otel-collector:4317`
* `LOG_LEVEL=INFO`、`LOG_FORMAT=logfmt|json`、`LOG_COLOR=true|false`

### 11.2 日志样例

```
level=INFO msg="handlers registered" service="greeter" version="0.1.0" http_port=8080 health_port=8081 metrics_port=9091
level=INFO msg="rpc ok" service="demo.v1.Greeter" method="Say" status="OK" cost_ms=7 req_id="r-123" user_id="u-9"
```

### 11.3 指标样例（Prom）

```
rpc_server_requests_total{service="demo.v1.Greeter",method="Say",code="OK"} 1
rpc_server_duration_ms_bucket{service="demo.v1.Greeter",method="Say",le="5"} 0
...
```

---

> **一句话总结**：`egg v0.1.0` 以 L0→L4 分层为骨架，提供日志/配置/观测/运行时/连接栈/客户端/集成器的**端到端闭环**；默认即最佳实践——一行起服、观测齐备、超时/熔断/重试、安全/校验/健康检查全开箱，既适合快速孵化，也经得起生产检验。
