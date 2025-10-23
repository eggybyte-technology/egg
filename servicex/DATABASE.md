# ServiceX Database Integration

## Overview

ServiceX provides seamless database integration for microservices, handling connection management, health checks, and auto-migration automatically.

## Features

- **Automatic Connection Management**: Database connections are initialized during service startup and closed gracefully during shutdown
- **Connection Pooling**: Configurable connection pool with sensible defaults
- **Auto-Migration**: Optional automatic schema migration using GORM
- **Health Checks**: Built-in health check integration
- **Multiple Drivers**: Support for MySQL, PostgreSQL, and SQLite

## Basic Usage

### Simple Configuration

```go
package main

import (
    "context"
    "time"
    
    "github.com/eggybyte-technology/egg/servicex"
)

type User struct {
    ID        uint   `gorm:"primarykey"`
    Name      string
    Email     string `gorm:"uniqueIndex"`
    CreatedAt time.Time
    UpdatedAt time.Time
}

func main() {
    ctx := context.Background()
    
    err := servicex.Run(ctx,
        servicex.WithService("my-service", "1.0.0"),
        
        // Configure database
        servicex.WithDatabase(servicex.DatabaseConfig{
            Driver:          "mysql",
            DSN:             "user:pass@tcp(localhost:3306)/db?parseTime=true",
            MaxIdleConns:    10,
            MaxOpenConns:    100,
            ConnMaxLifetime: time.Hour,
            PingTimeout:     5 * time.Second,
        }),
        
        // Auto-migrate models
        servicex.WithAutoMigrate(&User{}),
        
        // Register services
        servicex.WithRegister(func(app *servicex.App) error {
            // Get database instance
            db := app.MustDB()
            
            // Use db for repository initialization
            repo := NewUserRepository(db)
            service := NewUserService(repo)
            
            // Register handlers...
            return nil
        }),
    )
    
    if err != nil {
        panic(err)
    }
}
```

### Optional Database

For services that support running with or without a database:

```go
servicex.WithRegister(func(app *servicex.App) error {
    var repo UserRepository
    
    db := app.DB()
    if db != nil {
        // Use database repository
        repo = NewDatabaseUserRepository(db)
    } else {
        // Use in-memory repository
        repo = NewInMemoryUserRepository()
    }
    
    service := NewUserService(repo)
    // ... register handlers
    return nil
})
```

### Configuration from Environment

You can also load database configuration from environment variables:

```go
type AppConfig struct {
    Database servicex.DatabaseConfig
}

func main() {
    var cfg AppConfig
    
    servicex.Run(ctx,
        servicex.WithConfig(&cfg),
        servicex.WithDatabase(cfg.Database),
        // ...
    )
}
```

Environment variables:
- `DB_DRIVER`: Database driver (mysql, postgres, sqlite)
- `DB_DSN`: Database connection string
- `DB_MAX_IDLE`: Maximum idle connections (default: 10)
- `DB_MAX_OPEN`: Maximum open connections (default: 100)
- `DB_MAX_LIFETIME`: Connection maximum lifetime (default: 1h)
- `DB_PING_TIMEOUT`: Connection test timeout (default: 5s)

## Database Drivers

### MySQL

```go
servicex.WithDatabase(servicex.DatabaseConfig{
    Driver: "mysql",
    DSN:    "user:pass@tcp(host:3306)/dbname?parseTime=true&loc=Local",
})
```

**Important**: Always include `parseTime=true` for proper time handling.

### PostgreSQL

```go
servicex.WithDatabase(servicex.DatabaseConfig{
    Driver: "postgres",
    DSN:    "host=localhost user=postgres password=pass dbname=mydb sslmode=disable",
})
```

### SQLite

```go
servicex.WithDatabase(servicex.DatabaseConfig{
    Driver: "sqlite",
    DSN:    "file:test.db?cache=shared&mode=memory",
})
```

## Auto-Migration

ServiceX supports automatic schema migration using GORM:

```go
// Single model
servicex.WithAutoMigrate(&User{})

// Multiple models
servicex.WithAutoMigrate(&User{}, &Post{}, &Comment{})
```

**Note**: Auto-migration only creates missing tables and columns. It does not modify existing column types or delete unused columns.

## Advanced Usage

### Custom Migration Logic

For more complex migration needs, use `app.DB()` directly:

```go
servicex.WithRegister(func(app *servicex.App) error {
    db := app.MustDB()
    
    // Custom migration
    if err := db.AutoMigrate(&User{}); err != nil {
        return err
    }
    
    // Add custom indexes
    if err := db.Exec("CREATE INDEX idx_email ON users(email)").Error; err != nil {
        return err
    }
    
    // Seed data
    var count int64
    db.Model(&User{}).Count(&count)
    if count == 0 {
        db.Create(&User{Name: "Admin", Email: "admin@example.com"})
    }
    
    return nil
})
```

### Transaction Management

```go
func (s *UserService) TransferCredits(ctx context.Context, fromID, toID uint, amount int) error {
    db := s.db.WithContext(ctx)
    
    return db.Transaction(func(tx *gorm.DB) error {
        // Deduct from sender
        if err := tx.Model(&User{}).
            Where("id = ? AND credits >= ?", fromID, amount).
            Update("credits", gorm.Expr("credits - ?", amount)).Error; err != nil {
            return err
        }
        
        // Add to receiver
        if err := tx.Model(&User{}).
            Where("id = ?", toID).
            Update("credits", gorm.Expr("credits + ?", amount)).Error; err != nil {
            return err
        }
        
        return nil
    })
}
```

### Database Health Checks

ServiceX automatically integrates database health checks. The database connection is tested during startup, and any failure will prevent the service from starting.

You can also check database health at runtime:

```go
func (s *UserService) HealthCheck(ctx context.Context) error {
    db := s.db.WithContext(ctx)
    
    sqlDB, err := db.DB()
    if err != nil {
        return err
    }
    
    return sqlDB.PingContext(ctx)
}
```

## Connection Pool Configuration

### Recommended Settings

For typical web services:

```go
servicex.WithDatabase(servicex.DatabaseConfig{
    MaxIdleConns:    10,  // Keep 10 connections ready
    MaxOpenConns:    100, // Allow up to 100 concurrent connections
    ConnMaxLifetime: time.Hour, // Recycle connections after 1 hour
})
```

For high-traffic services:

```go
servicex.WithDatabase(servicex.DatabaseConfig{
    MaxIdleConns:    50,
    MaxOpenConns:    200,
    ConnMaxLifetime: 30 * time.Minute,
})
```

For low-traffic services:

```go
servicex.WithDatabase(servicex.DatabaseConfig{
    MaxIdleConns:    2,
    MaxOpenConns:    10,
    ConnMaxLifetime: 2 * time.Hour,
})
```

## Best Practices

1. **Always use context**: Pass context to all database operations for proper timeout and cancellation handling

   ```go
   db := s.db.WithContext(ctx)
   db.Where("id = ?", id).First(&user)
   ```

2. **Use prepared statements**: GORM uses prepared statements by default, but be aware when using raw SQL

3. **Handle errors properly**: Always check and handle database errors

   ```go
   result := db.Create(&user)
   if result.Error != nil {
       return fmt.Errorf("create user: %w", result.Error)
   }
   ```

4. **Use transactions for multi-step operations**: Ensure data consistency

5. **Avoid N+1 queries**: Use `Preload` or `Joins` to load associations

   ```go
   db.Preload("Posts").Find(&users)
   ```

6. **Set appropriate timeouts**: Configure `PingTimeout` based on your network latency

7. **Monitor connection pool**: Watch for connection pool exhaustion

## Troubleshooting

### Connection Timeout

If you see connection timeout errors:

1. Check database accessibility
2. Verify DSN is correct
3. Increase `PingTimeout` if network latency is high
4. Check firewall rules

### Too Many Connections

If you hit the database connection limit:

1. Reduce `MaxOpenConns`
2. Ensure connections are not being leaked (always use `defer` or proper cleanup)
3. Increase database max connections if possible

### Slow Queries

1. Enable query logging:
   ```go
   db.Logger = logger.Default.LogMode(logger.Info)
   ```

2. Add appropriate indexes
3. Use EXPLAIN to analyze query plans

## Examples

See the complete example in `examples/user-service` for a production-ready implementation.

## License

This package is part of the EggyByte framework and is licensed under the MIT License.

