# egg/servicex

## Overview

`servicex` provides a one-call microservice bootstrap for egg-based services. It wires configuration loading, observability, HTTP server, optional database, and Connect handlers into a minimal `main` function, with graceful shutdown and optional hot reload.

## Key Features

- Single-call startup via `servicex.Run`
- Minimal configuration surface; sensible defaults
- Unified initialization: config, observability, HTTP server, optional DB
- Graceful shutdown and signal handling
- Optional hot reload via configuration updates
- Connect handler registration with preconfigured interceptors

## Basic Usage

### 1. 基本使用

```go
package main

import (
    "context"
    "github.com/eggybyte-technology/egg/servicex"
    "github.com/eggybyte-technology/egg/configx"
)

// AppConfig extends the base config
type AppConfig struct {
    configx.BaseConfig
    // Add application-specific fields as needed
}

func main() {
    ctx := context.Background()
    var cfg AppConfig
    
    err := servicex.Run(ctx, servicex.Options{
        ServiceName: "my-service",
        Config:      &cfg,
        Register: func(app *servicex.App) error {
            // Register Connect handlers (optional)
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

### 2. With Database (Simplified API)

```go
package main

import (
    "context"
    "time"
    "github.com/eggybyte-technology/egg/servicex"
)

type User struct {
    ID    uint   `gorm:"primarykey"`
    Name  string
    Email string `gorm:"uniqueIndex"`
}

func main() {
    ctx := context.Background()
    
    err := servicex.Run(ctx,
        servicex.WithService("user-service", "1.0.0"),
        
        // Configure database connection
        servicex.WithDatabase(servicex.DatabaseConfig{
            Driver:          "mysql",
            DSN:             "user:pass@tcp(localhost:3306)/db?parseTime=true",
            MaxIdleConns:    10,
            MaxOpenConns:    100,
            ConnMaxLifetime: time.Hour,
        }),
        
        // Auto-migrate models
        servicex.WithAutoMigrate(&User{}),
        
        // Register services
        servicex.WithRegister(func(app *servicex.App) error {
            // Get database instance
            db := app.MustDB()
            
            // Initialize repository, service, and handler
            userRepo := NewUserRepository(db)
            userService := NewUserService(userRepo, app.Logger())
            userHandler := NewUserHandler(userService, app.Logger())
            
            // Register Connect handlers
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

### 3. Complete example

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
    // Application config
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

## Database Integration

ServiceX provides seamless database integration with automatic connection management, health checks, and auto-migration support.

### Features

- **Automatic Connection Management**: Database connections are initialized during startup and closed gracefully during shutdown
- **Connection Pooling**: Configurable connection pool with sensible defaults
- **Auto-Migration**: Optional automatic schema migration using GORM
- **Health Checks**: Built-in database health check integration
- **Multiple Drivers**: Support for MySQL, PostgreSQL, and SQLite

### Quick Start

```go
import (
    "time"
    "github.com/eggybyte-technology/egg/servicex"
)

type User struct {
    ID    uint   `gorm:"primarykey"`
    Name  string
    Email string `gorm:"uniqueIndex"`
}

err := servicex.Run(ctx,
    servicex.WithService("user-service", "1.0.0"),
    
    // Configure database
    servicex.WithDatabase(servicex.DatabaseConfig{
        Driver:          "mysql",
        DSN:             "user:pass@tcp(localhost:3306)/db?parseTime=true",
        MaxIdleConns:    10,
        MaxOpenConns:    100,
        ConnMaxLifetime: time.Hour,
    }),
    
    // Auto-migrate models
    servicex.WithAutoMigrate(&User{}),
    
    // Use database in registration
    servicex.WithRegister(func(app *servicex.App) error {
        db := app.MustDB() // Get database or panic if not configured
        // or
        db := app.DB() // Get database or nil if not configured
        
        // Use db for repositories...
        return nil
    }),
)
```

### Database Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Driver` | string | "mysql" | Database driver: mysql, postgres, or sqlite |
| `DSN` | string | "" | Database connection string |
| `MaxIdleConns` | int | 10 | Maximum idle connections in pool |
| `MaxOpenConns` | int | 100 | Maximum open connections |
| `ConnMaxLifetime` | time.Duration | 1h | Maximum connection lifetime |
| `PingTimeout` | time.Duration | 5s | Connection test timeout |

### Database Drivers

**MySQL**:
```go
servicex.WithDatabase(servicex.DatabaseConfig{
    Driver: "mysql",
    DSN:    "user:pass@tcp(host:3306)/dbname?parseTime=true&loc=Local",
})
```

**PostgreSQL**:
```go
servicex.WithDatabase(servicex.DatabaseConfig{
    Driver: "postgres",
    DSN:    "host=localhost user=postgres password=pass dbname=mydb sslmode=disable",
})
```

**SQLite**:
```go
servicex.WithDatabase(servicex.DatabaseConfig{
    Driver: "sqlite",
    DSN:    "file:test.db?cache=shared&mode=memory",
})
```

### Auto-Migration

ServiceX supports automatic schema migration:

```go
// Single model
servicex.WithAutoMigrate(&User{})

// Multiple models
servicex.WithAutoMigrate(&User{}, &Post{}, &Comment{})
```

For more advanced usage and examples, see [DATABASE.md](DATABASE.md).

## Configuration Options

### Environment variables

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `SERVICE_NAME` | `app` | Service name |
| `SERVICE_VERSION` | `0.0.0` | Service version |
| `ENABLE_TRACING` | `true` | Enable tracing |
| `ENABLE_HEALTH_CHECK` | `true` | Enable health check |
| `ENABLE_METRICS` | `true` | Enable metrics |
| `ENABLE_DEBUG_LOGS` | `false` | Enable debug logs |
| `SLOW_REQUEST_MILLIS` | `1000` | Slow request threshold (ms) |
| `PAYLOAD_ACCOUNTING` | `true` | Enable payload accounting |
| `SHUTDOWN_TIMEOUT` | `15s` | Shutdown timeout |

### Database

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `DB_DRIVER` | `mysql` | Database driver |
| `DB_DSN` | `` | Database DSN |
| `DB_MAX_IDLE` | `10` | Max idle connections |
| `DB_MAX_OPEN` | `100` | Max open connections |
| `DB_MAX_LIFETIME` | `1h` | Max connection lifetime |

## API Reference

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
    // internal fields
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

## Best Practices

1. Embed `configx.BaseConfig` for common fields
2. Handle startup errors in `main`
3. Rely on `Run` for graceful shutdown
4. Provide a custom logger where appropriate
5. Configure database only when needed
6. Use `app.Interceptors()` for preconfigured Connect interceptors

## Dependencies

Built on: `configx`, `connectx`, `obsx`, `core/log`

## Migration Guide

### From Bootstrap

If you used a previous bootstrap library, migrating to `servicex` is straightforward:

Before (Bootstrap):
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

Now (ServiceX):
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

Key changes:
- `Initializer` renamed to `Register`
- Replace `Bootstrap` with `App`
- Call `servicex.Run()` instead of `bootstrapService.Run()`
- Accessor methods remain consistent: `GetMux()` → `Mux()`, `GetLogger()` → `Logger()`

## Stability

Stable since v0.1.0.

## License

This package is part of the EggyByte framework and is licensed under the MIT License. See the root LICENSE file for details.
