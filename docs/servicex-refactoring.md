# ServiceX 重构总结

## 概述

本次重构优化了 `servicex` 模块的结构，使其更好地集成 egg 生态中的其他库，并使用 `internal` 文件夹清晰地组织代码结构。

## 主要改进

### 1. 清晰的目录结构

**之前：**
```
servicex/
├── app.go
├── container.go        # 暴露的容器实现
├── database.go
├── interceptors.go    # 暴露的拦截器实现
├── options.go
├── servicex.go
└── utils.go
```

**现在：**
```
servicex/
├── internal/              # 内部实现细节
│   ├── container.go       # DI 容器实现
│   └── interceptors.go   # 拦截器构建逻辑
├── app.go                 # 公开的 App API
├── database.go            # 公开的数据库配置 API
├── options.go             # 公开的配置选项
├── servicex.go            # 公开的 Run 函数
└── utils.go               # 辅助函数
```

### 2. 更好的模块集成

#### 使用 storex 管理数据库

**之前：** servicex 自行创建数据库连接

**现在：** 使用 `storex.NewGORMStore` 统一管理

```go
// servicex.go
store, err = storex.NewGORMStore(storex.GORMOptions{
    DSN:             cfg.dbConfig.DSN,
    Driver:          cfg.dbConfig.Driver,
    MaxIdleConns:    cfg.dbConfig.MaxIdleConns,
    MaxOpenConns:    cfg.dbConfig.MaxOpenConns,
    ConnMaxLifetime: cfg.dbConfig.ConnMaxLifetime,
    Logger:          cfg.logger,
})
```

#### 使用 connectx 构建拦截器

**之前：** servicex 直接构建拦截器

**现在：** 通过 `internal.BuildInterceptors` 调用 `connectx.DefaultInterceptors`

```go
// internal/interceptors.go
func BuildInterceptors(logger log.Logger, otel *obsx.Provider, 
    slowRequestMillis int64, enableDebugLogs, payloadAccounting bool) []connect.Interceptor {
    connectxOpts := connectx.Options{
        Logger:            logger,
        Otel:              otel,
        WithRequestBody:   enableDebugLogs,
        WithResponseBody:  enableDebugLogs,
        SlowRequestMillis: slowRequestMillis,
        PayloadAccounting: payloadAccounting,
    }
    return connectx.DefaultInterceptors(connectxOpts)
}
```

#### 使用 configx 管理配置

通过 `configx.BaseConfig` 和 `configx.DefaultManager` 统一配置管理。

### 3. 代码组织改进

#### Container 移动到 internal

```go
// internal/container.go
type Container struct {
    mu           sync.RWMutex
    constructors map[reflect.Type]reflect.Value
    instances    map[reflect.Type]reflect.Value
    building     map[reflect.Type]bool
}

func NewContainer() *Container {
    return &Container{...}
}
```

#### Interceptors 构建逻辑移动到 internal

所有拦截器构建相关的实现细节都封装在 `internal/interceptors.go` 中。

### 4. 公开 API 保持简洁

#### App 结构体

```go
// app.go
type App struct {
    mux           *http.ServeMux
    logger        log.Logger
    interceptors  []connect.Interceptor
    otel          *obsx.Provider
    container     *internal.Container      // 使用 internal
    shutdownHooks []func(context.Context) error
    db            *gorm.DB
}
```

#### Run 函数

```go
// servicex.go
func Run(ctx context.Context, opts ...Option) error {
    // ...配置处理...
    
    // 使用 internal 包的功能
    interceptors := internal.BuildInterceptors(...)
    app := &App{
        container: internal.NewContainer(),
        ...
    }
    
    // ...
}
```

## 技术细节

### Internal 包的作用

`internal` 包包含：
1. **Container**: DI 容器实现
2. **BuildInterceptors**: 拦截器构建逻辑

这些实现细节对用户是不可见的，用户只需要使用公开的 API。

### 库集成点

| 模块 | 集成点 | 说明 |
|------|--------|------|
| **storex** | 数据库连接管理 | 使用 `storex.NewGORMStore` 创建和关闭连接 |
| **configx** | 配置管理 | 使用 `configx.BaseConfig` 和 `configx.DefaultManager` |
| **connectx** | 拦截器栈 | 使用 `connectx.DefaultInterceptors` 构建拦截器 |
| **obsx** | 观测性 | 使用 `obsx.NewProvider` 和 `obsx.Provider.Shutdown` |
| **logx** | 日志 | 使用 `logx.New` 创建日志器 |
| **runtimex** | 运行时 | 使用 `runtimex.CheckHealth` 进行健康检查 |

## 优势

1. **清晰的职责分离**: 内部实现与公开 API 分离
2. **更好的封装**: 实现细节隐藏在 internal 包中
3. **统一的库集成**: 使用其他 egg 库的标准功能
4. **易于维护**: 代码结构清晰，职责明确
5. **向后兼容**: 公开 API 保持不变

## 使用示例

```go
package main

import (
    "context"
    "github.com/eggybyte-technology/egg/configx"
    "github.com/eggybyte-technology/egg/servicex"
)

type AppConfig struct {
    configx.BaseConfig
}

func main() {
    ctx := context.Background()
    cfg := &AppConfig{}
    
    err := servicex.Run(ctx,
        servicex.WithService("my-service", "1.0.0"),
        servicex.WithConfig(cfg),
        servicex.WithDatabase(servicex.FromBaseConfig(&cfg.Database)),
        servicex.WithAutoMigrate(&User{}),
        servicex.WithRegister(func(app *servicex.App) error {
            // app 提供所有需要的方法
            db := app.DB()
            logger := app.Logger()
            mux := app.Mux()
            interceptors := app.Interceptors()
            // ...
            return nil
        }),
    )
    if err != nil {
        panic(err)
    }
}
```

## 迁移指南

**无需更改代码！** 公开 API 完全兼容，现有的使用方式不需要修改。

唯一的内部变化是：
- `newContainer()` → `internal.NewContainer()`
- `buildInterceptors()` → `internal.BuildInterceptors()`

但这些变化对用户是不可见的。

## 总结

本次重构让 `servicex` 的结构更加清晰，更好地利用了 egg 生态中其他库的功能，并为未来的扩展打下了良好的基础。

