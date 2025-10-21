# 🗄️ StoreX Package

The `storex` package provides database integration for the EggyByte framework.

## Overview

This package offers a comprehensive database abstraction layer with GORM integration, connection pooling, and transaction management. It's designed to be production-ready with proper error handling and observability.

## Features

- **GORM integration** - Full GORM support with auto-migration
- **Connection pooling** - Configurable connection pool management
- **Transaction support** - ACID transaction management
- **Health checks** - Database health monitoring
- **Observability** - Metrics and tracing integration
- **Production ready** - Optimized for production environments

## Quick Start

```go
import "github.com/eggybyte-technology/egg/storex"

func main() {
    // Create database store
    store, err := storex.NewStore(ctx, logger, storex.Options{
        Driver: "mysql",
        DSN:    "user:password@tcp(localhost:3306)/db?charset=utf8mb4&parseTime=True&loc=Local",
        MaxIdle: 10,
        MaxOpen: 100,
        MaxLifetime: time.Hour,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Auto-migrate models
    if err := store.AutoMigrate(&User{}, &Order{}); err != nil {
        log.Fatal(err)
    }
    
    // Use store
    userRepo := NewUserRepository(store)
    user, err := userRepo.GetUser(ctx, "user-123")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("User: %s\n", user.Name)
}
```

## API Reference

### Types

#### Store

```go
type Store struct {
    db     *gorm.DB
    logger log.Logger
    config Options
}

// Close closes the database connection
func (s *Store) Close() error

// AutoMigrate runs auto migration for given models
func (s *Store) AutoMigrate(dst ...interface{}) error

// Health checks database health
func (s *Store) Health(ctx context.Context) error

// Transaction runs a function within a database transaction
func (s *Store) Transaction(ctx context.Context, fn func(*gorm.DB) error) error

// GetDB returns the underlying GORM DB instance
func (s *Store) GetDB() *gorm.DB
```

#### Options

```go
type Options struct {
    Driver     string        // Database driver (mysql, postgres, sqlite)
    DSN        string        // Database connection string
    MaxIdle    int           // Maximum number of idle connections
    MaxOpen    int           // Maximum number of open connections
    MaxLifetime time.Duration // Maximum connection lifetime
    LogLevel   string        // GORM log level (silent, error, warn, info)
    SlowThreshold time.Duration // Slow query threshold
}
```

### Functions

```go
// NewStore creates a new database store
func NewStore(ctx context.Context, logger log.Logger, opts Options) (*Store, error)

// NewStoreFromConfig creates a new database store from configuration
func NewStoreFromConfig(ctx context.Context, logger log.Logger, cfg DatabaseConfig) (*Store, error)
```

## Usage Examples

### Basic Store Setup

```go
func main() {
    // Create database store
    store, err := storex.NewStore(ctx, logger, storex.Options{
        Driver: "mysql",
        DSN:    "user:password@tcp(localhost:3306)/db?charset=utf8mb4&parseTime=True&loc=Local",
        MaxIdle: 10,
        MaxOpen: 100,
        MaxLifetime: time.Hour,
        LogLevel: "warn",
        SlowThreshold: 200 * time.Millisecond,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Auto-migrate models
    if err := store.AutoMigrate(&User{}, &Order{}); err != nil {
        log.Fatal(err)
    }
    
    // Use store
    useStore(store)
}
```

### Repository Pattern

```go
type UserRepository struct {
    store *storex.Store
}

func NewUserRepository(store *storex.Store) *UserRepository {
    return &UserRepository{store: store}
}

func (r *UserRepository) GetUser(ctx context.Context, userID string) (*User, error) {
    var user User
    err := r.store.GetDB().WithContext(ctx).Where("id = ?", userID).First(&user).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errors.New("NOT_FOUND", "user not found")
        }
        return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to get user")
    }
    
    return &user, nil
}

func (r *UserRepository) CreateUser(ctx context.Context, user *User) error {
    err := r.store.GetDB().WithContext(ctx).Create(user).Error
    if err != nil {
        return errors.Wrap(err, "DATABASE_ERROR", "failed to create user")
    }
    
    return nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, user *User) error {
    err := r.store.GetDB().WithContext(ctx).Save(user).Error
    if err != nil {
        return errors.Wrap(err, "DATABASE_ERROR", "failed to update user")
    }
    
    return nil
}

func (r *UserRepository) DeleteUser(ctx context.Context, userID string) error {
    err := r.store.GetDB().WithContext(ctx).Where("id = ?", userID).Delete(&User{}).Error
    if err != nil {
        return errors.Wrap(err, "DATABASE_ERROR", "failed to delete user")
    }
    
    return nil
}

func (r *UserRepository) ListUsers(ctx context.Context, limit, offset int) ([]*User, error) {
    var users []*User
    err := r.store.GetDB().WithContext(ctx).
        Limit(limit).
        Offset(offset).
        Find(&users).Error
    if err != nil {
        return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to list users")
    }
    
    return users, nil
}
```

### Transaction Management

```go
func (r *UserRepository) CreateUserWithProfile(ctx context.Context, user *User, profile *Profile) error {
    return r.store.Transaction(ctx, func(tx *gorm.DB) error {
        // Create user
        if err := tx.WithContext(ctx).Create(user).Error; err != nil {
            return errors.Wrap(err, "DATABASE_ERROR", "failed to create user")
        }
        
        // Create profile
        profile.UserID = user.ID
        if err := tx.WithContext(ctx).Create(profile).Error; err != nil {
            return errors.Wrap(err, "DATABASE_ERROR", "failed to create profile")
        }
        
        return nil
    })
}

func (r *UserRepository) TransferPoints(ctx context.Context, fromUserID, toUserID string, points int) error {
    return r.store.Transaction(ctx, func(tx *gorm.DB) error {
        // Deduct points from source user
        if err := tx.WithContext(ctx).
            Model(&User{}).
            Where("id = ?", fromUserID).
            Update("points", gorm.Expr("points - ?", points)).Error; err != nil {
            return errors.Wrap(err, "DATABASE_ERROR", "failed to deduct points")
        }
        
        // Add points to destination user
        if err := tx.WithContext(ctx).
            Model(&User{}).
            Where("id = ?", toUserID).
            Update("points", gorm.Expr("points + ?", points)).Error; err != nil {
            return errors.Wrap(err, "DATABASE_ERROR", "failed to add points")
        }
        
        return nil
    })
}
```

### Health Checks

```go
func main() {
    // Create database store
    store, err := storex.NewStore(ctx, logger, storex.Options{
        Driver: "mysql",
        DSN:    "user:password@tcp(localhost:3306)/db?charset=utf8mb4&parseTime=True&loc=Local",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Create health check handler
    mux := http.NewServeMux()
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        if err := store.Health(r.Context()); err != nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            w.Write([]byte("Database unhealthy"))
            return
        }
        
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })
    
    // Start server
    server := &http.Server{
        Addr:    ":8081",
        Handler: mux,
    }
    
    go func() {
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()
    
    // Graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        log.Fatal(err)
    }
}
```

### Configuration Integration

```go
type AppConfig struct {
    configx.BaseConfig
    
    // Database configuration
    Database DatabaseConfig
}

type DatabaseConfig struct {
    Driver       string        `env:"DB_DRIVER" default:"mysql"`
    DSN          string        `env:"DB_DSN" default:"user:password@tcp(localhost:3306)/db"`
    MaxIdle      int           `env:"DB_MAX_IDLE" default:"10"`
    MaxOpen      int           `env:"DB_MAX_OPEN" default:"100"`
    MaxLifetime  time.Duration `env:"DB_MAX_LIFETIME" default:"1h"`
    LogLevel     string        `env:"DB_LOG_LEVEL" default:"warn"`
    SlowThreshold time.Duration `env:"DB_SLOW_THRESHOLD" default:"200ms"`
}

func main() {
    // Load configuration
    var cfg AppConfig
    if err := configManager.Bind(&cfg); err != nil {
        log.Fatal(err)
    }
    
    // Create database store from configuration
    store, err := storex.NewStoreFromConfig(ctx, logger, cfg.Database)
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Auto-migrate models
    if err := store.AutoMigrate(&User{}, &Order{}); err != nil {
        log.Fatal(err)
    }
    
    // Use store
    useStore(store)
}
```

### Service Integration

```go
type UserService struct {
    logger log.Logger
    repo   *UserRepository
}

func NewUserService(logger log.Logger, store *storex.Store) *UserService {
    return &UserService{
        logger: logger,
        repo:   NewUserRepository(store),
    }
}

func (s *UserService) GetUser(ctx context.Context, req *connect.Request[GetUserRequest]) (*connect.Response[GetUserResponse], error) {
    // Get user from repository
    user, err := s.repo.GetUser(ctx, req.Msg.UserId)
    if err != nil {
        if errors.Is(err, "NOT_FOUND") {
            return nil, connect.NewError(connect.CodeNotFound, err)
        }
        return nil, connect.NewError(connect.CodeInternal, err)
    }
    
    return connect.NewResponse(&GetUserResponse{User: user}), nil
}

func (s *UserService) CreateUser(ctx context.Context, req *connect.Request[CreateUserRequest]) (*connect.Response[CreateUserResponse], error) {
    // Validate request
    if req.Msg.User == nil {
        return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("VALIDATION_ERROR", "user is required"))
    }
    
    // Create user
    user := &User{
        Name:  req.Msg.User.Name,
        Email: req.Msg.User.Email,
    }
    
    if err := s.repo.CreateUser(ctx, user); err != nil {
        return nil, connect.NewError(connect.CodeInternal, err)
    }
    
    return connect.NewResponse(&CreateUserResponse{User: user}), nil
}
```

## Model Definitions

### User Model

```go
type User struct {
    ID        uint      `json:"id" gorm:"primaryKey"`
    Name      string    `json:"name" gorm:"size:255;not null"`
    Email     string    `json:"email" gorm:"size:255;uniqueIndex;not null"`
    Points    int       `json:"points" gorm:"default:0"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (u *User) TableName() string {
    return "users"
}
```

### Order Model

```go
type Order struct {
    ID        uint      `json:"id" gorm:"primaryKey"`
    UserID    uint      `json:"user_id" gorm:"not null"`
    User      User      `json:"user" gorm:"foreignKey:UserID"`
    Amount    int       `json:"amount" gorm:"not null"`
    Status    string    `json:"status" gorm:"size:50;default:'pending'"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (o *Order) TableName() string {
    return "orders"
}
```

## Testing

```go
func TestStore(t *testing.T) {
    // Create test store
    store, err := storex.NewStore(context.Background(), &TestLogger{}, storex.Options{
        Driver: "sqlite",
        DSN:    ":memory:",
    })
    assert.NoError(t, err)
    defer store.Close()
    
    // Auto-migrate models
    err = store.AutoMigrate(&User{})
    assert.NoError(t, err)
    
    // Test health check
    err = store.Health(context.Background())
    assert.NoError(t, err)
    
    // Test repository
    repo := NewUserRepository(store)
    
    // Create user
    user := &User{
        Name:  "Test User",
        Email: "test@example.com",
    }
    
    err = repo.CreateUser(context.Background(), user)
    assert.NoError(t, err)
    assert.NotZero(t, user.ID)
    
    // Get user
    retrievedUser, err := repo.GetUser(context.Background(), fmt.Sprintf("%d", user.ID))
    assert.NoError(t, err)
    assert.Equal(t, user.Name, retrievedUser.Name)
    assert.Equal(t, user.Email, retrievedUser.Email)
}

func TestTransaction(t *testing.T) {
    // Create test store
    store, err := storex.NewStore(context.Background(), &TestLogger{}, storex.Options{
        Driver: "sqlite",
        DSN:    ":memory:",
    })
    assert.NoError(t, err)
    defer store.Close()
    
    // Auto-migrate models
    err = store.AutoMigrate(&User{})
    assert.NoError(t, err)
    
    // Test transaction
    err = store.Transaction(context.Background(), func(tx *gorm.DB) error {
        user := &User{
            Name:  "Test User",
            Email: "test@example.com",
        }
        
        if err := tx.Create(user).Error; err != nil {
            return err
        }
        
        // Simulate error to test rollback
        return errors.New("test error")
    })
    
    assert.Error(t, err)
    
    // Verify user was not created (transaction rolled back)
    var count int64
    store.GetDB().Model(&User{}).Count(&count)
    assert.Equal(t, int64(0), count)
}

type TestLogger struct{}

func (l *TestLogger) With(kv ...any) log.Logger { return l }
func (l *TestLogger) Debug(msg string, kv ...any) {}
func (l *TestLogger) Info(msg string, kv ...any) {}
func (l *TestLogger) Warn(msg string, kv ...any) {}
func (l *TestLogger) Error(err error, msg string, kv ...any) {}
```

## Best Practices

### 1. Use Repository Pattern

```go
type UserRepository struct {
    store *storex.Store
}

func (r *UserRepository) GetUser(ctx context.Context, userID string) (*User, error) {
    var user User
    err := r.store.GetDB().WithContext(ctx).Where("id = ?", userID).First(&user).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errors.New("NOT_FOUND", "user not found")
        }
        return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to get user")
    }
    
    return &user, nil
}
```

### 2. Use Transactions for Complex Operations

```go
func (r *UserRepository) TransferPoints(ctx context.Context, fromUserID, toUserID string, points int) error {
    return r.store.Transaction(ctx, func(tx *gorm.DB) error {
        // Deduct points from source user
        if err := tx.WithContext(ctx).
            Model(&User{}).
            Where("id = ?", fromUserID).
            Update("points", gorm.Expr("points - ?", points)).Error; err != nil {
            return errors.Wrap(err, "DATABASE_ERROR", "failed to deduct points")
        }
        
        // Add points to destination user
        if err := tx.WithContext(ctx).
            Model(&User{}).
            Where("id = ?", toUserID).
            Update("points", gorm.Expr("points + ?", points)).Error; err != nil {
            return errors.Wrap(err, "DATABASE_ERROR", "failed to add points")
        }
        
        return nil
    })
}
```

### 3. Handle Errors Properly

```go
func (r *UserRepository) GetUser(ctx context.Context, userID string) (*User, error) {
    var user User
    err := r.store.GetDB().WithContext(ctx).Where("id = ?", userID).First(&user).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errors.New("NOT_FOUND", "user not found")
        }
        return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to get user")
    }
    
    return &user, nil
}
```

### 4. Use Context for Cancellation

```go
func (r *UserRepository) GetUser(ctx context.Context, userID string) (*User, error) {
    var user User
    err := r.store.GetDB().WithContext(ctx).Where("id = ?", userID).First(&user).Error
    if err != nil {
        return nil, err
    }
    
    return &user, nil
}
```

## Thread Safety

All functions in this package are safe for concurrent use. The GORM database connection is designed to handle concurrent access safely.

## Dependencies

- **Go 1.21+** required
- **GORM** - ORM library
- **Database drivers** - MySQL, PostgreSQL, SQLite
- **Standard library** - Core functionality

## Version Compatibility

- **Go 1.21+** required
- **API Stability**: Evolving (L3 module)
- **Breaking Changes**: Possible in minor versions

## Contributing

Contributions are welcome! Please see the main project [Contributing Guide](../CONTRIBUTING.md) for details.

## License

This package is part of the EggyByte framework and is licensed under the MIT License.