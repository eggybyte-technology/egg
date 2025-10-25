// Package main provides the main entry point for the user service.
//
// Overview:
//
//	This example demonstrates a production-ready CRUD service using the egg
//	framework. It showcases proper layering (handler/service/repository),
//	database integration with GORM, and comprehensive error handling.
//
// Key Features:
//   - Full CRUD operations: Create, Read, Update, Delete, List (with pagination)
//   - Database integration: MySQL with auto-migration and connection pooling
//   - Layered architecture: Clear separation of concerns
//   - Connect RPC: Modern HTTP/2-based RPC with streaming support
//   - Structured logging: All operations logged with context
//   - Comprehensive validation: Email format, required fields, uniqueness
//   - Mock repository: Fallback in-memory implementation for testing
//   - Automatic health checks: Database connectivity verification
//   - Automatic metrics: Request counts, latencies, error rates
//
// Architecture:
//
//   - Handler layer: Connect RPC protocol implementation (thin adapter)
//
//   - Service layer: Business logic and domain validation
//
//   - Repository layer: Database operations and persistence
//
//   - Model layer: Domain entities and validation rules
//
//     This demonstrates the egg framework's recommended pattern for complex
//     services that need proper layering and testability.
//
// Usage:
//
//	Run with database:
//	  DB_DSN="user:pass@tcp(localhost:3306)/dbname" ./user-service
//
//	Run without database (mock mode):
//	  ./user-service
//
//	Configure via environment:
//	  SERVICE_NAME=user-service HTTP_PORT=8080 ./user-service
//
// Endpoints:
//   - HTTP: 8080 (configurable via HTTP_PORT)
//   - Health: 8081 (configurable via HEALTH_PORT)
//   - Metrics: 9091 (configurable via METRICS_PORT)
//
// Database:
//
//	The service auto-migrates the schema on startup. If DB_DSN is not provided,
//	it falls back to an in-memory mock repository for demonstration purposes.
//
// Dependencies:
//   - servicex: unified service initialization (L4)
//   - configx: configuration management (L2)
//   - logx: structured logging (L1)
//   - connectx: Connect interceptor stack (L3)
//   - core/errors: error codes and wrapping (L0)
//   - GORM: database ORM
package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/eggybyte-technology/egg/core/errors"
	userv1connect "github.com/eggybyte-technology/egg/examples/user-service/gen/go/user/v1/userv1connect"
	"github.com/eggybyte-technology/egg/examples/user-service/internal/config"
	"github.com/eggybyte-technology/egg/examples/user-service/internal/handler"
	"github.com/eggybyte-technology/egg/examples/user-service/internal/model"
	"github.com/eggybyte-technology/egg/examples/user-service/internal/repository"
	"github.com/eggybyte-technology/egg/examples/user-service/internal/service"
	"github.com/eggybyte-technology/egg/logx"
	"github.com/eggybyte-technology/egg/servicex"
	"github.com/google/uuid"
)

func main() {
	// Create context for the service
	ctx := context.Background()

	// Create console logger for development (human-readable)
	logger := logx.New(
		logx.WithFormat(logx.FormatConsole),
		logx.WithLevel(slog.LevelInfo),
		logx.WithColor(true),
	)

	// Initialize configuration - will be populated by servicex
	cfg := &config.AppConfig{}

	// Run the service using servicex with database integration
	// WithAppConfig automatically detects BaseConfig and uses Database configuration
	err := servicex.Run(ctx,
		servicex.WithService("user-service", "0.1.0"),
		servicex.WithLogger(logger),
		servicex.WithAppConfig(cfg), // Auto-detects database config from BaseConfig
		servicex.WithAutoMigrate(&model.User{}),
		servicex.WithRegister(registerServices),
	)
	if err != nil {
		logger.Error(err, "service failed to start")
	}
}

// registerServices registers all service handlers
func registerServices(app *servicex.App) error {
	// Get repository (database-backed or mock)
	var userRepo repository.UserRepository
	if db := app.DB(); db != nil {
		app.Logger().Info("using database-backed repository")
		userRepo = repository.NewUserRepository(db)
	} else {
		app.Logger().Info("no database configured, using mock repository")
		userRepo = &mockUserRepository{}
	}

	// Initialize service and handler
	userService := service.NewUserService(userRepo, app.Logger())
	userHandler := handler.NewUserHandler(userService, app.Logger())

	// Register Connect handler
	path, connectHandler := userv1connect.NewUserServiceHandler(
		userHandler,
		connect.WithInterceptors(app.Interceptors()...),
	)

	app.Mux().Handle(path, connectHandler)
	app.Logger().Info("registered Connect handler", "path", path)
	app.Logger().Info("user service initialized successfully")
	return nil
}

// mockUserRepository is an in-memory implementation of UserRepository.
//
// This implementation provides a complete CRUD interface without requiring
// a database, useful for:
//   - Quick demonstrations and testing
//   - Development without external dependencies
//   - CI/CD environments
//
// Concurrency:
//
//	Safe for concurrent use via read-write mutex protection.
//
// Limitations:
//   - Data is lost on service restart (no persistence)
//   - Not suitable for production use
//   - No transaction support
//
// Note:
//
//	This is automatically used when DB_DSN is not configured. For production,
//	always configure a real database connection.
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
