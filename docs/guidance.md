# 🧱 egg v0.1.0 模块级统一实现规范

## 1️⃣ 总体结构

```
github.com/eggybyte-technology/egg/
├── go.work
├── Makefile
├── docs/
│   ├── ARCHITECTURE.md
│   ├── LOGGING.md
│   └── MODULE_GUIDE.md
├── internal/
│   ├── testutil/        # 跨模块测试工具（非导出）
│   └── version/         # 自动注入版本信息
├── core/                # L0：零依赖核心
├── logx/                # L1：日志
├── configx/             # L2：配置加载与热更
├── obsx/                # L2：观测
├── httpx/               # L2：HTTP 辅助
├── runtimex/            # L3：运行时
├── connectx/            # L3：Connect 拦截器栈
├── clientx/             # L3：Connect 客户端
├── servicex/            # L4：集成器
├── storex/              # 可选：存储与事务
├── testingx/            # 测试辅助模块
└── examples/
    ├── minimal-connect-service/
    └── user-service/
```

---

## 2️⃣ 模块间依赖约束

| 模块         | 向下依赖                       | 说明                             |
| ---------- | -------------------------- | ------------------------------ |
| `core`     | 无                          | 基础接口、错误、上下文、身份                 |
| `logx`     | core                       | 标准日志实现                         |
| `configx`  | core, logx                 | 多源配置、校验、热更                     |
| `obsx`     | core, logx                 | OTel/Prom 初始化                  |
| `httpx`    | core, logx                 | Bind & Validate、Secure Headers |
| `runtimex` | core, logx, obsx, httpx    | HTTP Server 生命周期               |
| `connectx` | core, logx, obsx, configx  | 拦截器栈与 RPC 治理                   |
| `clientx`  | core, logx, obsx, connectx | 客户端治理层                         |
| `servicex` | 所有上层 L3                    | 单函数集成启动                        |
| `storex`   | core, logx, configx        | DB 事务与健康检查                     |
| `testingx` | 所有 L0–L3                   | 测试工具，不参与运行时依赖                  |

👉 **规则**：只能依赖同层或更低层；严禁跨层、循环依赖。

---

## 3️⃣ 每个模块的标准目录与文件模板

以 `connectx` 为例（其余模块遵循相同规范）：

```
connectx/
├── go.mod
├── README.md
├── doc.go                    # 模块文档（Godoc）
├── options.go                # Option 结构体与默认值
├── interceptor_timeout.go    # 超时拦截器
├── interceptor_logging.go    # 日志拦截器
├── interceptor_metrics.go    # 指标拦截器
├── interceptor_errors.go     # 错误映射拦截器
├── interceptor_auth.go       # 身份注入拦截器
├── chain.go                  # 拦截器链工具
├── registry.go               # 方法信息注册与动态配置
├── types.go                  # 公共类型定义
├── internal/
│   └── reflectutil.go        # 私有工具，不导出
└── test/
    ├── timeout_test.go
    ├── logging_test.go
    └── chain_test.go
```

**所有模块应包含以下五类文件：**

1. `doc.go` → 模块描述、使用方式、示例。
2. `options.go` → Option 定义、默认值与构造函数。
3. 核心实现 + 内部工具（必要时放 `internal/`）。
4. `test/` → 单元测试 + Mock。
5. `README.md` → 简明使用说明（用于 GitHub 可读）。

---

## 4️⃣ 模块通用开发规范

### 🔸 命名规范

* 文件名：功能+后缀，如 `timeout_interceptor.go`。
* 变量：首字母缩写使用小写（如 `cfg`, `ctx`, `log`）。
* 公开函数需完整注释，以 Godoc 格式说明用途。

### 🔸 函数式选项统一风格

```go
type Option func(*Options)

func WithDefaultTimeout(ms int64) Option {
    return func(o *Options) { o.DefaultTimeoutMs = ms }
}
```

### 🔸 错误与日志

* 错误统一使用 `core/errors` 构建，配合 `WithOp` 链式包装；
* 日志统一使用 `logx`，从 `context` 提取 `logx.FromContext(ctx)`；
* 打印时 **单行 logfmt + 排序键**。

### 🔸 配置加载

所有模块支持：

```go
cfg := configx.Bind(&MyConfig{})
```

并通过 `validate` 标签进行结构体字段校验。

### 🔸 观测埋点

使用 `obsx` 的 `MeterProvider` 统一命名规则：

```
rpc.server.requests_total
rpc.server.duration_ms_bucket
rpc.client.retries_total
```

### 🔸 健康检查接口

统一定义：

```go
type HealthChecker interface {
    Name() string
    Check(ctx context.Context) error
}
```

### 🔸 测试要求

* 每个模块至少 80% 覆盖率；
* 使用 `testingx.NewMockLogger()` 与 `testingx.NewContextWithIdentity()`；
* 集成测试放在 `examples/`。

---

## 5️⃣ go.work 配置

```text
go 1.23

use (
    ./core
    ./logx
    ./configx
    ./obsx
    ./httpx
    ./runtimex
    ./connectx
    ./clientx
    ./servicex
    ./storex
    ./testingx
    ./examples
)
```

> 每个子模块独立 `go.mod` ，版本号独立打 tag。

---

## 6️⃣ 发布与版本策略

* 每个模块单独 semver 发版：`core/v0.1.0`、`servicex/v0.1.0` 等；
* CI 按 `paths` 差异触发对应模块构建；
* 在 docs/ARCHITECTURE.md 中维护模块依赖图。

---

## 7️⃣ 团队开发约定

1. **分层约束守卫**：新增模块需注明所在层级及依赖方向。
2. **公共接口注释完整**：导出符号必须有 Godoc 注释。
3. **内部封装**：非导出代码放在 `internal/`，不得被外部模块导入。
4. **包命名一致性**：全小写，不含下划线。
5. **错误码集中管理**：新增 Code 需在 `core/errors/codes.go` 登记。
6. **日志字段标准化**：统一字段名 `level,msg,service,method,latency_ms,...`。
7. **配置文件 / ConfigMap 路径统一**：`data/rpc.yaml` 、`data/app.yaml` 。
8. **Pull Request 检查**：lint + unit test + gorelease 必须通过。

---

## 8️⃣ 示例：`servicex` 模块结构

```
servicex/
├── go.mod
├── README.md
├── doc.go
├── options.go              # WithService, WithConfig 等选项
├── app.go                  # App 结构体，Mux、Logger、Config、Interceptors
├── run.go                  # Run(ctx, options...) 主入口
├── di.go                   # Provide/Resolve 轻量 DI 实现
├── shutdown.go             # 关闭栈实现
├── interceptors.go         # 内置拦截器聚合
└── test/
    └── run_test.go
```

`servicex.Run` 组合调用 `configx`, `obsx`, `connectx`, `runtimex` 等，形成“一行起服”。

---

## ✅ 总结

| 层级     | 模块                            | 职责简述          |
| ------ | ----------------------------- | ------------- |
| **L0** | core                          | 零依赖基础设施       |
| **L1** | logx                          | 结构化日志         |
| **L2** | configx / obsx / httpx        | 配置、观测、HTTP 辅助 |
| **L3** | runtimex / connectx / clientx | 运行时与 RPC 通信栈  |
| **L4** | servicex                      | 框架集成与一键启动     |
| **附属** | storex / testingx             | 持久层与测试支持      |

> 这套结构使 egg 成为一个**分层清晰、规范统一、可持续演进**的微服务通用库。
> 任何新增功能模块都必须沿用此目录与接口模式，确保一致性与可维护性。
