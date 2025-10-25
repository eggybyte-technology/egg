# egg/storex

## Overview

`storex` provides storage interfaces and GORM integration with health check support.
It offers a clean abstraction for database operations with automatic connection pooling,
health monitoring, and registry management.

## Key Features

- Clean storage interface abstraction
- GORM integration for SQL databases
- Connection health checks
- Registry for multiple storage backends
- Configurable connection pooling
- Support for MySQL, PostgreSQL, SQLite
- Clean separation of interface and implementation

## Dependencies

Layer: **Auxiliary (Storage Layer)**  
Depends on: `core/log`, `gorm.io/gorm`, database drivers

## Installation

```bash
go get github.com/eggybyte-technology/egg/storex@latest
```

## Basic Usage

```go
import (
    "context"
    "github.com/eggybyte-technology/egg/storex"
)

func main() {
    // Create GORM store
    store, err := storex.NewGORMStore(storex.GORMOptions{
        DSN:    "user:pass@tcp(localhost:3306)/mydb?parseTime=true",
        Driver: "mysql",
        Logger: logger,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Get GORM DB instance
    db := store.GetDB()
    
    // Use GORM normally
    var users []User
    db.Find(&users)
}
```

## API Reference

### Store Interface

```go
// Store defines the interface for storage backends
type Store interface {
    // Ping checks if the storage backend is healthy
    Ping(ctx context.Context) error
    
    // Close closes the storage connection
    Close() error
}
```

### GORMStore Interface

```go
// GORMStore extends Store with GORM-specific functionality
type GORMStore interface {
    Store
    
    // GetDB returns the underlying GORM database instance
    GetDB() *gorm.DB
}
```

### Registry

```go
// Registry manages multiple storage connections and their health
type Registry struct {
    // ... internal fields
}

// NewRegistry creates a new storage registry
func NewRegistry() *Registry

// Register registers a storage backend with the given name
func (r *Registry) Register(name string, store Store) error

// Unregister removes a storage backend from the registry
func (r *Registry) Unregister(name string) error

// Ping performs health checks on all registered storage backends
func (r *Registry) Ping(ctx context.Context) error

// Close closes all registered storage connections
func (r *Registry) Close() error

// List returns the names of all registered stores
func (r *Registry) List() []string

// Get returns a registered store by name
func (r *Registry) Get(name string) (Store, bool)
```

### GORM Options

```go
type GORMOptions struct {
    DSN             string        // Database connection string
    Driver          string        // Database driver (mysql, postgres, sqlite)
    MaxIdleConns    int           // Maximum number of idle connections
    MaxOpenConns    int           // Maximum number of open connections
    ConnMaxLifetime time.Duration // Maximum connection lifetime
    Logger          log.Logger    // Logger for database operations
}
```

### Constructor Functions

```go
// NewGORMStore creates a new GORM store with the given options
func NewGORMStore(opts GORMOptions) (GORMStore, error)

// NewMySQLStore creates a new MySQL store with the given DSN
func NewMySQLStore(dsn string, logger log.Logger) (GORMStore, error)

// NewPostgresStore creates a new PostgreSQL store with the given DSN
func NewPostgresStore(dsn string, logger log.Logger) (GORMStore, error)

// NewSQLiteStore creates a new SQLite store with the given DSN
func NewSQLiteStore(dsn string, logger log.Logger) (GORMStore, error)
```

## Architecture

The storex module provides storage abstraction:

```
storex/
├── storex.go            # Public API (~143 lines)
│   ├── Store            # Storage interface
│   ├── GORMStore        # GORM-specific interface
│   ├── Registry         # Registry wrapper
│   └── Constructors     # NewGORMStore, NewMySQLStore, etc.
└── internal/
    ├── gorm.go          # GORM implementation
    │   └── gormStore    # GORM store implementation
    └── registry.go      # Registry implementation
        └── registryImpl # Registry with health checks
```

**Design Highlights:**
- Public interfaces define contracts
- GORM implementation isolated in internal package
- Registry supports multiple backends
- Health checks for monitoring

## Example: MySQL Database

```go
package main

import (
    "context"
    "github.com/eggybyte-technology/egg/storex"
    "gorm.io/gorm"
)

type User struct {
    ID    uint   `gorm:"primarykey"`
    Name  string `gorm:"not null"`
    Email string `gorm:"uniqueIndex;not null"`
}

func main() {
    // Create MySQL store
    store, err := storex.NewMySQLStore(
        "user:pass@tcp(localhost:3306)/mydb?parseTime=true",
        logger,
    )
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Check health
    ctx := context.Background()
    if err := store.Ping(ctx); err != nil {
        log.Fatal("Database not healthy:", err)
    }
    
    // Get GORM DB
    db := store.GetDB()
    
    // Auto-migrate
    db.AutoMigrate(&User{})
    
    // Create user
    user := &User{Name: "John", Email: "john@example.com"}
    result := db.Create(user)
    if result.Error != nil {
        log.Fatal(result.Error)
    }
    
    // Query users
    var users []User
    db.Find(&users)
    for _, u := range users {
        fmt.Printf("User: %s (%s)\n", u.Name, u.Email)
    }
}
```

## Example: Connection Pooling

```go
// Configure connection pool for production
store, err := storex.NewGORMStore(storex.GORMOptions{
    DSN:             "user:pass@tcp(localhost:3306)/mydb?parseTime=true",
    Driver:          "mysql",
    MaxIdleConns:    10,   // Idle connections in pool
    MaxOpenConns:    100,  // Maximum open connections
    ConnMaxLifetime: 1 * time.Hour,  // Recycle connections after 1 hour
    Logger:          logger,
})
```

## Example: Multiple Databases

```go
func main() {
    // Create registry
    registry := storex.NewRegistry()
    
    // Register MySQL store
    mysqlStore, _ := storex.NewMySQLStore(
        "user:pass@tcp(localhost:3306)/mydb",
        logger,
    )
    registry.Register("mysql", mysqlStore)
    
    // Register PostgreSQL store
    pgStore, _ := storex.NewPostgresStore(
        "postgres://user:pass@localhost:5432/mydb",
        logger,
    )
    registry.Register("postgres", pgStore)
    
    // Check all stores
    ctx := context.Background()
    if err := registry.Ping(ctx); err != nil {
        log.Fatal("Some stores are unhealthy:", err)
    }
    
    // Get specific store
    mysql, _ := registry.Get("mysql")
    if gormStore, ok := mysql.(storex.GORMStore); ok {
        db := gormStore.GetDB()
        // Use MySQL database
    }
    
    // Cleanup
    registry.Close()
}
```

## Example: Health Checks

```go
import "github.com/eggybyte-technology/egg/runtimex"

type DatabaseHealthChecker struct {
    store storex.Store
}

func (c *DatabaseHealthChecker) Name() string {
    return "database"
}

func (c *DatabaseHealthChecker) Check(ctx context.Context) error {
    return c.store.Ping(ctx)
}

func main() {
    store, _ := storex.NewMySQLStore(dsn, logger)
    
    // Register health checker
    checker := &DatabaseHealthChecker{store: store}
    runtimex.RegisterHealthChecker(checker)
    
    // Health endpoint will now check database connectivity
}
```

## Example: Transaction Support

```go
func transferFunds(db *gorm.DB, from, to string, amount int) error {
    return db.Transaction(func(tx *gorm.DB) error {
        // Debit from account
        result := tx.Model(&Account{}).
            Where("id = ? AND balance >= ?", from, amount).
            Update("balance", gorm.Expr("balance - ?", amount))
        if result.Error != nil {
            return result.Error
        }
        if result.RowsAffected == 0 {
            return errors.New("insufficient funds")
        }
        
        // Credit to account
        if err := tx.Model(&Account{}).
            Where("id = ?", to).
            Update("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
            return err
        }
        
        return nil
    })
}

func main() {
    store, _ := storex.NewMySQLStore(dsn, logger)
    db := store.GetDB()
    
    if err := transferFunds(db, "acc-1", "acc-2", 100); err != nil {
        log.Fatal(err)
    }
}
```

## Integration with servicex

storex is automatically integrated in servicex:

```go
import "github.com/eggybyte-technology/egg/servicex"

type User struct {
    ID    uint   `gorm:"primarykey"`
    Name  string `gorm:"not null"`
    Email string `gorm:"uniqueIndex;not null"`
}

func main() {
    cfg := &AppConfig{}
    
    err := servicex.Run(ctx,
        servicex.WithConfig(cfg),
        servicex.WithDatabase(servicex.FromBaseConfig(&cfg.Database)),
        servicex.WithAutoMigrate(&User{}),  // Auto-migrate models
        servicex.WithRegister(register),
    )
}

func register(app *servicex.App) error {
    // Get GORM DB instance
    db := app.MustDB()
    
    // Use database
    var users []User
    db.Find(&users)
    
    return nil
}
```

## Database Drivers

### MySQL

```bash
go get gorm.io/driver/mysql
```

```go
import "gorm.io/driver/mysql"

dsn := "user:pass@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
store, _ := storex.NewMySQLStore(dsn, logger)
```

### PostgreSQL

```bash
go get gorm.io/driver/postgres
```

```go
import "gorm.io/driver/postgres"

dsn := "host=localhost user=user password=pass dbname=mydb port=5432 sslmode=disable"
store, _ := storex.NewPostgresStore(dsn, logger)
```

### SQLite

```bash
go get gorm.io/driver/sqlite
```

```go
import "gorm.io/driver/sqlite"

dsn := "./test.db"
store, _ := storex.NewSQLiteStore(dsn, logger)
```

## Best Practices

1. **Always ping after creation** - Verify connection before use
2. **Configure connection pooling** - Set appropriate limits for your workload
3. **Use transactions** - For operations that must be atomic
4. **Close connections** - Always defer `Close()` after creation
5. **Health check registration** - Register with runtimex for monitoring
6. **Connection lifecycle** - Recycle connections periodically
7. **Error handling** - Check GORM errors and handle appropriately

## Performance Considerations

- **Connection Pool Size**: Tune `MaxIdleConns` and `MaxOpenConns` based on workload
- **Connection Lifetime**: Recycle connections to prevent stale connections
- **Query Optimization**: Use indexes, avoid N+1 queries
- **Transaction Scope**: Keep transactions as short as possible
- **Prepared Statements**: GORM uses prepared statements by default

## Testing

```go
func TestUserRepository(t *testing.T) {
    // Create SQLite in-memory database for testing
    store, err := storex.NewSQLiteStore(":memory:", logger)
    require.NoError(t, err)
    defer store.Close()
    
    db := store.GetDB()
    
    // Migrate schema
    db.AutoMigrate(&User{})
    
    // Test operations
    user := &User{Name: "Test", Email: "test@example.com"}
    result := db.Create(user)
    require.NoError(t, result.Error)
    assert.NotZero(t, user.ID)
    
    // Query
    var found User
    db.First(&found, user.ID)
    assert.Equal(t, "Test", found.Name)
}
```

## Stability

**Status**: Stable  
**Layer**: Auxiliary (Storage)  
**API Guarantees**: Backward-compatible changes only

The storex module is production-ready and follows semantic versioning.

## License

This package is part of the egg framework and is licensed under the MIT License.
See the root LICENSE file for details.
