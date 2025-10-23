# EggyByte Go 微服务框架设计指南

## 1) 愿景与基线原则

* **极薄内核 + 可插拔卫星库 + 开箱即用框架**：只引入需要的模块，最小依赖、最快构建；`servicex` 提供一键启动的企业级微服务框架
* **Connect-first**：统一拦截器栈（恢复、日志、追踪、指标、身份注入），0 业务侵入
* **统一端口策略**：默认**单端口**承载 HTTP/Connect/gRPC-Web，**健康/指标独立端口**
* **K8s "名称法"**：ConfigMap 仅注入**名称**，运行时监听并热更新；Secret 用 `secretKeyRef`，服务发现区分 `headless/clusterip`
* **分层认证模型**：Higress 层负责认证与身份注入，微服务层专注权限检查与业务逻辑
* **稳定 API**：`core`、`runtimex`、`servicex` 尽量稳定；其余模块小步快跑

---

## 2) 仓库与模块布局

**仓库**：`github.com/eggybyte-technology/egg`

```
egg/
├─ go.work
├─ README.md
├─ docs/
│  ├─ guide.md                  # 框架设计指南
│  ├─ egg-cli.md                # CLI 工具使用说明
│  └─ CONTRIBUTING.md
├─ core/        # L1：零依赖的接口与通用工具（稳定）
│  ├─ go.mod -> module github.com/eggybyte-technology/egg/core
│  ├─ identity/                 # 身份容器与权限检查
│  ├─ errors/                   # 结构化错误处理
│  ├─ log/                      # 日志接口
│  └─ utils/                     # 通用工具函数
├─ runtimex/    # L2：运行时（生命周期/服务器/健康/指标/基础配置；不含 Connect/K8s）
│  └─ go.mod -> module github.com/eggybyte-technology/egg/runtimex
├─ servicex/    # L2：微服务框架（整合 bootstrap/connectx/configx/obsx；一键启动）
│  └─ go.mod -> module github.com/eggybyte-technology/egg/servicex
├─ connectx/    # L3：Connect 绑定 + 统一拦截器 + 身份注入 + 权限检查
│  ├─ go.mod -> module github.com/eggybyte-technology/egg/connectx
│  └─ internal/                 # 内部拦截器实现
├─ configx/     # L3：统一配置（Env/File + K8s ConfigMap 热更新）
│  └─ go.mod -> module github.com/eggybyte-technology/egg/configx
├─ obsx/        # L3：OpenTelemetry/Prometheus 初始化
│  └─ go.mod -> module github.com/eggybyte-technology/egg/obsx
├─ k8sx/        # L3：ConfigMap 名称法监听、服务发现、Secret 契约
│  ├─ go.mod -> module github.com/eggybyte-technology/egg/k8sx
│  └─ internal/                 # 内部实现
├─ storex/      # L3：TiDB/MySQL/GORM、仓库注册与健康探针（可选）
│  ├─ go.mod -> module github.com/eggybyte-technology/egg/storex
│  └─ internal/                 # 内部实现
├─ cli/         # CLI 工具（独立模块）
│  ├─ go.mod -> module github.com/eggybyte-technology/egg/cli
│  ├─ cmd/egg/                  # CLI 命令实现
│  └─ internal/                 # CLI 内部实现
└─ examples/
   ├─ minimal-connect-service/   # 最小可运行服务示例（独立 go.mod）
   └─ user-service/              # 完整业务服务示例（独立 go.mod）
```

**依赖方向（只许向下）**
`core → {runtimex, servicex} → {connectx, configx, obsx, k8sx, storex}`；各模块间禁止循环依赖。
`servicex` 可依赖所有下层模块以提供整合功能；卫星模块仅依赖必要的上游模块。

**go.work（根）**

```go
go 1.23
use (
  ./core
  ./runtimex
  ./servicex
  ./connectx
  ./configx
  ./obsx
  ./k8sx
  ./storex
  ./cli
  ./examples/minimal-connect-service
  ./examples/user-service
)
```

---

## 3) 模块职责与对外 API（精简而完备）

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
* `identity`（身份容器与权限检查工具）

  ```go
  package identity
  type UserInfo struct {
    UserID   string   // 用户唯一标识
    UserName string   // 用户显示名称
    Roles    []string // 用户角色列表
  }
  type RequestMeta struct {
    RequestID     string // 请求追踪ID
    InternalToken string // 内部服务令牌
    RemoteIP      string // 客户端IP
    UserAgent     string // 客户端用户代理
  }
  
  // 身份注入与获取
  func WithUser(ctx context.Context, u *UserInfo) context.Context
  func UserFrom(ctx context.Context) (*UserInfo, bool)
  func WithMeta(ctx context.Context, m *RequestMeta) context.Context
  func MetaFrom(ctx context.Context) (*RequestMeta, bool)
  
  // 权限检查便捷方法
  func HasRole(ctx context.Context, role string) bool
  func HasAnyRole(ctx context.Context, roles ...string) bool
  func IsInternalService(ctx context.Context, serviceName string) bool
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

## 3.2.1 `servicex`（微服务框架：开箱即用的一键启动）

`servicex` 是更高层的整合框架，建立在 `runtimex`、`connectx`、`configx`、`obsx` 等基础模块之上，提供**一键启动**的微服务解决方案。

```go
package servicex

// App provides access to service components during registration.
// This is the only interface exposed to service registration functions.
type App struct {
	mux          *http.ServeMux
	logger       log.Logger
	interceptors []connect.Interceptor
	db           *gorm.DB
	otel         *obsx.Provider
}

// Options holds configuration for service initialization.
type Options struct {
	// Service identification
	ServiceName    string `env:"SERVICE_NAME" default:"app"`
	ServiceVersion string `env:"SERVICE_VERSION" default:"0.0.0"`

	// Configuration
	Config any // Configuration struct that embeds configx.BaseConfig

	// Database (optional)
	Database *DatabaseConfig  // Database configuration
	Migrate  DatabaseMigrator // Database migration function

	// Service registration
	Register ServiceRegistrar // Service registration function

	// Observability
	EnableTracing bool `env:"ENABLE_TRACING" default:"true"`

	// Feature flags
	EnableHealthCheck bool `env:"ENABLE_HEALTH_CHECK" default:"true"`
	EnableMetrics     bool `env:"ENABLE_METRICS" default:"true"`
	EnableDebugLogs   bool `env:"ENABLE_DEBUG_LOGS" default:"false"`

	// Connect interceptor options
	SlowRequestMillis int64 `env:"SLOW_REQUEST_MILLIS" default:"1000"`
	PayloadAccounting bool  `env:"PAYLOAD_ACCOUNTING" default:"true"`

	// Shutdown timeout
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" default:"15s"`

	// Logger (optional, will create default if nil)
	Logger log.Logger
}

// Run starts a microservice with the given options.
// This is the main entry point for servicex - it handles all initialization
// and provides a single function call to start a complete microservice.
func Run(ctx context.Context, opts Options) error
```

**核心特性**：

* **一键启动**：`servicex.Run()` 处理所有初始化，提供单函数调用启动完整微服务
* **自动配置管理**：整合 `configx`，支持环境变量、文件、K8s ConfigMap 多来源配置
* **内置观测性**：自动集成 `obsx`，提供追踪、指标收集和结构化日志
* **智能拦截器栈**：使用 `connectx` 提供恢复、日志、身份注入等统一拦截器
* **数据库集成**（可选）：支持 GORM 连接、连接池配置和自动迁移
* **健康检查**：内置健康和指标端点，支持数据库状态检查
* **优雅关闭**：自动处理信号捕获和资源清理
* **默认 Logger**：提供结构化日志实现，无需手动配置

**典型使用方式**：

```go
func main() {
	ctx := context.Background()

	// servicex.Run() 处理一切：配置、观测、数据库、拦截器、服务器启动
	err := servicex.Run(ctx, servicex.Options{
		ServiceName: "my-service",
		ServiceVersion: "1.0.0",
		Config: &AppConfig{}, // 继承 configx.BaseConfig
		Register: func(app *servicex.App) error {
			// 注册 Connect handlers - servicex 已配置好拦截器栈
			path, handler := myv1connect.NewMyServiceHandler(
				myHandler,
				connect.WithInterceptors(app.Interceptors()...),
			)
			app.Mux().Handle(path, handler)
			return nil
		},
		EnableTracing:     true,
		EnableHealthCheck: true,
		EnableMetrics:     true,
		// Database: &servicex.DatabaseConfig{...}, // 可选
		// Logger: customLogger, // 可选，默认会自动创建
	})
	if err != nil {
		panic(err)
	}
}
```

**适用场景**：

* **快速原型**：几分钟内启动带完整特性的微服务
* **标准化部署**：团队所有服务使用统一启动框架
* **生产环境**：内置最佳实践，减少配置错误
* **渐进式迁移**：可选择性启用高级特性

**与底层模块关系**：

`servicex` 依赖并整合多个底层模块，但**不取代**它们：
- `runtimex`：提供底层 HTTP 服务器和生命周期管理
- `connectx`：提供拦截器栈和 Connect 绑定
- `configx`：提供配置管理和热更新
- `obsx`：提供观测性和指标收集
- 业务代码仍可直接使用这些底层模块的细粒度控制

## 3.3 `connectx`（Connect 绑定 + 统一拦截器 + 身份注入）

* **分层认证模型**：Higress 层通过 ext-auth 插件完成认证，将用户身份注入请求头；微服务层仅需提取身份信息并进行权限检查
* **零业务侵入**：统一拦截器栈自动处理身份注入、错误映射、日志记录等横切关注点

* 统一拦截器（恢复、日志、追踪、指标、错误映射、身份注入）

  ```go
  package connectx

  type HeaderMapping struct {
    RequestID     string // "X-Request-Id"
    InternalToken string // "X-Internal-Token"
    UserID        string // "X-User-Id"
    UserName      string // "X-User-Name"
    Roles         string // "X-User-Roles"
    RealIP        string // "X-Real-IP"
    ForwardedFor  string // "X-Forwarded-For"
    UserAgent     string // "User-Agent"
  }

  type Options struct {
    Logger            log.Logger
    Otel              *obsx.Provider // nil 时禁用 trace/metrics
    Headers           HeaderMapping  // 可覆盖默认映射
    WithRequestBody   bool           // 生产默认 false
    WithResponseBody  bool
    SlowRequestMillis int64          // 慢请求阈值
    PayloadAccounting bool           // 记录入出站字节
  }

  func DefaultInterceptors(o Options) []connect.Interceptor

  // 绑定工具：把 protoc-gen-connect-go 生成的 handler 路由到 mux
  func Bind(mux *http.ServeMux, path string, h http.Handler)
  ```

* **错误映射**
  `core/errors` → Connect `Code` → HTTP：
  * `INVALID_ARGUMENT` → `CodeInvalidArgument` → 400
  * `NOT_FOUND` → `CodeNotFound` → 404
  * `ALREADY_EXISTS` → `CodeAlreadyExists` → 409
  * `PERMISSION_DENIED` → `CodePermissionDenied` → 403
  * `UNAUTHENTICATED` → `CodeUnauthenticated` → 401
  * `INTERNAL`/默认 → `CodeInternal` → 500

* **权限检查便捷方法**
  ```go
  // 检查用户是否具有指定角色
  if identity.HasRole(ctx, "admin") {
    // 管理员操作
  }
  
  // 检查用户是否具有任一角色
  if identity.HasAnyRole(ctx, "admin", "editor") {
    // 管理员或编辑者操作
  }
  
  // 检查是否为内部服务调用
  if identity.IsInternalService(ctx, "user-service") {
    // 内部服务调用，跳过权限检查
  }
  ```

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

## 4) 统一配置与端口策略

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

## 5) 实际项目示例

### 5.1 最小服务示例（examples/minimal-connect-service）

**特点**：演示框架基础功能，包含配置管理、拦截器、健康检查等核心特性。

**`main.go`**

```go
func main() {
  ctx := context.Background()

  // servicex.Run() 一键启动：自动处理配置、观测、拦截器、服务器等
  err := servicex.Run(ctx, servicex.Options{
    ServiceName: "minimal-connect-service",
    ServiceVersion: "1.0.0",
    Config: &AppConfig{}, // 继承 configx.BaseConfig
    Register: func(app *servicex.App) error {
      // 注册 Connect handlers - servicex 已配置好拦截器栈
      greeterService := &GreeterService{}
      path, handler := greetv1connect.NewGreeterServiceHandler(
        greeterService,
        connect.WithInterceptors(app.Interceptors()...),
      )
      app.Mux().Handle(path, handler)

      // 使用 servicex 提供的 logger（自动配置）
      app.Logger().Info("Service handlers registered successfully")
      return nil
    },
    EnableTracing:     true,
    EnableHealthCheck: true,
    EnableMetrics:     true,
    EnableDebugLogs:   false,
    SlowRequestMillis: 1000,
    PayloadAccounting: true,
    ShutdownTimeout:   15 * time.Second,
    // Logger: customLogger, // 可选，默认会自动创建结构化 logger
  })

  if err != nil {
    panic(err) // servicex 已处理内部日志记录
  }
}
```

**核心优势对比**：

| 方面 | 传统方式 | servicex 方式 |
|------|---------|-------------|
| **代码行数** | ~60 行 | ~25 行 |
| **配置管理** | 手动初始化 | 自动处理 |
| **拦截器栈** | 手动配置 | 内置最佳实践 |
| **观测性** | 手动集成各组件 | 开箱即用 |
| **健康检查** | 手动添加 | 内置支持 |
| **错误处理** | 分散处理 | 统一处理 |

**渐进式增强**：servicex 支持选择性启用高级特性，如数据库集成、自定义迁移等。

### 5.2 完整业务服务示例（examples/user-service）

**特点**：演示完整的分层架构，包含数据库集成、业务逻辑、错误处理等企业级特性。

**项目结构**
```
user-service/
├── cmd/server/main.go          # 服务入口
├── internal/
│   ├── config/                 # 配置管理
│   │   ├── app_config.go       # 应用配置
│   │   └── k8s_watcher.go      # K8s 配置监听
│   ├── handler/                # Connect 协议处理
│   │   └── user_handler.go     # 用户服务处理器
│   ├── service/                # 业务逻辑层
│   │   └── user_service.go     # 用户业务服务
│   ├── repository/             # 数据访问层
│   │   └── user_repository.go  # 用户数据仓库
│   └── model/                  # 数据模型
│       ├── user.go             # 用户模型
│       └── errors.go           # 领域错误
├── api/                        # Protobuf 定义
└── gen/                        # 生成的代码
```

**分层架构实现**

```go
// 1. 配置层 - 继承 BaseConfig 并扩展业务配置
type AppConfig struct {
  configx.BaseConfig
  
  // 数据库配置
  Database DatabaseConfig `env:"DATABASE" yaml:"database"`
  
  // 业务配置
  Business BusinessConfig `env:"BUSINESS" yaml:"business"`
  
  // 功能开关
  Features FeatureConfig `env:"FEATURES" yaml:"features"`
}

// 2. 数据模型层 - 定义领域实体
type User struct {
  ID        string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
  Email     string    `gorm:"uniqueIndex;not null;size:255" json:"email"`
  Name      string    `gorm:"not null;size:255" json:"name"`
  CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
  UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// 3. 仓储层 - 数据访问接口
type UserRepository interface {
  Create(ctx context.Context, user *model.User) (*model.User, error)
  GetByID(ctx context.Context, id string) (*model.User, error)
  Update(ctx context.Context, user *model.User) (*model.User, error)
  Delete(ctx context.Context, id string) error
  List(ctx context.Context, page, pageSize int) ([]*model.User, int64, error)
}

// 4. 业务服务层 - 核心业务逻辑
type UserService interface {
  CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error)
  GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error)
  UpdateUser(ctx context.Context, req *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error)
  DeleteUser(ctx context.Context, req *userv1.DeleteUserRequest) (*userv1.DeleteUserResponse, error)
  ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error)
}

// 5. 处理器层 - Connect 协议适配
type UserHandler struct {
  userv1connect.UnimplementedUserServiceHandler
  service service.UserService
  logger  log.Logger
}
```

**服务启动流程（使用 servicex）**

```go
func main() {
  ctx := context.Background()

  // servicex.Run() 一键启动完整微服务
  err := servicex.Run(ctx, servicex.Options{
    ServiceName: "user-service",
    ServiceVersion: "1.0.0",
    Config: &config.AppConfig{}, // 继承 configx.BaseConfig
    // Database: &servicex.DatabaseConfig{...}, // 可选：启用数据库
    // Migrate: func(db *gorm.DB) error {        // 可选：数据库迁移
    //   return db.AutoMigrate(&model.User{})
    // },
    Register: func(app *servicex.App) error {
      // 依赖注入：servicex 已处理配置、观测、数据库等
      var userRepo repository.UserRepository
      if db := app.DB(); db != nil {
        userRepo = repository.NewUserRepository(db)
        app.Logger().Info("Database repository initialized")
      } else {
        // 无数据库时使用内存仓储
        userRepo = &mockUserRepository{}
        app.Logger().Info("Using in-memory repository")
      }

      // 业务服务和处理器
      userService := service.NewUserService(userRepo, app.Logger())
      userHandler := handler.NewUserHandler(userService, app.Logger())

      // 注册 Connect handlers
      path, connectHandler := userv1connect.NewUserServiceHandler(
        userHandler,
        connect.WithInterceptors(app.Interceptors()...),
      )
      app.Mux().Handle(path, connectHandler)

      app.Logger().Info("User service registered successfully", log.Str("path", path))
      return nil
    },
    EnableTracing:     true,
    EnableHealthCheck: true, // 自动包含数据库健康检查
    EnableMetrics:     true,
    EnableDebugLogs:   false,
    SlowRequestMillis: 1000,
    PayloadAccounting: true,
    ShutdownTimeout:   15 * time.Second,
  })

  if err != nil {
    panic(err) // servicex 已处理内部日志记录
  }
}
```

**核心优势对比**：

| 方面 | 传统方式 | servicex 方式 |
|------|---------|-------------|
| **代码行数** | ~100 行 | ~35 行 |
| **组件初始化** | 手动逐个初始化 | servicex 统一处理 |
| **依赖管理** | 手动依赖注入 | servicex 自动注入 |
| **错误处理** | 分散错误处理 | 统一错误处理 |
| **资源清理** | 手动清理各项资源 | servicex 优雅关闭 |
| **可测试性** | 较难测试启动流程 | 高度可测试 |

**企业级特性支持**：servicex 内置支持数据库连接池、健康检查、指标收集、分布式追踪等生产环境必需特性。

### 5.3 架构模式与最佳实践

**分层架构原则**
- **Handler 层**：负责 Connect 协议适配，处理请求/响应转换
- **Service 层**：包含核心业务逻辑，不依赖传输协议
- **Repository 层**：数据访问抽象，支持多种存储后端
- **Model 层**：领域实体和错误定义

**错误处理策略**
```go
// 领域错误定义
var (
  ErrUserNotFound = errors.New("USER_NOT_FOUND", "user not found")
  ErrInvalidEmail = errors.New("INVALID_EMAIL", "invalid email address")
  ErrEmailExists  = errors.New("EMAIL_EXISTS", "email already exists")
)

// 错误包装和传播
func (s *userService) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
  if s.repo == nil {
    return nil, errors.New("SERVICE_UNAVAILABLE", "database repository not available")
  }
  
  user, err := s.repo.Create(ctx, user)
  if err != nil {
    return nil, err // 错误已在 repository 层包装
  }
  
  return response, nil
}
```

**配置管理最佳实践**
- 继承 `configx.BaseConfig` 获取基础配置
- 使用环境变量提供默认值
- 支持 K8s ConfigMap 热更新
- 配置验证确保数据完整性

**数据库集成模式**
- 使用 GORM 进行 ORM 映射
- 支持自动迁移和连接池配置
- 优雅处理数据库不可用的情况
- 健康检查包含数据库状态

**观测性集成**
- OpenTelemetry 自动追踪
- 结构化日志记录
- 性能指标收集
- 慢请求检测

---

## 6) 版本、发版与 CI/CD（Monorepo 的子目录 Tag）

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

## 7) 质量与治理

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

## 8) 迁移与落地顺序（不含任何异步承诺，仅操作顺序）

1. 创建仓库 **egg** 并按上面结构初始化 `core/runtimex/servicex/connectx/configx/obsx/k8sx/storex` 与 `go.work`。
2. 先发布 `core/v1.0.0`、`runtimex/v1.0.0` 与 `servicex/v1.0.0`（子目录 Tag）。
3. 在 `connectx/configx/obsx/k8sx` 补齐默认实现与测试后，各自打 `v1.0.0`。
4. **两条路径供选择**：
   * **快速路径**：直接使用 `servicex` 获得开箱即用的完整微服务框架
   * **精细控制路径**：按需引入 `core + runtimex + connectx (+ configx + obsx + k8sx + storex)`
5. 按模块独立发版，最大化减少耦合与回归面。

---

## 9) 小结

* **egg = Monorepo 的多模块通用库族 + 开箱即用框架**：
  * `core`（零依赖的基础接口与身份容器）
  * `runtimex`（与传输无关的运行时内核）
  * `servicex`（一键启动的企业级微服务框架，建立在各模块之上）
  * `connectx`（Higress 身份注入 + 统一拦截器 + 错误映射 + 权限检查工具）
  * `configx`（统一配置：Env/File + K8s ConfigMap 热更新）
  * `obsx`（OpenTelemetry/Prometheus 初始化）
  * `k8sx`（名称法监听/服务发现/Secret 契约）
  * `storex`（可选的数据库适配）

* **分层架构**：
  * **极薄内核**：`core` 提供零依赖的接口和通用工具
  * **运行时层**：`runtimex` 和 `servicex` 提供不同粒度的启动抽象
  * **功能层**：各卫星库提供特定领域功能，按需引入
* **分层认证模型**：Higress 层负责认证与身份注入，微服务层专注权限检查与业务逻辑
* **按需引入**：不需要 K8s/DB 的服务不会被动带入依赖
* **统一端口策略**：默认单端口承载 HTTP/Connect/gRPC-Web，健康/指标独立端口
* **统一观测口径**：日志/指标/追踪/错误处理平台一致
* **开发者体验优先**：`servicex` 提供开箱即用的企业级特性，极大降低微服务开发门槛