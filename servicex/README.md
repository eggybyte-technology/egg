# ServiceX

ServiceX 提供统一的微服务初始化框架，让微服务能够以极简的 main 函数形式实现服务的拉起、端点注册等初始化流程。

## 特性

- **一函数启动**: 通过 `servicex.Run()` 一个函数调用启动完整的微服务
- **极简配置**: 最小化配置选项，其余功能自动默认化
- **统一初始化**: 聚合配置管理、可观测性、数据库连接和 HTTP 服务器初始化
- **优雅关闭**: 支持信号处理和优雅关闭
- **热重载**: 支持配置热重载
- **可观测性**: 集成 OpenTelemetry 和结构化日志
- **数据库支持**: 可选的数据库连接和自动迁移
- **Connect 支持**: 内置 Connect 拦截器和端点注册

## 快速开始

### 1. 基本使用

```go
package main

import (
    "context"
    "github.com/eggybyte-technology/egg/servicex"
    "github.com/eggybyte-technology/egg/configx"
)

// AppConfig 扩展基础配置
type AppConfig struct {
    configx.BaseConfig
    // 添加应用特定配置
}

func main() {
    ctx := context.Background()
    var cfg AppConfig
    
    err := servicex.Run(ctx, servicex.Options{
        ServiceName: "my-service",
        Config:      &cfg,
        Register: func(app *servicex.App) error {
            // 注册 Connect 处理器
            // path, handler := greetv1connect.NewGreeterServiceHandler(
            //     greeter,
            //     connect.WithInterceptors(app.Interceptors()...),
            // )
            // app.Mux().Handle(path, handler)
            return nil
        },
    })
    if err != nil {
        panic(err)
    }
}
```

### 2. 带数据库的服务

```go
package main

import (
    "context"
    "github.com/eggybyte-technology/egg/servicex"
    "github.com/eggybyte-technology/egg/configx"
    "gorm.io/gorm"
)

type AppConfig struct {
    configx.BaseConfig
}

func main() {
    ctx := context.Background()
    var cfg AppConfig
    
    err := servicex.Run(ctx, servicex.Options{
        ServiceName: "user-service",
        Config:      &cfg,
        Database: &servicex.DatabaseConfig{
            Driver: "mysql",
            DSN:    "user:pass@tcp(localhost:3306)/db",
        },
        Migrate: func(db *gorm.DB) error {
            return db.AutoMigrate(&User{})
        },
        Register: func(app *servicex.App) error {
            // 初始化 repository
            var userRepo UserRepository
            if db := app.DB(); db != nil {
                userRepo = NewUserRepository(db)
            }
            
            // 初始化 service 和 handler
            userService := NewUserService(userRepo, app.Logger())
            userHandler := NewUserHandler(userService, app.Logger())
            
            // 注册 Connect 处理器
            // path, handler := userv1connect.NewUserServiceHandler(
            //     userHandler,
            //     connect.WithInterceptors(app.Interceptors()...),
            // )
            // app.Mux().Handle(path, handler)
            return nil
        },
    })
    if err != nil {
        panic(err)
    }
}
```

### 3. 完整示例

```go
package main

import (
    "context"
    "fmt"
    
    "connectrpc.com/connect"
    "github.com/eggybyte-technology/egg/core/log"
    "github.com/eggybyte-technology/egg/servicex"
    greetv1connect "github.com/example/greet-service/gen/go/greet/v1/greetv1connect"
)

type AppConfig struct {
    // 应用配置
}

type GreeterService struct{}

func (s *GreeterService) SayHello(ctx context.Context, req *connect.Request[greetv1.SayHelloRequest]) (*connect.Response[greetv1.SayHelloResponse], error) {
    name := req.Msg.Name
    if name == "" {
        name = "World"
    }
    
    response := &greetv1.SayHelloResponse{
        Message: fmt.Sprintf("Hello, %s!", name),
    }
    
    return connect.NewResponse(response), nil
}

func main() {
    logger := &SimpleLogger{}
    
    err := servicex.Run(context.Background(), servicex.Options{
        ServiceName: "greet-service",
        Config:      &AppConfig{},
        Register: func(app *servicex.App) error {
            greeter := &GreeterService{}
            path, handler := greetv1connect.NewGreeterServiceHandler(
                greeter,
                connect.WithInterceptors(app.Interceptors()...),
            )
            app.Mux().Handle(path, handler)
            return nil
        },
        Logger: logger,
    })
    if err != nil {
        logger.Error(err, "Service failed")
    }
}
```

## 配置选项

### 环境变量

| 变量名 | 默认值 | 描述 |
|--------|--------|------|
| `SERVICE_NAME` | `app` | 服务名称 |
| `SERVICE_VERSION` | `0.0.0` | 服务版本 |
| `ENABLE_TRACING` | `true` | 启用链路追踪 |
| `ENABLE_HEALTH_CHECK` | `true` | 启用健康检查 |
| `ENABLE_METRICS` | `true` | 启用指标 |
| `ENABLE_DEBUG_LOGS` | `false` | 启用调试日志 |
| `SLOW_REQUEST_MILLIS` | `1000` | 慢请求阈值（毫秒） |
| `PAYLOAD_ACCOUNTING` | `true` | 启用负载统计 |
| `SHUTDOWN_TIMEOUT` | `15s` | 关闭超时时间 |

### 数据库配置

| 变量名 | 默认值 | 描述 |
|--------|--------|------|
| `DB_DRIVER` | `mysql` | 数据库驱动 |
| `DB_DSN` | `` | 数据库连接字符串 |
| `DB_MAX_IDLE` | `10` | 最大空闲连接数 |
| `DB_MAX_OPEN` | `100` | 最大打开连接数 |
| `DB_MAX_LIFETIME` | `1h` | 连接最大生存时间 |

## API 参考

### Options

```go
type Options struct {
    ServiceName    string
    ServiceVersion string
    Config         any
    Database       *DatabaseConfig
    Migrate        DatabaseMigrator
    Register       ServiceRegistrar
    EnableTracing  bool
    EnableHealthCheck bool
    EnableMetrics     bool
    EnableDebugLogs   bool
    SlowRequestMillis int64
    PayloadAccounting bool
    ShutdownTimeout   time.Duration
    Logger            log.Logger
}
```

### App

```go
type App struct {
    // 内部字段
}

func (a *App) Mux() *http.ServeMux
func (a *App) Logger() log.Logger
func (a *App) Interceptors() []interface{}
func (a *App) DB() *gorm.DB
func (a *App) OtelProvider() *obsx.Provider
```

### DatabaseConfig

```go
type DatabaseConfig struct {
    Driver      string
    DSN         string
    MaxIdle     int
    MaxOpen     int
    MaxLifetime time.Duration
}
```

## 最佳实践

1. **配置结构**: 让配置结构嵌入 `configx.BaseConfig` 以获得基础配置字段
2. **错误处理**: 在 main 函数中处理启动错误
3. **优雅关闭**: `Run` 方法自动处理优雅关闭
4. **日志**: 提供自定义 logger 以获得更好的日志体验
5. **数据库**: 仅在需要时配置数据库连接
6. **拦截器**: 使用 `app.Interceptors()` 获取预配置的 Connect 拦截器

## 与现有库的集成

ServiceX 库构建在以下现有库之上：

- `configx`: 配置管理
- `connectx`: Connect 拦截器
- `obsx`: OpenTelemetry 集成
- `core/log`: 结构化日志

这确保了与现有生态系统的完全兼容性。

## 迁移指南

### 从 Bootstrap 迁移

如果你之前使用 `bootstrap` 库，迁移到 `servicex` 非常简单：

**之前 (Bootstrap)**:
```go
bootstrapService, err := bootstrap.NewService(bootstrap.Options{
    ServiceName: "my-service",
    Config:      &cfg,
    Initializer: func(b *bootstrap.Bootstrap) error {
        // 注册处理器
        return nil
    },
})
if err != nil {
    return err
}
return bootstrapService.Run(ctx)
```

**现在 (ServiceX)**:
```go
return servicex.Run(ctx, servicex.Options{
    ServiceName: "my-service",
    Config:      &cfg,
    Register: func(app *servicex.App) error {
        // 注册处理器
        return nil
    },
})
```

主要变化：
- `Initializer` 重命名为 `Register`
- `Bootstrap` 参数改为 `App`
- 直接调用 `servicex.Run()` 而不是 `bootstrapService.Run()`
- 访问器方法保持一致：`GetMux()` → `Mux()`, `GetLogger()` → `Logger()` 等
