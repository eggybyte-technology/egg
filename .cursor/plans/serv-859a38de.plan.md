<!-- 859a38de-7b4e-4573-bda8-0814a782a943 c9384f1a-50a0-495f-85af-df65ab5a0e7d -->
# servicex：一函数启动微服务的聚合入口

### 目标

- 在 `servicex/` 新增模块，聚合 `bootstrap`、`connectx`、`configx`、`obsx` 等，导出极简 API：`servicex.Run(ctx, Options{...})`。
- 保留唯一扩展点：`Register(app *servicex.App) error`；其余（配置读取、健康检查、指标、Tracing、Connect 拦截器、优雅停机、DB 可选迁移等）全部默认化。
- 将 `examples/minimal-connect-service` 与 `examples/user-service` 的 `main.go` 改为基于 `servicex` 的最简写法。

### 关键设计

- 包路径：`go.eggybyte.com/egg/servicex`
- 核心类型：
  - `type Options struct { ServiceName string; Config any; Logger log.Logger; EnableTracing bool; EnableHealthCheck bool; EnableMetrics bool; EnableDebugLogs bool; SlowRequestMillis int64; PayloadAccounting bool; ShutdownTimeout time.Duration; Database *DatabaseConfig; Migrate func(db *gorm.DB) error; Register func(app *App) error }`
  - `type App struct { ... }`（只暴露最小访问器）
    - `Mux() *http.ServeMux`
    - `Logger() log.Logger`
    - `Interceptors() []connect.Interceptor`
    - `DB() *gorm.DB`
    - `OtelProvider() obsx.Provider`（或等价抽象）
  - `func Run(ctx context.Context, o Options) error`
- 默认拦截器：由 `connectx.DefaultInterceptors` 构造，自动注入 Logger、OTel、慢请求阈值、payload 统计、是否输出 body 等。
- DB 支持：可选 `DatabaseConfig` + `Migrate`，当配置存在时自动初始化与迁移。
- `Run` 内部映射为 `bootstrap.NewService(bootstrap.Options{...}).Run(ctx)`，但对外隐藏 bootstrap 细节。

### 最简用法（示例片段）

- 最小 Greet 服务：
```go
func main() {
    ctx := context.Background()
    var cfg AppConfig
    err := servicex.Run(ctx, servicex.Options{
        ServiceName: "greet-service",
        Config:      &cfg,
        Register: func(app *servicex.App) error {
            greeter := &GreeterService{}
            path, h := greetv1connect.NewGreeterServiceHandler(
                greeter,
                connect.WithInterceptors(app.Interceptors()...),
            )
            app.Mux().Handle(path, h)
            return nil
        },
    })
    if err != nil { panic(err) }
}
```

- 用户服务（含 DB 与迁移）：
```go
err := servicex.Run(ctx, servicex.Options{
    ServiceName: "user-service",
    Config:      &cfg,
    Database:    &servicex.DatabaseConfig{Driver: "mysql", DSN: "", MaxIdle:10, MaxOpen:100, MaxLifetime: time.Hour},
    Migrate: func(db *gorm.DB) error { return db.AutoMigrate(&model.User{}) },
    Register: func(app *servicex.App) error {
        var repo repository.UserRepository
        if db := app.DB(); db != nil { repo = repository.NewUserRepository(db) }
        svc := service.NewUserService(repo, app.Logger())
        h := handler.NewUserHandler(svc, app.Logger())
        path, ch := userv1connect.NewUserServiceHandler(h, connect.WithInterceptors(app.Interceptors()...))
        app.Mux().Handle(path, ch)
        return nil
    },
})
```


### 具体改动

- 新增目录与文件（带完整 GoDoc，分工清晰）
  - `servicex/doc.go`（包级文档）
  - `servicex/options.go`（Options、DatabaseConfig 定义与校验）
  - `servicex/app.go`（App 封装与访问器）
  - `servicex/interceptors.go`（默认拦截器装配）
  - `servicex/run.go`（Run 实现，映射到 bootstrap）
  - `servicex/README.md`（快速上手与进阶）
  - `servicex/servicex_test.go`（核心单测：选项映射、拦截器、无 DB/有 DB）
- 初始化模块：通过 `go mod init go.eggybyte.com/egg/servicex`，并 `go work use ./servicex`。
- 改造两个示例的 `main.go` 以使用 `servicex`。
- 更新 `docs/guide.md` 一章，推荐新入口；保留 bootstrap 文档作为高级用法。

### 兼容性

- 现有 `bootstrap` 无破坏；`servicex` 为更上层的易用包装。

### 验证

- 本地跑通两示例（无 DB 与有 DB）；观察健康检查、指标、Tracing、生效的 Connect 拦截器。

### To-dos

- [ ] 创建 servicex 模块与基础目录、go.mod、doc.go
- [ ] 实现 Options、App 访问器与 Run 壳子
- [ ] 将 Options 映射到 bootstrap.Options 并打通运行
- [ ] 集成 connectx 默认拦截器与选项
- [ ] 可选 DB 初始化与迁移钩子
- [ ] 为选项映射、拦截器、DB 分支添加单测
- [ ] 改造 minimal-connect-service 使用 servicex
- [ ] 改造 user-service 使用 servicex（含迁移）
- [ ] 更新 docs/guide.md 与新增 servicex/README.md
- [ ] 后续将 CLI 模板切换为 servicex 风格