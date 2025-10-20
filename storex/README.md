# StoreX Module

<div align="center">

**Storage adapters and database integration for Egg services**

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg)](LICENSE)

</div>

## 📦 Overview

The `storex` module provides storage adapters and database integration for Egg services. It offers GORM integration supporting MySQL, PostgreSQL, and SQLite with connection management and health probes.

## ✨ Features

- 🗄️ **Database Adapters** - GORM integration for multiple databases
- 🔌 **Connection Management** - Automatic connection pooling and management
- ❤️ **Health Probes** - Database health check endpoints
- 🔄 **Migration Support** - Database migration utilities
- 📝 **Structured Logging** - Context-aware logging
- 🛡️ **Error Handling** - Robust error handling and recovery
- 🎯 **Easy Configuration** - Simple setup and configuration
- 🔧 **Multiple Databases** - MySQL, PostgreSQL, SQLite support

## 🏗️ Architecture

```
storex/
├── storex.go        # Main storage interface
├── internal/
│   └── gorm.go      # GORM adapter implementation
└── storex_test.go   # Tests
```

## 🚀 Quick Start

### Installation

```bash
go get github.com/eggybyte-technology/egg/storex@latest
```

### Basic Usage

```go
package main

import (
    "context"
    "time"

    "github.com/eggybyte-technology/egg/storex"
    "github.com/eggybyte-technology/egg/core/log"
    "gorm.io/gorm"
)

// User model
type User struct {
    ID        uint      `gorm:"primaryKey"`
    Email     string    `gorm:"uniqueIndex;not null"`
    Name      string    `gorm:"not null"`
    CreatedAt time.Time
    UpdatedAt time.Time
}

func main() {
    // Create logger
    logger := &YourLogger{} // Implement log.Logger interface

    // Create storage client
    ctx := context.Background()
    client, err := storex.NewClient(ctx, storex.Options{
        Driver:   "postgres",
        DSN:      "postgres://user:pass@localhost/db?sslmode=disable",
        Logger:   logger,
    })
    if err != nil {
        log.Fatal("Failed to create storage client:", err)
    }
    defer client.Close()

    // Auto-migrate models
    if err := client.AutoMigrate(&User{}); err != nil {
        log.Fatal("Failed to migrate:", err)
    }

    // Use storage
    userService := &UserService{db: client.DB()}
    
    // Create user
    user, err := userService.CreateUser(ctx, &User{
        Email: "user@example.com",
        Name:  "John Doe",
    })
    if err != nil {
        log.Fatal("Failed to create user:", err)
    }

    log.Info("User created", "id", user.ID, "email", user.Email)
}
```

## 📖 API Reference

### Client Options

```go
type Options struct {
    Driver   string
    DSN      string
    Logger   log.Logger
    MaxIdle  int
    MaxOpen  int
    MaxLifetime time.Duration
}

type Client interface {
    DB() *gorm.DB
    Close() error
    Health(ctx context.Context) error
    AutoMigrate(models ...interface{}) error
}
```

### Main Functions

```go
// NewClient creates a new storage client
func NewClient(ctx context.Context, opts Options) (Client, error)

// DefaultClient creates a client with default options
func DefaultClient(ctx context.Context, driver, dsn string) (Client, error)
```

## 🔧 Configuration

### Environment Variables

```bash
# Database configuration
export DB_DRIVER="postgres"
export DB_DSN="postgres://user:pass@localhost/db?sslmode=disable"
export DB_MAX_IDLE="10"
export DB_MAX_OPEN="100"
export DB_MAX_LIFETIME="1h"

# MySQL example
export DB_DRIVER="mysql"
export DB_DSN="user:pass@tcp(localhost:3306)/db?charset=utf8mb4&parseTime=True&loc=Local"

# SQLite example
export DB_DRIVER="sqlite"
export DB_DSN="file:test.db?cache=shared&mode=memory"
```

### Configuration File

```yaml
# config.yaml
database:
  driver: "postgres"
  dsn: "postgres://user:pass@localhost/db?sslmode=disable"
  max_idle: 10
  max_open: 100
  max_lifetime: "1h"
```

## 🛠️ Advanced Usage

### Custom Models

```go
// User model with custom methods
type User struct {
    ID        uint      `gorm:"primaryKey"`
    Email     string    `gorm:"uniqueIndex;not null"`
    Name      string    `gorm:"not null"`
    CreatedAt time.Time
    UpdatedAt time.Time
}

// Custom methods
func (u *User) BeforeCreate(tx *gorm.DB) error {
    // Custom validation
    if u.Email == "" {
        return errors.New("email is required")
    }
    return nil
}

func (u *User) AfterCreate(tx *gorm.DB) error {
    // Custom logic after creation
    log.Info("User created", "id", u.ID, "email", u.Email)
    return nil
}
```

### Repository Pattern

```go
// UserRepository implements user data access
type UserRepository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
    return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *User) error {
    return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) GetByID(ctx context.Context, id uint) (*User, error) {
    var user User
    err := r.db.WithContext(ctx).First(&user, id).Error
    if err != nil {
        return nil, err
    }
    return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
    var user User
    err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
    if err != nil {
        return nil, err
    }
    return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *User) error {
    return r.db.WithContext(ctx).Save(user).Error
}

func (r *UserRepository) Delete(ctx context.Context, id uint) error {
    return r.db.WithContext(ctx).Delete(&User{}, id).Error
}
```

### Transaction Support

```go
func (s *UserService) CreateUserWithProfile(ctx context.Context, user *User, profile *Profile) error {
    return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // Create user
        if err := tx.Create(user).Error; err != nil {
            return err
        }

        // Create profile
        profile.UserID = user.ID
        if err := tx.Create(profile).Error; err != nil {
            return err
        }

        return nil
    })
}
```

### Health Checks

```go
func main() {
    // Create client
    client, err := storex.NewClient(ctx, storex.Options{
        Driver: "postgres",
        DSN:    "postgres://user:pass@localhost/db?sslmode=disable",
        Logger: logger,
    })
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }
    defer client.Close()

    // Health check
    if err := client.Health(ctx); err != nil {
        log.Error("Database health check failed:", err)
    } else {
        log.Info("Database is healthy")
    }

    // Periodic health checks
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-ticker.C:
                if err := client.Health(ctx); err != nil {
                    log.Error("Database health check failed:", err)
                }
            case <-ctx.Done():
                return
            }
        }
    }()
}
```

## 🔧 Integration with Other Modules

### ConfigX Integration

```go
func main() {
    // Load configuration
    var cfg struct {
        Database struct {
            Driver string `env:"DB_DRIVER" default:"postgres"`
            DSN    string `env:"DB_DSN" required:"true"`
        }
    }

    mgr, _ := configx.DefaultManager(ctx, logger)
    _ = mgr.Bind(&cfg)

    // Create storage client with configuration
    client, err := storex.NewClient(ctx, storex.Options{
        Driver: cfg.Database.Driver,
        DSN:    cfg.Database.DSN,
        Logger: logger,
    })
    if err != nil {
        log.Fatal("Failed to create storage client:", err)
    }
    defer client.Close()
}
```

### RuntimeX Integration

```go
func main() {
    // Create storage client
    client, err := storex.NewClient(ctx, storex.Options{
        Driver: "postgres",
        DSN:    "postgres://user:pass@localhost/db?sslmode=disable",
        Logger: logger,
    })
    if err != nil {
        log.Fatal("Failed to create storage client:", err)
    }
    defer client.Close()

    // Add health check endpoint
    mux := http.NewServeMux()
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        if err := client.Health(r.Context()); err != nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            w.Write([]byte("Database unhealthy"))
            return
        }
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Healthy"))
    })

    // Configure runtime
    opts := runtimex.Options{
        Logger: logger,
        HTTP: &runtimex.HTTPOptions{
            Addr: ":8080",
            H2C:  true,
            Mux:  mux,
        },
        Health:  &runtimex.Endpoint{Addr: ":8081"},
        Metrics: &runtimex.Endpoint{Addr: ":9091"},
    }

    // Run service
    runtimex.Run(ctx, nil, opts)
}
```

## 🧪 Testing

Run tests:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

## 📈 Test Coverage

| Component | Coverage |
|-----------|----------|
| StoreX | Good |

## 🔍 Troubleshooting

### Common Issues

1. **Database Connection Failed**
   ```bash
   # Check if database is running
   pg_isready -h localhost -p 5432
   
   # Check connection string
   echo $DB_DSN
   ```

2. **Migration Failed**
   ```go
   // Check if models are properly defined
   type User struct {
       ID    uint   `gorm:"primaryKey"`
       Email string `gorm:"uniqueIndex;not null"`
   }
   ```

3. **Performance Issues**
   ```go
   // Optimize connection pool
   client, err := storex.NewClient(ctx, storex.Options{
       Driver:      "postgres",
       DSN:         dsn,
       MaxIdle:     10,
       MaxOpen:     100,
       MaxLifetime: time.Hour,
   })
   ```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Create a Pull Request

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](../../LICENSE) file for details.

---

<div align="center">

**Built with ❤️ by EggyByte Technology**

</div>
