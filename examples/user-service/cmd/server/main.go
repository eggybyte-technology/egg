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
	err := servicex.Run(ctx,
		servicex.WithService("user-service", "0.1.0"),
		servicex.WithLogger(logger),
		servicex.WithConfig(cfg),
		servicex.WithDatabase(servicex.FromBaseConfig(&cfg.Database)),
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
