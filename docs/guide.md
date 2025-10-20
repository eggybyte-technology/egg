# 1) 愿景与基线原则

* **极薄内核 + 可插拔卫星库**：只引入需要的模块，最小依赖、最快构建。
* **Connect-first**：统一拦截器栈（恢复、日志、追踪、指标、身份注入），0 业务侵入。
* **统一端口策略**：默认**单端口**承载 HTTP/Connect/gRPC-Web，**健康/指标独立端口**。
* **K8s “名称法”**：ConfigMap 仅注入**名称**，运行时监听并热更新；Secret 用 `secretKeyRef`，服务发现区分 `headless/clusterip`。
* **稳定 API**：`core` 与 `runtimex` 尽量稳定；其余模块小步快跑。

---

# 2) 仓库与模块布局

**仓库**：`github.com/eggybyte-technology/egg`

```
egg/
├─ go.work
├─ README.md
├─ docs/
│  ├─ ARCHITECTURE.md
│  ├─ RELEASING.md
│  └─ CONTRIBUTING.md
├─ core/        # L1：零依赖的接口与通用工具（稳定）
│  └─ go.mod -> module github.com/eggybyte-technology/egg/core
├─ runtimex/    # L2：运行时（生命周期/服务器/健康/指标/基础配置；不含 Connect/K8s）
│  └─ go.mod -> module github.com/eggybyte-technology/egg/runtimex
├─ connectx/    # L3：Connect 绑定 + 统一拦截器 + 身份注入（无鉴权）
│  └─ go.mod -> module github.com/eggybyte-technology/egg/connectx
├─ configx/     # L3：统一配置（Env/File + K8s ConfigMap 热更新）
│  └─ go.mod -> module github.com/eggybyte-technology/egg/configx
├─ obsx/        # L3：OpenTelemetry/Prometheus 初始化
│  └─ go.mod -> module github.com/eggybyte-technology/egg/obsx
├─ k8sx/        # L3：ConfigMap 名称法监听、服务发现、Secret 契约
│  └─ go.mod -> module github.com/eggybyte-technology/egg/k8sx
├─ storex/      # L3：TiDB/MySQL/GORM、仓库注册与健康探针（可选）
│  └─ go.mod -> module github.com/eggybyte-technology/egg/storex
└─ examples/
   └─ minimal-connect-service/   # 最小可运行服务示例（独立 go.mod）
```

**依赖方向（只许向下）**
`core → runtimex → {connectx, obsx}`；`{configx, k8sx, storex}` 仅依赖 `core`（`configx` 可选使用 `k8sx` 提供的 ConfigMap 监听）。
禁止反向依赖（例如 `core` 绝不 import `connectx`）。

**go.work（根）**

```go
go 1.23
use (
  ./core
  ./runtimex
  ./connectx
  ./configx
  ./obsx
  ./k8sx
  ./storex
  ./examples/minimal-connect-service
)
```

---

# 3) 模块职责与对外 API（精简而完备）

## 3.1 `core`（零依赖，极稳定）

* `log`（接口，与 slog 思想兼容）

  ```go
  package log
  type Logger interface {
    With(kv ...any) Logger
    Debug(msg string, kv ...any)
    Info(msg string, kv ...any)
    Warn(msg string, kv ...any)
    Error(err error, msg string, kv ...any)
  }
  // 快捷键值（建议用法；也可直接使用 slog.Attr 风格）
  func Str(k, v string) any
  func Int(k string, v int) any
  func Dur(k string, v time.Duration) any
  ```
* `errors`（与标准库兼容的分层错误）

  ```go
  package errors
  type Code string // e.g. "INVALID_ARGUMENT", "NOT_FOUND", "INTERNAL"
  type E struct{ Code Code; Op string; Err error; Msg string }
  func New(code Code, msg string) error
  func Wrap(code Code, op string, err error) error
  func CodeOf(err error) Code
  ```
* `identity`（仅做“身份容器”，不做鉴权）

  ```go
  package identity
  type UserInfo struct {
    UserID, UserName, Tenant string
    Roles []string
  }
  type RequestMeta struct {
    RequestID, InternalToken string
    RemoteIP, UserAgent      string
  }
  func WithUser(ctx context.Context, u *UserInfo) context.Context
  func UserFrom(ctx context.Context) (*UserInfo, bool)
  func WithMeta(ctx context.Context, m *RequestMeta) context.Context
  func MetaFrom(ctx context.Context) (*RequestMeta, bool)
  ```
* `utils`：时间/重试/切片/并发 helpers（真通用，慎增）。

## 3.2 `runtimex`（运行时内核，不含 Connect/K8s）

* 生命周期编排、统一端口策略、独立健康/指标端口、基础 env→struct 配置。

  ```go
  package runtimex
  type Service interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
  }

  type Endpoint struct{ Addr string } // ":8081" 等
  type HTTPOptions struct{
    Addr string
    H2C  bool
    Mux  *http.ServeMux
  }
  type Options struct {
    Logger          log.Logger
    HTTP            *HTTPOptions // Connect-only：单端口承载 HTTP/Connect/gRPC-Web（h2/h2c）
    Health, Metrics *Endpoint
    ShutdownTimeout time.Duration
  }
  func Run(ctx context.Context, svcs []Service, opts Options) error
  ```

> 约定：**健康/指标端口永远独立**；默认单端口承载 HTTP/Connect/gRPC-Web（h2/h2c）。

## 3.3 `connectx`（Connect 绑定 + 统一拦截器 + 身份注入）

* **默认从 Higress 注入的请求头提取身份**，注入 `core/identity`，不做鉴权。

* 统一拦截器（恢复、日志、追踪、指标、错误映射、身份注入）。

  ```go
  package connectx

  type HeaderMapping struct {
    RequestID      string // "X-Request-Id"
    InternalToken  string // "X-Internal-Token"
    UserID         string // "X-User-Id"
    UserName       string // "X-User-Name"
    Tenant         string // "X-User-Tenant"
    Roles          string // "X-User-Roles"
    RealIP         string // "X-Real-IP"
    ForwardedFor   string // "X-Forwarded-For"
    UserAgent      string // "User-Agent"
  }

  type Options struct {
    Logger             log.Logger
    Otel               *obsx.Provider // nil 时禁用 trace/metrics
    Headers            HeaderMapping  // 可覆盖默认映射
    WithRequestBody    bool           // 生产默认 false
    WithResponseBody   bool
    SlowRequestMillis  int64          // 慢请求阈值
    PayloadAccounting  bool           // 记录入出站字节
  }

  func DefaultInterceptors(o Options) []connect.Interceptor

  // 绑定工具：把 protoc-gen-connect-go 生成的 handler 路由到 mux
  func Bind(mux *http.ServeMux, path string, h http.Handler)
  ```

* **错误映射（建议）**
  `core/errors` → Connect `Code` → HTTP：

  * `INVALID_ARGUMENT` → `CodeInvalidArgument` → 400
  * `NOT_FOUND` → `CodeNotFound` → 404
  * `ALREADY_EXISTS` → 409
  * `INTERNAL`/默认 → 500
    （如需 `PERMISSION_DENIED/UNAUTHENTICATED` 也可映射，但本框架不做校验）

* **日志字段口径（最低集）**
  `ts, level, service, version, env, instance, trace_id, span_id, req_id, rpc_system=connect, rpc_service, rpc_method, status, latency_ms, remote_ip, user_agent, payload_in, payload_out`

## 3.4 `configx`（统一配置：Env/File + K8s ConfigMap 热更新）

```go
package configx

// Source 描述一个配置来源（环境变量、文件、内存镜像、K8s ConfigMap）。
// 实现需保证线程安全，并在更新时向外发布快照。
type Source interface {
  // Load 读取当前配置快照（不可变 Map），用于启动时的初始合并。
  Load(ctx context.Context) (map[string]string, error)
  // Watch 启动更新监听；每次配置变更时，通过返回的 chan 发布最新快照。
  // 实现应在 ctx 取消时退出且关闭 chan，避免 goroutine 泄漏。
  Watch(ctx context.Context) (<-chan map[string]string, error)
}

// Manager 管理多来源配置，提供快照读取与热更新广播。
// 合并策略：后加入的来源优先级更高（后写覆盖先写）。
type Manager interface {
  // Snapshot 返回最近一次合并后的全量配置副本。
  Snapshot() map[string]string
  // Value 返回某 key 的值与是否存在。
  Value(key string) (string, bool)
  // Bind 将配置解码到结构体；支持 env 标签与默认值；当配置更新时可选触发回调。
  Bind(target any, opts ...BindOption) error
  // OnUpdate 订阅更新事件（去抖与节流可选），回调在独立 goroutine 执行，需考虑超时与并发安全。
  OnUpdate(fn func(snapshot map[string]string)) (unsubscribe func())
}

// Options 配置管理器选项。
type Options struct {
  Logger       log.Logger
  Sources      []Source
  Debounce     time.Duration // 更新合并去抖时长（默认 200ms）
}

// NewManager 构造配置管理器；会依次加载各来源并开始监听热更新。
func NewManager(ctx context.Context, o Options) (Manager, error)

// 便捷来源实现（建议）：
//   - EnvSource: 从进程环境变量读取（可配置前缀、大小写规则）
//   - FileSource: 从本地 JSON/YAML/TOML 文件读取（可选热加载）
//   - K8sConfigMapSource: 通过 k8sx.WatchConfigMap 监听指定名称的 ConfigMap，并在数据变更时发布快照

// 键名建议：统一使用 SNAKE_CASE，例如 HTTP_PORT、METRICS_PORT、OTEL_EXPORTER_OTLP_ENDPOINT
```

**模式与推荐实践**：

- **Env-only 模式（本地/非 K8s）**：仅使用 `EnvSource`，所有配置从环境变量读取；适合容器外或简单部署。
- **K8s 动态模式（名称法）**：
  - 静态基线从环境变量读取（容器 `env` 或 `secretKeyRef` 注入）。
  - 通过设置 `APP_CONFIGMAP_NAME`，启用 `K8sConfigMapSource` 按名称监听并发布动态配置（仅覆盖“动态键”）。
  - 建议对动态键采用明确前缀或名单（例如 `APP_`、`FEATURE_`），避免误覆盖静态端口/身份等关键参数。

**多 ConfigMap 来源（按需组合）**：

- 支持通过多个环境变量传入不同职能的 ConfigMap 名称，例如：
  - `APP_CONFIGMAP_NAME`：应用级动态配置（推荐仅动态键）；
  - `CACHE_CONFIGMAP_NAME`：缓存相关开关与限额；
  - `ACL_CONFIGMAP_NAME`：访问控制名单等；
  - 也可按约定识别任意 `*_CONFIGMAP_NAME` 键。
- 合并优先级为“后加入覆盖先加入”。建议顺序：Env 基线 → 应用级 → 领域级（如 Cache/ACL）。
- 推荐使用键前缀进行“命名空间”隔离（如 `CACHE_`/`ACL_`），并仅允许这些前缀的键被动态覆盖。

```go
// 多 ConfigMap 来源构建示例：识别若干显式环境变量。
func buildSources(ctx context.Context, logger log.Logger) ([]configx.Source, error) {
  env := configx.NewEnvSource(configx.EnvOptions{Prefix: ""})
  sources := []configx.Source{env}

  names := []string{
    os.Getenv("APP_CONFIGMAP_NAME"),   // 应用级
    os.Getenv("CACHE_CONFIGMAP_NAME"), // 缓存域
    os.Getenv("ACL_CONFIGMAP_NAME"),   // 访问控制域
  }
  for _, name := range names {
    if name == "" { continue }
    s := configx.NewK8sConfigMapSource(name, configx.K8sOptions{Namespace: os.Getenv("NAMESPACE")})
    sources = append(sources, s)
  }
  return sources, nil
}
```

**基础配置基类（建议）**：为所有微服务提供稳定且可继承的静态配置集合（仅来自环境变量）。

```go
// BaseConfig 聚合服务识别、端口与观测等静态参数，仅从环境变量读取。
type BaseConfig struct {
  ServiceName     string `env:"SERVICE_NAME" default:"app"`
  ServiceVersion  string `env:"SERVICE_VERSION" default:"0.0.0"`
  Env             string `env:"ENV" default:"dev"`

  // Connect-only：单端口承载 HTTP/Connect/gRPC-Web（h2/h2c）。
  HTTPPort        string `env:"HTTP_PORT" default:":8080"`
  HealthPort      string `env:"HEALTH_PORT" default:":8081"`
  MetricsPort     string `env:"METRICS_PORT" default:":9091"`

  // 观测与动态配置开关
  OTLPEndpoint    string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" default:""`
  ConfigMapName   string `env:"APP_CONFIGMAP_NAME" default:""` // 为空则 Env-only
  DebounceMillis  int    `env:"CONFIG_DEBOUNCE_MS" default:"200"`
}

// 业务服务在其强类型配置上“继承”（通过匿名字段嵌入）BaseConfig，
// 并新增自己的业务键；业务键可在 K8s 动态模式下由 ConfigMap 覆盖。
type AppConfig struct {
  BaseConfig

  // 动静结合：Env 提供默认值；若启用 K8s 模式，则可被 ConfigMap 覆盖
  SlowRequestMillis int64 `env:"SLOW_REQUEST_MILLIS" default:"1000"`
  RateLimitQPS      int   `env:"RATE_LIMIT_QPS" default:"100"`
  // FeatureFlags 等复杂结构建议使用 JSON 字符串或多键前缀展开
}
```

**典型使用方式**：

1. 根据是否存在一个或多个 `*_CONFIGMAP_NAME` 环境变量决定模式：
   - 若均为空 → Env-only：仅加入 `EnvSource`。
   - 若任一非空 → K8s 动态模式：`EnvSource` + 若干 `K8sConfigMapSource(name)`；仅动态键会被覆盖。
2. 使用 `Bind(&cfg)` 将快照解码到强类型配置；在 `OnUpdate` 回调中原子重建受动态键影响的依赖（如限流器、白名单、开关）。
3. 对敏感值遵循“Secret 不入库”：由应用通过 env + `secretKeyRef` 注入，`configx` 不直接读取 Secret。

**合并与优先级**（默认从低到高）
- 低：`FileSource`（如存在）
- 中：`EnvSource`（容器环境变量）
- 高：`K8sConfigMapSource`（仅覆盖约定的“动态键”）

> 推荐：配置键尽量稳定并文档化；避免频繁动态变更导致抖动，必要时使用 `Debounce` 去抖并在消费者侧做最小化重载。

## 3.5 `obsx`（OpenTelemetry/Prometheus 初始化）

```go
package obsx
type Options struct {
  ServiceName, ServiceVersion string
  OTLPEndpoint string // "otel-collector:4317"
  EnableRuntimeMetrics bool
  ResourceAttrs map[string]string
  TraceSamplerRatio float64 // 0.0~1.0
}
type Provider struct {
  TracerProvider *sdktrace.TracerProvider
  MeterProvider  *sdkmetric.MeterProvider
}
func NewProvider(ctx context.Context, o Options) (*Provider, error)
func (p *Provider) Shutdown(ctx context.Context) error
```

**指标命名建议**

* `rpc.server.duration` (Histogram, ms)
* `rpc.server.requests` (Counter; labels: code, service, method)
* `rpc.server.payload_bytes` (UpDownCounter; labels: direction=in|out)

## 3.6 `k8sx`（名称法监听 & 服务发现 & Secret 契约）

```go
package k8sx
type WatchOptions struct {
  Namespace    string
  ResyncPeriod time.Duration
  Logger       log.Logger
}
func WatchConfigMap(ctx context.Context, name string, o WatchOptions, onUpdate func(data map[string]string)) error

type ServiceKind string // "headless" | "clusterip"
func Resolve(ctx context.Context, service string, kind ServiceKind) ([]string /* host:port */, error)
```

> **约定**：库不直接读 Secret 值；由应用通过 env + `secretKeyRef` 注入。

## 3.7 `storex`（可选）

* 连接注册、GORM/TiDB 适配、迁移钩子、`Ping()` 健康；对上游仅暴露最小接口，避免耦合。

---

# 4) 统一配置与端口策略

* 环境变量（建议默认；可由 `configx.EnvSource` 读取）
  `SERVICE_NAME`、`SERVICE_VERSION`、`ENV`
  `HTTP_PORT=:8080`
  `HEALTH_PORT=:8081`、`METRICS_PORT=:9091`
  `OTEL_EXPORTER_OTLP_ENDPOINT`（给 `obsx`）
  `APP_CONFIGMAP_NAME`、`CACHE_CONFIGMAP_NAME`、`ACL_CONFIGMAP_NAME`（多 ConfigMap 名称法监听）
  亦可采用约定：识别所有 `*_CONFIGMAP_NAME` 变量作为动态来源
  `DISCOVERY_TARGET_SERVICE_NAME` / `DISCOVERY_TARGET_SERVICE_KIND=headless|clusterip`

* **默认（Connect-only）**：单端口（h2/h2c）承载 HTTP/Connect/gRPC-Web；健康/指标独立端口。

**配置来源与优先级（建议）**：

- `FileSource`（可选，最低优先级）：本地 JSON/YAML/TOML 文件；适合本地开发或容器镜像内默认。
- `EnvSource`（中）：容器环境变量；通过 K8s `env` 与 `secretKeyRef` 注入敏感值。
- `K8sConfigMapSource`（高）：通过 `APP_CONFIGMAP_NAME` 指定名称；在变更时热更新覆盖。

> 变更生效策略：由业务在 `configx.Manager.OnUpdate` 回调中原子重建依赖（例如更新限流器阈值），避免长尾竞态；对高频抖动键，应设置 `Debounce`。

---

# 5) 最小可运行示例（examples/minimal-connect-service）

**`main.go`**

```go
func main() {
  ctx := context.Background()
  logger := newYourLogger() // 实现 core/log.Logger

  otel, _ := obsx.NewProvider(ctx, obsx.Options{
    ServiceName: os.Getenv("SERVICE_NAME"),
    ServiceVersion: os.Getenv("SERVICE_VERSION"),
    OTLPEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
    EnableRuntimeMetrics: true,
  })
  defer otel.Shutdown(ctx)

  // Connect-only：单端口承载 HTTP/Connect/gRPC-Web（h2/h2c）
  mux := http.NewServeMux()
  ints := connectx.DefaultInterceptors(connectx.Options{
    Logger: logger,
    Otel:   otel,
  })

  // 由 protoc-gen-connect-go 生成
  path, handler := yourapi.NewGreeterServiceHandler(
    NewGreeterImpl(),
    connect.WithInterceptors(ints...),
  )
  connectx.Bind(mux, path, handler)

  _ = runtimex.Run(ctx, nil, runtimex.Options{
    Logger:  logger,
    HTTP:    &runtimex.HTTPOptions{Addr: getenv("HTTP_PORT", ":8080"), H2C: true, Mux: mux},
    Health:  &runtimex.Endpoint{Addr: getenv("HEALTH_PORT", ":8081")},
    Metrics: &runtimex.Endpoint{Addr: getenv("METRICS_PORT", ":9091")},
    ShutdownTimeout: 15 * time.Second,
  })
}
```

**使用 `configx` 进行配置加载与热更新（片段）**

```go
// 继承 BaseConfig 的应用配置：静态基线来自环境变量；动态键（如限流/开关）
// 在启用 K8s 模式时由 ConfigMap 覆盖。
type AppConfig struct {
  configx.BaseConfig
  SlowMillis   int64 `env:"SLOW_REQUEST_MILLIS" default:"1000"`
  RateLimitQPS int   `env:"RATE_LIMIT_QPS" default:"100"`
}

func setupConfig(ctx context.Context, logger log.Logger) (configx.Manager, AppConfig, error) {
  env := configx.NewEnvSource(configx.EnvOptions{Prefix: ""})
  var sources []configx.Source
  sources = append(sources, env)
  if name := os.Getenv("APP_CONFIGMAP_NAME"); name != "" {
    cms := configx.NewK8sConfigMapSource(name, configx.K8sOptions{Namespace: os.Getenv("NAMESPACE")})
    sources = append(sources, cms)
  }
  m, err := configx.NewManager(ctx, configx.Options{Logger: logger, Sources: sources, Debounce: 200 * time.Millisecond})
  if err != nil { return nil, AppConfig{}, err }
  var ac AppConfig
  if err := m.Bind(&ac); err != nil { return nil, AppConfig{}, err }
  // 可选：监听热更新，仅针对动态键重建依赖
  m.OnUpdate(func(snap map[string]string){
    var next AppConfig
    _ = m.Bind(&next)
    // 依据 next 更新限流/白名单等运行时对象（避免变更静态端口/身份等）
  })
  return m, ac, nil
}
```

**最小服务（超简聚合版，推荐）**

```go
// main.go（聚合式拉起，最少样板代码）
func main() {
  ctx := context.Background()
  logger := newYourLogger() // 实现 core/log.Logger

  // 1) 配置：Env 基线 + 可选多 ConfigMap（APP/CACHE/ACL）
  sources, _ := buildSources(ctx, logger)
  m, _ := configx.NewManager(ctx, configx.Options{Logger: logger, Sources: sources, Debounce: time.Duration(configDebounceMs()) * time.Millisecond})
  var cfg AppConfig
  _ = m.Bind(&cfg)

  // 2) 观测
  otel, _ := obsx.NewProvider(ctx, obsx.Options{
    ServiceName: cfg.ServiceName, ServiceVersion: cfg.ServiceVersion,
    OTLPEndpoint: cfg.OTLPEndpoint, EnableRuntimeMetrics: true,
  })
  defer otel.Shutdown(ctx)

  // 3) Connect-only 路由 + 拦截器
  mux := http.NewServeMux()
  ints := connectx.DefaultInterceptors(connectx.Options{Logger: logger, Otel: otel, SlowRequestMillis: cfg.SlowMillis})
  path, handler := yourapi.NewGreeterServiceHandler(NewGreeterImpl(), connect.WithInterceptors(ints...))
  connectx.Bind(mux, path, handler)

  // 4) 运行时（单端口 + 健康/指标独立端口）
  _ = runtimex.Run(ctx, nil, runtimex.Options{
    Logger:  logger,
    HTTP:    &runtimex.HTTPOptions{Addr: cfg.HTTPPort, H2C: true, Mux: mux},
    Health:  &runtimex.Endpoint{Addr: cfg.HealthPort},
    Metrics: &runtimex.Endpoint{Addr: cfg.MetricsPort},
    ShutdownTimeout: 15 * time.Second,
  })
}

// configDebounceMs 从环境变量读取去抖时长，缺省 200ms。
func configDebounceMs() int {
  if v := os.Getenv("CONFIG_DEBOUNCE_MS"); v != "" { i, _ := strconv.Atoi(v); return i }
  return 200
}
```

---

# 6) 版本、发版与 CI/CD（Monorepo 的子目录 Tag）

* **子目录 Tag 规范**

  * `core/v1.0.0`、`runtimex/v1.0.0`、`connectx/v1.2.0`、`obsx/v1.1.0` ……
  * 当主版本 **v2+**：模块路径带 `/v2`（例如 `module github.com/.../connectx/v2`），Tag 用 `connectx/v2.0.0`。
* **业务仓库引用示例**

  ```bash
  go get github.com/eggybyte-technology/egg/core@core/v1.0.0
  go get github.com/eggybyte-technology/egg/runtimex@runtimex/v1.0.0
  go get github.com/eggybyte-technology/egg/connectx@connectx/v1.2.0
  ```
* **CI：两类工作流**

  1. PR 校验：按 `paths` 变化矩阵化地只测改动模块。
  2. Release：**匹配子目录 Tag** 的推送触发，仅构建发布该模块；可附 `gorelease` 做 API 兼容检查、预热 Go Proxy（公有时）。

---

# 7) 质量与治理

* **接口稳定性**：

  * `core`/`runtimex` 严控破坏性变更；导出符号标注 `// Stable` 与 `// Experimental`。
  * 卫星模块（`connectx/k4sx/obsx/storex`）快节奏小版本；破坏性变更走主版本。
* **统一质量门槛**：

  * `go vet`、`golangci-lint`、`go test -race -cover`
  * `gorelease`（对比上一个 tag 的 API 兼容性）
  * `govulncheck`（供应链安全）
* **Observability 基线**：

  * 指标：所有服务统一暴露 `rpc.server.*` 与运行时指标；
  * 日志：统一字段与等级策略；
  * Trace：统一 service 名称、版本与采样策略。
* **文档**：

  * `docs/ARCHITECTURE.md`（分层/依赖/端口/错误/日志/指标与 trace 规范）
  * `docs/RELEASING.md`（子目录 Tag、v2+ 路径规则、撤回/retract）
  * `docs/CONTRIBUTING.md`（代码风格、提交规范、测试要求、变更审阅流程）

---

# 8) 迁移与落地顺序（不含任何异步承诺，仅操作顺序）

1. 创建仓库 **egg** 并按上面结构初始化 `core/runtimex/connectx/obsx/k8sx/storex` 与 `go.work`。
2. 先发布 `core/v1.0.0` 与 `runtimex/v1.0.0`（子目录 Tag）。
3. 在 `connectx/obsx/k8sx` 补齐默认实现与测试后，各自打 `v1.0.0`。
4. 业务服务逐步把原依赖替换为：**按需引入** `core + runtimex + connectx (+ obsx + k8sx + storex)`。
5. 按模块独立发版，最大化减少耦合与回归面。

---

## 小结

* **egg = Monorepo 的多模块通用库族**：

  * `core`（零依赖的基础接口与容器）
  * `runtimex`（与传输无关的运行时内核）
  * `connectx`（Higress 头 → 身份注入 + 统一拦截器 + 错误映射 + 绑定工具）
  * `configx`（统一配置：Env/File + K8s ConfigMap 热更新）
  * `obsx`（Otel/Prom 初始化）
  * `k8sx`（名称法监听/发现/Secret 契约）
  * `storex`（可选的 DB 适配）

* **用什么引什么**：不需要 K8s/DB 的服务，不会被动带入；默认单端口统一、健康/指标独立端口；日志/指标/追踪/错误口径平台一致。