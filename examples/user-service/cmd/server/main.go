// Package main provides the main entry point for the user service.
//
// Overview:
//   - Responsibility: Service initialization using servicex library
//   - Key Types: Main function with minimal service setup
//   - Concurrency Model: Graceful shutdown handled by servicex
//   - Error Semantics: Startup errors are logged and cause exit
//   - Performance Notes: Optimized for fast startup and graceful shutdown
//
// Usage:
//
//	go run cmd/server/main.go
//	./user-service
package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/eggybyte-technology/egg/core/errors"
	"github.com/eggybyte-technology/egg/core/log"
	userv1connect "github.com/eggybyte-technology/egg/examples/user-service/gen/go/user/v1/userv1connect"
	"github.com/eggybyte-technology/egg/examples/user-service/internal/config"
	"github.com/eggybyte-technology/egg/examples/user-service/internal/handler"
	"github.com/eggybyte-technology/egg/examples/user-service/internal/model"
	"github.com/eggybyte-technology/egg/examples/user-service/internal/repository"
	"github.com/eggybyte-technology/egg/examples/user-service/internal/service"
	"github.com/eggybyte-technology/egg/servicex"
	"github.com/google/uuid"
)

// SimpleLogger implements the log.Logger interface for basic logging.
type SimpleLogger struct{}

func (l *SimpleLogger) With(kv ...any) log.Logger   { return l }
func (l *SimpleLogger) Debug(msg string, kv ...any) { fmt.Printf("[DEBUG] %s %v\n", msg, kv) }
func (l *SimpleLogger) Info(msg string, kv ...any)  { fmt.Printf("[INFO] %s %v\n", msg, kv) }
func (l *SimpleLogger) Warn(msg string, kv ...any)  { fmt.Printf("[WARN] %s %v\n", msg, kv) }
func (l *SimpleLogger) Error(err error, msg string, kv ...any) {
	if err != nil {
		fmt.Printf("[ERROR] %s: %v %v\n", msg, err, kv)
	} else {
		fmt.Printf("[ERROR] %s %v\n", msg, kv)
	}
}

func main() {
	// Initialize logger
	logger := &SimpleLogger{}
	logger.Info("Starting user service")

	// Create context
	ctx := context.Background()

	// Initialize configuration
	var cfg config.AppConfig

	// Run the service using servicex
	err := servicex.Run(ctx, servicex.Options{
		ServiceName: "user-service",
		Config:      &cfg,
		// Database: &servicex.DatabaseConfig{
		// 	Driver:      "mysql",
		// 	DSN:         "", // Will be set from config
		// 	MaxIdle:     10,
		// 	MaxOpen:     100,
		// 	MaxLifetime: 1 * time.Hour,
		// },
		// Migrate: func(db *gorm.DB) error {
		// 	return db.AutoMigrate(&model.User{})
		// },
		Register: func(app *servicex.App) error {
			// Initialize repository
			var userRepo repository.UserRepository
			if db := app.DB(); db != nil {
				userRepo = repository.NewUserRepository(db)
				logger.Info("Repository initialized successfully")
			} else {
				// Use in-memory repository for demo
				userRepo = &mockUserRepository{}
				logger.Info("Using mock repository (no database)")
			}

			// Initialize service and handler
			userService := service.NewUserService(userRepo, logger)
			userHandler := handler.NewUserHandler(userService, logger)

			// Create Connect handler with interceptors
			path, connectHandler := userv1connect.NewUserServiceHandler(
				userHandler,
				connect.WithInterceptors(app.Interceptors()...),
			)

			// Register handler
			app.Mux().Handle(path, connectHandler)
			logger.Info("Registered Connect handler", log.Str("path", path))

			logger.Info("User service initialized successfully")
			return nil
		},
		EnableTracing:     true,
		EnableHealthCheck: true,
		EnableMetrics:     true,
		EnableDebugLogs:   false,
		SlowRequestMillis: 1000,
		PayloadAccounting: true,
		ShutdownTimeout:   15 * time.Second,
		Logger:            logger,
	})
	if err != nil {
		logger.Error(err, "Service failed")
		return
	}

	logger.Info("User service stopped gracefully")
}

// mockUserRepository is a simple in-memory implementation for demo purposes
type mockUserRepository struct {
	users map[string]*model.User
	mutex sync.RWMutex
}

func (m *mockUserRepository) Create(ctx context.Context, user *model.User) (*model.User, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.users == nil {
		m.users = make(map[string]*model.User)
	}

	// Check if email already exists
	for _, existingUser := range m.users {
		if existingUser.Email == user.Email {
			return nil, errors.Wrap(errors.CodeAlreadyExists, "email check", model.ErrEmailExists)
		}
	}

	// Generate ID and timestamps
	user.ID = uuid.New().String()
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	m.users[user.ID] = user
	return user, nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.users == nil {
		return nil, errors.Wrap(errors.CodeNotFound, "get user", errors.New(errors.CodeNotFound, "user not found"))
	}

	user, exists := m.users[id]
	if !exists {
		return nil, errors.Wrap(errors.CodeNotFound, "get user", errors.New(errors.CodeNotFound, "user not found"))
	}

	return user, nil
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.users == nil {
		return nil, errors.Wrap(errors.CodeNotFound, "get user", errors.New(errors.CodeNotFound, "user not found"))
	}

	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}

	return nil, errors.Wrap(errors.CodeNotFound, "get user", errors.New(errors.CodeNotFound, "user not found"))
}

func (m *mockUserRepository) Update(ctx context.Context, user *model.User) (*model.User, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.users == nil {
		return nil, errors.Wrap(errors.CodeNotFound, "update user", errors.New(errors.CodeNotFound, "user not found"))
	}

	existingUser, exists := m.users[user.ID]
	if !exists {
		return nil, errors.Wrap(errors.CodeNotFound, "update user", errors.New(errors.CodeNotFound, "user not found"))
	}

	// Update fields
	existingUser.Name = user.Name
	existingUser.Email = user.Email
	existingUser.UpdatedAt = time.Now()

	m.users[user.ID] = existingUser
	return existingUser, nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.users == nil {
		return errors.Wrap(errors.CodeNotFound, "delete user", errors.New(errors.CodeNotFound, "user not found"))
	}

	if _, exists := m.users[id]; !exists {
		return errors.Wrap(errors.CodeNotFound, "delete user", errors.New(errors.CodeNotFound, "user not found"))
	}

	delete(m.users, id)
	return nil
}

func (m *mockUserRepository) List(ctx context.Context, page, pageSize int) ([]*model.User, int64, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.users == nil {
		return []*model.User{}, 0, nil
	}

	users := make([]*model.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}

	total := int64(len(users))
	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= int(total) {
		return []*model.User{}, total, nil
	}

	if end > int(total) {
		end = int(total)
	}

	return users[start:end], total, nil
}
