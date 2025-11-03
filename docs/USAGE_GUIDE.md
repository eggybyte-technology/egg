# Egg 框架通用库使用指南

本文档提供了 Egg 框架所有通用库的完整使用指南，帮助开发者快速上手和深入使用各个模块。

## 目录

- [快速开始](#快速开始)
- [L0: 核心层](#l0-核心层)
- [L1: 基础层](#l1-基础层)
- [L2: 能力层](#l2-能力层)
- [L3: 运行时通信层](#l3-运行时通信层)
- [L4: 集成层](#l4-集成层)
- [辅助模块](#辅助模块)
- [最佳实践](#最佳实践)
- [常见场景](#常见场景)

---

## 快速开始

### 最简单的服务启动

```go
package main

import (
    "context"
    "go.eggybyte.com/egg/configx"
    "go.eggybyte.com/egg/servicex"
)

type AppConfig struct {
    configx.BaseConfig  // 包含数据库、HTTP端口等配置
}

func register(app *servicex.App) error {
    // 注册你的服务处理器
    logger := app.Logger()
    logger.Info("service registered")
    return nil
}

func main() {
    ctx := context.Background()
    cfg := &AppConfig{}
    
    servicex.Run(ctx,
        servicex.WithService("my-service", "1.0.0"),
        servicex.WithAppConfig(cfg),
        servicex.WithRegister(register),
    )
}
```

### 环境变量配置

```bash
# 日志级别
LOG_LEVEL=info  # debug, info, warn, error

# 服务配置
SERVICE_NAME=my-service
SERVICE_VERSION=1.0.0
ENV=production

# 端口配置
HTTP_PORT=8080
HEALTH_PORT=8081
METRICS_PORT=9091

# 数据库配置（可选）
DB_DRIVER=mysql
DB_DSN=user:password@tcp(localhost:3306)/mydb?charset=utf8mb4&parseTime=True
DB_MAX_IDLE=10
DB_MAX_OPEN=100
DB_MAX_LIFETIME=1h

# 观测性配置（可选）
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
```

---

## L0: 核心层

核心层提供零依赖的基础接口和类型，是框架的基础。

### core/errors - 结构化错误处理

#### 快速开始

```go
import "go.eggybyte.com/egg/core/errors"

// 创建错误
err := errors.New("NOT_FOUND", "user not found")

// 检查错误类型
if errors.Is(err, "NOT_FOUND") {
    // 处理未找到错误
}

// 包装错误
wrappedErr := errors.Wrap(err, "DATABASE_ERROR", "failed to query user")
```

#### 常用错误码

```go
// 系统错误
errors.New("INTERNAL_ERROR", "internal server error")
errors.New("SERVICE_UNAVAILABLE", "service temporarily unavailable")
errors.New("TIMEOUT", "operation timeout")

// 验证错误
errors.New("VALIDATION_ERROR", "input validation failed")
errors.New("INVALID_FORMAT", "invalid data format")
errors.New("MISSING_REQUIRED", "required field missing")

// 认证授权
errors.New("UNAUTHENTICATED", "user not authenticated")
errors.New("PERMISSION_DENIED", "insufficient permissions")
errors.New("TOKEN_EXPIRED", "authentication token expired")

// 业务逻辑
errors.New("NOT_FOUND", "resource not found")
errors.New("ALREADY_EXISTS", "resource already exists")
errors.New("CONFLICT", "business rule conflict")
```

#### 典型用法

```go
// 在服务层
func (s *UserService) GetUser(ctx context.Context, userID string) (*User, error) {
    if userID == "" {
        return nil, errors.New("MISSING_REQUIRED", "user ID is required")
    }
    
    user, err := s.repo.GetUser(ctx, userID)
    if err != nil {
        if errors.Is(err, "NOT_FOUND") {
            return nil, errors.New("NOT_FOUND", "user not found")
        }
        return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to get user")
    }
    
    return user, nil
}

// 在 Connect 处理器中
func (h *Handler) GetUser(ctx context.Context, req *connect.Request[GetUserRequest]) (*connect.Response[GetUserResponse], error) {
    user, err := h.service.GetUser(ctx, req.Msg.UserId)
    if err != nil {
        code := errors.Code(err)
        switch code {
        case "NOT_FOUND":
            return nil, connect.NewError(connect.CodeNotFound, err)
        case "PERMISSION_DENIED":
            return nil, connect.NewError(connect.CodePermissionDenied, err)
        case "VALIDATION_ERROR":
            return nil, connect.NewError(connect.CodeInvalidArgument, err)
        default:
            return nil, connect.NewError(connect.CodeInternal, err)
        }
    }
    return connect.NewResponse(&GetUserResponse{User: user}), nil
}
```

### core/identity - 身份和请求元数据

#### 快速开始

```go
import "go.eggybyte.com/egg/core/identity"

// 存储用户信息到上下文
ctx := identity.WithUser(ctx, &identity.UserInfo{
    UserID:   "user-123",
    UserName: "john.doe",
    Roles:    []string{"admin", "user"},
})

// 从上下文获取用户信息
if user, ok := identity.UserFrom(ctx); ok {
    fmt.Printf("User: %s (%s)\n", user.UserName, user.UserID)
}

// 检查权限
if identity.HasRole(ctx, "admin") {
    // 用户有管理员权限
}
```

#### 典型用法

```go
// 在 HTTP 中间件中提取身份
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        
        // 从 Header 提取用户信息
        userID := r.Header.Get("X-User-Id")
        userName := r.Header.Get("X-User-Name")
        roles := strings.Split(r.Header.Get("X-User-Roles"), ",")
        
        // 存储到上下文
        ctx = identity.WithUser(ctx, &identity.UserInfo{
            UserID:   userID,
            UserName: userName,
            Roles:    roles,
        })
        
        // 存储请求元数据
        ctx = identity.WithMeta(ctx, &identity.RequestMeta{
            RequestID: r.Header.Get("X-Request-Id"),
            RemoteIP:  r.RemoteAddr,
            UserAgent: r.UserAgent(),
        })
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// 在业务逻辑中使用
func processOrder(ctx context.Context, order *Order) error {
    // 检查权限
    if !identity.HasRole(ctx, "user") {
        return errors.New("PERMISSION_DENIED", "user role required")
    }
    
    // 获取用户信息
    user, ok := identity.UserFrom(ctx)
    if !ok {
        return errors.New("UNAUTHENTICATED", "user not authenticated")
    }
    
    // 使用用户信息
    order.UserID = user.UserID
    return nil
}
```

### core/log - 日志接口

#### 快速开始

```go
import "go.eggybyte.com/egg/core/log"

// 使用 logx 实现（推荐）
logger := logx.New(logx.WithFormat(logx.FormatLogfmt))

// 结构化日志
logger.Info("user created",
    log.Str("user_id", user.ID),
    log.Str("email", user.Email),
    log.Int64("timestamp", time.Now().Unix()),
)

// 错误日志
logger.Error(err, "failed to process request",
    log.Str("request_id", reqID),
)
```

#### 典型用法

```go
// 在服务中使用
type UserService struct {
    logger log.Logger
    repo   UserRepository
}

func (s *UserService) CreateUser(ctx context.Context, user *User) error {
    s.logger.Info("creating user", log.Str("email", user.Email))
    
    if err := s.repo.CreateUser(ctx, user); err != nil {
        s.logger.Error(err, "failed to create user",
            log.Str("email", user.Email),
        )
        return err
    }
    
    s.logger.Info("user created successfully",
        log.Str("user_id", user.ID),
        log.Str("email", user.Email),
    )
    return nil
}
```

### core/utils - 工具函数

#### 快速开始

```go
import "go.eggybyte.com/egg/core/utils"

// 重试逻辑
err := utils.Retry(ctx, 3, time.Second, func() error {
    return someOperation()
})

// 切片操作
filtered := utils.Filter(slice, func(item string) bool {
    return len(item) > 0
})

mapped := utils.Map(slice, func(item string) string {
    return strings.ToUpper(item)
})
```

---

## L1: 基础层

### logx - 结构化日志实现

#### 快速开始

```go
import (
    "log/slog"
    "go.eggybyte.com/egg/logx"
)

// 创建日志器
logger := logx.New(
    logx.WithFormat(logx.FormatLogfmt),  // 或 FormatJSON, FormatConsole
    logx.WithLevel(slog.LevelInfo),
    logx.WithColor(true),  // 开发环境启用颜色
)

// 日志输出
logger.Info("user created", "user_id", "u-123", "email", "user@example.com")
logger.Warn("slow request", "duration_ms", 1500, "path", "/api/users")
logger.Error(err, "database connection failed", "retry_count", 3)
```

#### 日志格式

**Logfmt 格式**（推荐用于生产环境）：
```
level=INFO msg="user created" email=user@example.com user_id=u-123
level=WARN msg="slow request" duration_ms=1500 path=/api/users
```

**JSON 格式**（用于日志聚合系统）：
```json
{"level":"INFO","msg":"user created","email":"user@example.com","user_id":"u-123"}
```

**Console 格式**（用于开发环境）：
```
INFO    2024-01-15 10:30:00  user created
        email: user@example.com
        user_id: u-123
```

#### 上下文感知日志

```go
import (
    "go.eggybyte.com/egg/core/identity"
    "go.eggybyte.com/egg/logx"
)

func handleRequest(ctx context.Context, baseLogger log.Logger) {
    // 从上下文创建日志器（自动包含 request_id, user_id）
    logger := logx.FromContext(ctx, baseLogger)
    
    // 日志自动包含上下文信息
    logger.Info("processing request")
    // 输出: level=INFO msg="processing request" request_id=req-123 user_id=u-456
}

// 在 HTTP 处理器中
func MyHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // 注入请求元数据
    ctx = identity.WithMeta(ctx, identity.Meta{
        RequestID: "req-123",
    })
    
    // 注入用户信息
    ctx = identity.WithUser(ctx, identity.User{
        UserID: "u-456",
    })
    
    handleRequest(ctx, logger)
}
```

#### 敏感字段屏蔽

```go
logger := logx.New(
    logx.WithFormat(logx.FormatLogfmt),
    logx.WithSensitiveFields("password", "token", "secret", "api_key"),
)

// 这些字段会被屏蔽
logger.Info("user auth",
    "username", "john",
    "password", "secret123",        // 输出: password=***
    "api_key", "sk_live_123456",   // 输出: api_key=***
)
```

#### 日志级别控制

```bash
# 通过环境变量控制
LOG_LEVEL=debug go run main.go    # 显示所有日志
LOG_LEVEL=info go run main.go     # 只显示 info 及以上
LOG_LEVEL=warn go run main.go     # 只显示 warn 和 error
LOG_LEVEL=error go run main.go    # 只显示 error
```

---

## L2: 能力层

### configx - 配置管理

#### 快速开始

```go
import "go.eggybyte.com/egg/configx"

type AppConfig struct {
    configx.BaseConfig  // 包含服务名、版本、端口、数据库等
    
    // 自定义配置
    DatabaseURL string `env:"DATABASE_URL" default:"postgres://localhost/mydb"`
    MaxConns    int    `env:"MAX_CONNS" default:"10"`
    Debug       bool   `env:"DEBUG" default:"false"`
}

func main() {
    ctx := context.Background()
    
    // 创建配置管理器（自动使用环境变量）
    manager, err := configx.DefaultManager(ctx, logger)
    if err != nil {
        log.Fatal(err)
    }
    
    // 绑定配置
    var cfg AppConfig
    err = manager.Bind(&cfg)
    if err != nil {
        log.Fatal(err)
    }
    
    // 使用配置
    fmt.Printf("Database URL: %s\n", cfg.DatabaseURL)
}
```

#### BaseConfig 结构

```go
type BaseConfig struct {
    // 服务标识
    ServiceName    string `env:"SERVICE_NAME" default:"app"`
    ServiceVersion string `env:"SERVICE_VERSION" default:"0.0.0"`
    Env            string `env:"ENV" default:"dev"`
    
    // 端口配置
    HTTPPort    string `env:"HTTP_PORT" default:":8080"`
    HealthPort  string `env:"HEALTH_PORT" default:":8081"`
    MetricsPort string `env:"METRICS_PORT" default:":9091"`
    
    // 观测性
    OTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" default:""`
    
    // 配置管理
    ConfigMapName  string `env:"APP_CONFIGMAP_NAME" default:""`
    DebounceMillis int    `env:"CONFIG_DEBOUNCE_MS" default:"200"`
    
    // 数据库（自动检测）
    Database DatabaseConfig
}
```

#### 多源配置

```go
// 配置优先级：环境变量 < 文件 < ConfigMap
sources := []configx.Source{
    configx.NewEnvSource(configx.EnvOptions{}),
    configx.NewFileSource("config.yaml", configx.FileOptions{
        Watch: true,
        Format: "yaml",
    }),
    configx.NewK8sConfigMapSource("app-config", configx.K8sOptions{
        Namespace: "default",
        Logger: logger,
    }),
}

manager, err := configx.NewManager(ctx, configx.Options{
    Logger: logger,
    Sources: sources,
    Debounce: 300 * time.Millisecond,
})
```

#### 热重载

```go
var cfg AppConfig
var mu sync.RWMutex

// 绑定配置并设置更新回调
err = manager.Bind(&cfg, configx.WithUpdateCallback(func() {
    mu.Lock()
    defer mu.Unlock()
    
    // 重新绑定配置
    if err := manager.Bind(&cfg); err != nil {
        logger.Error(err, "failed to reload config")
        return
    }
    
    logger.Info("configuration reloaded",
        "database_url", cfg.DatabaseURL,
        "max_conns", cfg.MaxConns,
    )
}))

// 使用配置时加锁
func getConfig() AppConfig {
    mu.RLock()
    defer mu.RUnlock()
    return cfg
}
```

### obsx - 观测性（OpenTelemetry）

#### 快速开始

```go
import "go.eggybyte.com/egg/obsx"

// 创建 OpenTelemetry Provider
provider, err := obsx.NewProvider(ctx, obsx.Options{
    ServiceName:    "user-service",
    ServiceVersion: "1.0.0",
    OTLPEndpoint:   "otel-collector:4317",  // 可选，不设置则只提供本地 Prometheus
})
if err != nil {
    log.Fatal(err)
}
defer provider.Shutdown(ctx)

// 获取 Tracer
tracer := provider.TracerProvider().Tracer("user-service")
ctx, span := tracer.Start(ctx, "GetUser")
defer span.End()

// 获取 Meter（创建自定义指标）
meter := provider.MeterProvider().Meter("user-service")
counter, _ := meter.Int64Counter("user.operations")
counter.Add(ctx, 1, attribute.String("operation", "create"))

// 暴露 Prometheus 指标端点
http.Handle("/metrics", provider.PrometheusHandler())
```

#### 在 servicex 中使用

```go
// servicex 会自动初始化 obsx（如果设置了 OTEL_EXPORTER_OTLP_ENDPOINT）
func register(app *servicex.App) error {
    provider := app.OtelProvider()
    if provider != nil {
        tracer := provider.TracerProvider().Tracer("my-service")
        // 使用 tracer...
    }
    return nil
}
```

#### 自定义指标

```go
// 创建计数器
registrationCounter, _ := meter.Int64Counter(
    "user.registrations.total",
    metric.WithDescription("Total user registrations"),
    metric.WithUnit("{registration}"),
)

// 增加计数
registrationCounter.Add(ctx, 1, 
    metric.WithAttributes(
        attribute.String("source", "web"),
        attribute.String("country", "US"),
    ),
)

// 创建直方图（用于记录延迟）
durationHistogram, _ := meter.Float64Histogram(
    "payment.process.duration",
    metric.WithDescription("Payment processing duration in seconds"),
    metric.WithUnit("s"),
)

// 记录延迟
durationHistogram.Record(ctx, 0.125, 
    metric.WithAttributes(
        attribute.String("payment_method", "credit_card"),
        attribute.String("status", "success"),
    ),
)
```

### httpx - HTTP 工具

#### 快速开始

```go
import "go.eggybyte.com/egg/httpx"

type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=2,max=50"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"required,min=18,max=120"`
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
    // 绑定并验证请求
    var req CreateUserRequest
    if err := httpx.BindAndValidate(r, &req); err != nil {
        httpx.WriteError(w, err, http.StatusBadRequest)
        return
    }
    
    // 业务逻辑
    user := createUser(req)
    
    // 返回 JSON 响应
    httpx.WriteJSON(w, http.StatusCreated, map[string]any{
        "user": user,
    })
}
```

#### 安全中间件

```go
// 添加安全头
secureHandler := httpx.SecureMiddleware(httpx.SecurityHeaders{
    ContentTypeOptions: true,  // X-Content-Type-Options: nosniff
    FrameOptions:       true,  // X-Frame-Options: DENY
    ReferrerPolicy:     true,  // Referrer-Policy: no-referrer
    StrictTransportSec: true,  // Strict-Transport-Security
    HSTSMaxAge:         31536000,  // 1 year
})

// CORS 中间件
corsHandler := httpx.CORSMiddleware(httpx.CORSOptions{
    AllowedOrigins:   []string{"https://example.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge:           3600,
})

// 应用中间件
handler := secureHandler(corsHandler(mux))
```

---

## L3: 运行时通信层

### runtimex - 运行时管理

#### 快速开始

```go
import "go.eggybyte.com/egg/runtimex"

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    })
    
    err := runtimex.Run(ctx, nil, runtimex.Options{
        Logger: logger,
        HTTP: &runtimex.HTTPOptions{
            Port: 8080,
            Mux:  mux,
        },
        Health: &runtimex.Endpoint{Port: 8081},
        Metrics: &runtimex.Endpoint{Port: 9091},
        ShutdownTimeout: 15 * time.Second,
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

#### 自定义服务

```go
type DatabaseService struct {
    db     *sql.DB
    logger log.Logger
}

func (s *DatabaseService) Start(ctx context.Context) error {
    s.logger.Info("connecting to database")
    var err error
    s.db, err = sql.Open("postgres", "connection-string")
    if err != nil {
        return err
    }
    return s.db.PingContext(ctx)
}

func (s *DatabaseService) Stop(ctx context.Context) error {
    s.logger.Info("closing database connection")
    return s.db.Close()
}

// 使用
err := runtimex.Run(ctx, []runtimex.Service{dbService}, runtimex.Options{
    Logger: logger,
    HTTP: &runtimex.HTTPOptions{Port: 8080, Mux: mux},
})
```

#### 健康检查

```go
type DatabaseHealthChecker struct {
    db *sql.DB
}

func (c *DatabaseHealthChecker) Name() string {
    return "database"
}

func (c *DatabaseHealthChecker) Check(ctx context.Context) error {
    return c.db.PingContext(ctx)
}

// 注册健康检查器
runtimex.RegisterHealthChecker(&DatabaseHealthChecker{db: db})

// 健康检查端点自动可用
// GET /healthz - 返回所有检查器的状态
```

### connectx - Connect RPC 拦截器

#### 快速开始

```go
import (
    "connectrpc.com/connect"
    "go.eggybyte.com/egg/connectx"
)

// 创建默认拦截器栈
interceptors := connectx.DefaultInterceptors(connectx.Options{
    Logger:            logger,
    Otel:              otelProvider,  // 可选，nil 则禁用追踪
    SlowRequestMillis: 1000,
    DefaultTimeoutMs:  30000,
})

// 使用拦截器
path, handler := userv1connect.NewUserServiceHandler(
    service,
    connect.WithInterceptors(interceptors...),
)
mux.Handle(path, handler)
```

#### 拦截器顺序

1. **Recovery** - 捕获 panic
2. **Timeout** - 超时控制
3. **Identity** - 身份提取
4. **Metrics** - 指标收集（如果启用 OpenTelemetry）
5. **Error Mapping** - 错误码映射
6. **Logging** - 请求日志

#### 在 servicex 中使用

```go
// servicex 自动配置拦截器
func register(app *servicex.App) error {
    path, handler := userv1connect.NewUserServiceHandler(
        service,
        connect.WithInterceptors(app.Interceptors()...),
    )
    app.Mux().Handle(path, handler)
    return nil
}
```

#### 自定义 Header 映射

```go
headers := connectx.HeaderMapping{
    RequestID:     "X-Trace-Id",
    InternalToken: "X-Internal-Auth",
    UserID:        "X-Auth-User-Id",
    UserName:      "X-Auth-User-Name",
    Roles:         "X-Auth-Roles",
}

interceptors := connectx.DefaultInterceptors(connectx.Options{
    Logger:  logger,
    Headers: headers,
})
```

### clientx - HTTP 客户端

#### 快速开始

```go
import "go.eggybyte.com/egg/clientx"

// 创建带重试和熔断器的 HTTP 客户端
httpClient := clientx.NewHTTPClient("https://api.example.com",
    clientx.WithTimeout(5*time.Second),
    clientx.WithRetry(3),
    clientx.WithCircuitBreaker(true),
)

// 创建 Connect 客户端
client := userv1connect.NewUserServiceClient(
    httpClient,
    "https://api.example.com",
)

// 发起请求
resp, err := client.GetUser(ctx, connect.NewRequest(&userv1.GetUserRequest{
    UserId: "u-123",
}))
```

#### 重试配置

```go
// 激进重试（用于关键操作）
httpClient := clientx.NewHTTPClient("https://api.example.com",
    clientx.WithTimeout(30*time.Second),
    clientx.WithRetry(5),  // 最多重试 5 次
)
```

#### 熔断器

```go
// 启用熔断器（默认开启）
httpClient := clientx.NewHTTPClient("https://api.example.com",
    clientx.WithTimeout(5*time.Second),
    clientx.WithRetry(3),
    clientx.WithCircuitBreaker(true),  // 连续 5 次失败后打开
)

// 检查熔断器状态
if errors.Is(err, gobreaker.ErrOpenState) {
    log.Println("Circuit breaker is open, skipping requests")
    time.Sleep(60 * time.Second)  // 等待熔断器关闭
}
```

---

## L4: 集成层

### servicex - 一键服务启动

#### 完整示例

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    
    "connectrpc.com/connect"
    "go.eggybyte.com/egg/configx"
    "go.eggybyte.com/egg/servicex"
    userv1 "myapp/gen/go/user/v1"
    userv1connect "myapp/gen/go/user/v1/userv1connect"
)

type AppConfig struct {
    configx.BaseConfig
    
    JWTSecret string `env:"JWT_SECRET" default:"secret"`
    SMTPHost  string `env:"SMTP_HOST" default:"localhost"`
}

type User struct {
    ID    uint   `gorm:"primarykey"`
    Name  string `gorm:"not null"`
    Email string `gorm:"uniqueIndex;not null"`
}

func register(app *servicex.App) error {
    logger := app.Logger()
    db := app.MustDB()
    
    // 创建服务
    handler := &UserServiceHandler{
        logger: logger,
        db:     db,
    }
    
    // 注册 Connect 服务
    path, svcHandler := userv1connect.NewUserServiceHandler(
        handler,
        connect.WithInterceptors(app.Interceptors()...),
    )
    app.Mux().Handle(path, svcHandler)
    
    logger.Info("user service registered", "path", path)
    return nil
}

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigChan
        cancel()
    }()
    
    cfg := &AppConfig{}
    
    err := servicex.Run(ctx,
        servicex.WithService("user-service", "1.0.0"),
        servicex.WithAppConfig(cfg),  // 自动检测数据库配置
        servicex.WithAutoMigrate(&User{}),
        servicex.WithRegister(register),
    )
    if err != nil {
        os.Exit(1)
    }
}
```

#### 数据库自动检测

```go
// 当使用 WithAppConfig 时，servicex 会自动：
// 1. 创建 configx.Manager
// 2. 从环境变量绑定配置到 cfg
// 3. 提取 BaseConfig.Database 字段
// 4. 如果 DB_DSN 设置了，自动初始化数据库连接

type AppConfig struct {
    configx.BaseConfig  // 包含 Database 字段
    CustomSetting string `env:"CUSTOM_SETTING"`
}

servicex.Run(ctx,
    servicex.WithAppConfig(cfg),  // 自动处理数据库
    servicex.WithAutoMigrate(&User{}),
    servicex.WithRegister(register),
)

// 在 register 函数中访问数据库
func register(app *servicex.App) error {
    db := app.DB()  // *gorm.DB，如果配置了数据库
    if db == nil {
        // 数据库未配置
    }
    // 使用数据库...
}
```

#### 依赖注入

```go
func register(app *servicex.App) error {
    // 注册构造函数
    app.Provide(NewUserRepository)
    app.Provide(NewUserService)
    app.Provide(func() *gorm.DB { return app.MustDB() })
    app.Provide(func() log.Logger { return app.Logger() })
    
    // 解析服务
    var userService *UserService
    if err := app.Resolve(&userService); err != nil {
        return err
    }
    
    // 注册处理器
    handler := NewUserServiceHandler(userService)
    path, svcHandler := userv1connect.NewUserServiceHandler(
        handler,
        connect.WithInterceptors(app.Interceptors()...),
    )
    app.Mux().Handle(path, svcHandler)
    
    return nil
}
```

#### 关闭钩子

```go
func register(app *servicex.App) error {
    // 创建后台工作器
    worker := NewBackgroundWorker(app.Logger())
    go worker.Start()
    
    // 注册关闭钩子（LIFO 顺序执行）
    app.AddShutdownHook(func(ctx context.Context) error {
        app.Logger().Info("stopping background worker")
        return worker.Stop(ctx)
    })
    
    return nil
}
```

---

## 辅助模块

### storex - 存储抽象

#### 快速开始

```go
import "go.eggybyte.com/egg/storex"

// 创建 MySQL 存储
store, err := storex.NewMySQLStore(
    "user:pass@tcp(localhost:3306)/mydb?parseTime=true",
    logger,
)
if err != nil {
    log.Fatal(err)
}
defer store.Close()

// 获取 GORM DB
db := store.GetDB()

// 使用 GORM
db.AutoMigrate(&User{})
db.Create(&User{Name: "John", Email: "john@example.com"})
```

#### 在 servicex 中使用

```go
// servicex 自动配置数据库
func register(app *servicex.App) error {
    db := app.DB()  // *gorm.DB
    if db == nil {
        return fmt.Errorf("database not configured")
    }
    
    // 使用数据库
    var users []User
    db.Find(&users)
    
    return nil
}
```

#### 连接池配置

```go
store, err := storex.NewGORMStore(storex.GORMOptions{
    DSN:             "user:pass@tcp(localhost:3306)/mydb",
    Driver:          "mysql",
    MaxIdleConns:    10,   // 空闲连接数
    MaxOpenConns:    100,  // 最大打开连接数
    ConnMaxLifetime: 1 * time.Hour,  // 连接最大生命周期
    Logger:          logger,
})
```

#### 多个数据库

```go
registry := storex.NewRegistry()

// 注册 MySQL
mysqlStore, _ := storex.NewMySQLStore(mysqlDSN, logger)
registry.Register("mysql", mysqlStore)

// 注册 PostgreSQL
pgStore, _ := storex.NewPostgresStore(pgDSN, logger)
registry.Register("postgres", pgStore)

// 检查所有存储
ctx := context.Background()
if err := registry.Ping(ctx); err != nil {
    log.Fatal("Some stores are unhealthy:", err)
}

// 获取特定存储
mysql, _ := registry.Get("mysql")
if gormStore, ok := mysql.(storex.GORMStore); ok {
    db := gormStore.GetDB()
    // 使用 MySQL 数据库
}
```

### k8sx - Kubernetes 集成

#### ConfigMap 监听

```go
import "go.eggybyte.com/egg/k8sx"

// 监听 ConfigMap 变化
err := k8sx.WatchConfigMap(ctx, "app-config", k8sx.WatchOptions{
    Namespace: "default",
    Logger:    logger,
}, func(data map[string]string) {
    logger.Info("config updated", "keys", len(data))
    // 更新应用配置
    updateConfig(data)
})
```

#### 服务发现

```go
// 解析 Headless 服务端点
endpoints, err := k8sx.Resolve(ctx, "my-service", k8sx.ServiceKindHeadless)
// endpoints = ["10.0.1.5:8080", "10.0.1.6:8080", "10.0.1.7:8080"]

// 解析 ClusterIP 服务
endpoints, err := k8sx.Resolve(ctx, "my-service", k8sx.ServiceKindClusterIP)
// endpoints = ["my-service.default.svc.cluster.local:8080"]
```

### testingx - 测试工具

#### Mock Logger

```go
import "go.eggybyte.com/egg/testingx"

func TestService(t *testing.T) {
    logger := testingx.NewMockLogger(t)
    
    service := NewService(logger)
    service.DoSomething()
    
    // 断言日志
    logger.AssertLogged("info", "something happened")
}
```

#### 上下文辅助

```go
import "go.eggybyte.com/egg/testingx"

func TestWithIdentity(t *testing.T) {
    ctx := testingx.NewContextWithIdentity(t, &identity.UserInfo{
        UserID: "u-1",
        Roles:  []string{"admin"},
    })
    
    // 测试代码可以使用这个上下文
    user, ok := identity.UserFrom(ctx)
    assert.True(t, ok)
    assert.Equal(t, "u-1", user.UserID)
}
```

---

## 最佳实践

### 1. 配置管理

```go
// ✅ 推荐：使用 BaseConfig
type AppConfig struct {
    configx.BaseConfig
    CustomField string `env:"CUSTOM_FIELD" default:"value"`
}

// ✅ 推荐：使用 WithAppConfig 自动检测数据库
servicex.Run(ctx,
    servicex.WithAppConfig(cfg),
    servicex.WithAutoMigrate(&User{}),
)

// ❌ 避免：手动管理配置
manager, _ := configx.DefaultManager(ctx, logger)
manager.Bind(&cfg)
servicex.Run(ctx, servicex.WithConfig(cfg), servicex.WithDatabase(...))
```

### 2. 日志记录

```go
// ✅ 推荐：使用结构化日志
logger.Info("user created",
    log.Str("user_id", user.ID),
    log.Str("email", user.Email),
)

// ❌ 避免：字符串拼接
logger.Info(fmt.Sprintf("user created: %s (%s)", user.ID, user.Email))

// ✅ 推荐：使用上下文日志
logger := logx.FromContext(ctx, baseLogger)

// ✅ 推荐：屏蔽敏感字段
logger := logx.New(
    logx.WithSensitiveFields("password", "token", "api_key"),
)
```

### 3. 错误处理

```go
// ✅ 推荐：使用结构化错误
if errors.Is(err, "NOT_FOUND") {
    return nil, errors.New("NOT_FOUND", "user not found")
}

// ✅ 推荐：包装错误保留上下文
return errors.Wrap(err, "DATABASE_ERROR", "failed to save user")

// ❌ 避免：丢失错误上下文
return fmt.Errorf("failed to save user")  // 丢失了原始错误
```

### 4. 身份验证

```go
// ✅ 推荐：在中间件中提取身份
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        ctx = identity.WithUser(ctx, extractUser(r))
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// ✅ 推荐：在业务逻辑中检查权限
if !identity.HasRole(ctx, "admin") {
    return errors.New("PERMISSION_DENIED", "admin role required")
}
```

### 5. 数据库操作

```go
// ✅ 推荐：使用 Repository 模式
type UserRepository interface {
    GetUser(ctx context.Context, id string) (*User, error)
}

// ✅ 推荐：在事务中使用上下文
func (r *UserRepository) TransferFunds(ctx context.Context, from, to string, amount int) error {
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // 转账逻辑
    })
}
```

---

## 常见场景

### 场景 1: 创建完整的微服务

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    
    "connectrpc.com/connect"
    "go.eggybyte.com/egg/configx"
    "go.eggybyte.com/egg/servicex"
    userv1connect "myapp/gen/go/user/v1/userv1connect"
)

type AppConfig struct {
    configx.BaseConfig
    CustomSetting string `env:"CUSTOM_SETTING" default:"value"`
}

type User struct {
    ID    uint   `gorm:"primarykey"`
    Name  string `gorm:"not null"`
    Email string `gorm:"uniqueIndex;not null"`
}

func register(app *servicex.App) error {
    logger := app.Logger()
    db := app.MustDB()
    
    // 创建业务服务
    repo := NewUserRepository(db)
    svc := NewUserService(repo, logger)
    handler := NewUserHandler(svc, logger)
    
    // 注册 Connect 服务
    path, svcHandler := userv1connect.NewUserServiceHandler(
        handler,
        connect.WithInterceptors(app.Interceptors()...),
    )
    app.Mux().Handle(path, svcHandler)
    
    logger.Info("service registered", "path", path)
    return nil
}

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigChan
        cancel()
    }()
    
    cfg := &AppConfig{}
    
    err := servicex.Run(ctx,
        servicex.WithService("user-service", "1.0.0"),
        servicex.WithAppConfig(cfg),
        servicex.WithAutoMigrate(&User{}),
        servicex.WithRegister(register),
    )
    if err != nil {
        os.Exit(1)
    }
}
```

### 场景 2: 调用外部服务

```go
import "go.eggybyte.com/egg/clientx"

// 创建客户端
httpClient := clientx.NewHTTPClient("https://api.example.com",
    clientx.WithTimeout(5*time.Second),
    clientx.WithRetry(3),
    clientx.WithCircuitBreaker(true),
)

client := userv1connect.NewUserServiceClient(httpClient, "https://api.example.com")

// 发起请求
resp, err := client.GetUser(ctx, connect.NewRequest(&userv1.GetUserRequest{
    UserId: "u-123",
}))
```

### 场景 3: 配置热重载

```go
var cfg AppConfig
var mu sync.RWMutex

manager, _ := configx.DefaultManager(ctx, logger)

manager.Bind(&cfg, configx.WithUpdateCallback(func() {
    mu.Lock()
    defer mu.Unlock()
    
    if err := manager.Bind(&cfg); err != nil {
        logger.Error(err, "failed to reload config")
        return
    }
    
    logger.Info("configuration reloaded")
    // 应用新配置
    applyConfig(&cfg)
}))
```

### 场景 4: 自定义指标

```go
func register(app *servicex.App) error {
    provider := app.OtelProvider()
    if provider != nil {
        meter := provider.MeterProvider().Meter("user-service")
        
        // 创建自定义指标
        counter, _ := meter.Int64Counter("user.operations")
        histogram, _ := meter.Float64Histogram("user.operation.duration")
        
        // 在业务逻辑中使用
        start := time.Now()
        counter.Add(ctx, 1, attribute.String("operation", "create"))
        
        // ... 业务逻辑 ...
        
        duration := time.Since(start).Seconds()
        histogram.Record(ctx, duration, attribute.String("operation", "create"))
    }
    
    return nil
}
```

---

## 总结

Egg 框架提供了从底层核心到高级集成的完整工具链：

- **L0 核心层**：零依赖的基础接口和类型
- **L1 基础层**：生产级日志实现
- **L2 能力层**：配置管理、观测性、HTTP 工具
- **L3 运行时层**：服务生命周期、RPC 拦截器、HTTP 客户端
- **L4 集成层**：一键服务启动，自动集成所有组件

通过 `servicex.Run()` 可以快速启动一个生产就绪的微服务，包含配置管理、日志记录、数据库连接、健康检查、指标收集等所有功能。

更多详细信息请参考各模块的 README 文档。




