// Package main provides the main entry point for the user service.
//
// Overview:
//
//	This example demonstrates a production-ready CRUD service using the egg
//	framework. It showcases proper layering (handler/service/repository),
//	database integration with GORM, service-to-service communication with clientx,
//	and comprehensive error handling.
//
// Key Features:
//   - Full CRUD operations: Create, Read, Update, Delete, List (with pagination)
//   - Database integration: MySQL/PostgreSQL with auto-migration and connection pooling
//   - Layered architecture: Clear separation of concerns (handler/service/repository/model)
//   - Connect RPC: Modern HTTP/2-based RPC with streaming support
//   - Service-to-service communication: clientx with retry, circuit breaker, and internal token
//   - Structured logging: All operations logged with context
//   - Comprehensive validation: Email format, required fields, uniqueness constraints
//   - Automatic health checks: Database connectivity verification
//   - Automatic metrics: Request counts, latencies, error rates, database stats
//   - Internal token authentication: Method-level protection for admin operations
//
// Architecture:
//
//   - Handler layer: Connect RPC protocol implementation (thin adapter)
//   - Service layer: Business logic and domain validation
//   - Repository layer: Database operations and persistence
//   - Model layer: Domain entities and validation rules
//   - Client layer: Service-to-service communication using clientx
//
// This demonstrates the egg framework's recommended pattern for production
// services requiring proper layering, testability, maintainability, and
// inter-service communication.
//
// Usage:
//
//	Database is required. Configure via environment variable:
//	  DB_DSN="user:pass@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local" ./user-service
//
//	Additional configuration via environment:
//	  SERVICE_NAME=user-service HTTP_PORT=8080 HEALTH_PORT=8081 ./user-service
//
// Endpoints:
//   - HTTP: 8080 (configurable via HTTP_PORT)
//   - Health: 8081 (configurable via HEALTH_PORT)
//   - Metrics: 9091 (configurable via METRICS_PORT)
//
// Database:
//
//	The service auto-migrates the schema on startup. Database connection is
//	mandatory - the service will fail to start if DB_DSN is not configured.
//	Supported databases: MySQL, PostgreSQL, SQLite (via GORM).
//
// Dependencies:
//   - servicex: unified service initialization (L4)
//   - configx: configuration management (L2)
//   - logx: structured logging (L1)
//   - connectx: Connect interceptor stack (L3)
//   - clientx: HTTP client factory for service-to-service communication (L3)
//   - storex: database connection and transaction management (附属)
//   - core/errors: error codes and wrapping (L0)
//   - GORM: database ORM
package main

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"go.eggybyte.com/egg/core/log"
	userv1connect "go.eggybyte.com/egg/examples/user-service/gen/go/user/v1/userv1connect"
	"go.eggybyte.com/egg/examples/user-service/internal/client"
	"go.eggybyte.com/egg/examples/user-service/internal/config"
	"go.eggybyte.com/egg/examples/user-service/internal/handler"
	"go.eggybyte.com/egg/examples/user-service/internal/model"
	"go.eggybyte.com/egg/examples/user-service/internal/repository"
	"go.eggybyte.com/egg/examples/user-service/internal/service"
	"go.eggybyte.com/egg/servicex"
	"gorm.io/gorm"
)

func main() {
	// Create context for the service
	ctx := context.Background()

	// Initialize configuration - will be populated by servicex
	cfg := &config.AppConfig{}

	// Run the service using servicex with database integration
	// servicex automatically creates a logger with LOG_LEVEL from environment
	// WithAppConfig automatically detects BaseConfig and uses Database configuration
	err := servicex.Run(ctx,
		servicex.WithService("user-service", "0.1.0"),
		servicex.WithAppConfig(cfg), // Auto-detects database config from BaseConfig
		servicex.WithAutoMigrate(&model.User{}),
		servicex.WithMetricsConfig(true, true, true, false), // Enable runtime, process, and DB metrics
		servicex.WithRegister(registerServices),
	)
	if err != nil {
		// Logger is not available here if service fails, use panic
		panic(fmt.Sprintf("service failed to start: %v", err))
	}
}

// registerServices registers all service handlers with the application.
//
// This function is called by servicex during initialization. It demonstrates
// production-ready service registration with mandatory database dependency
// using the simplified RegisterServices helper.
//
// Parameters:
//   - app: servicex application instance providing database, logger, mux, and interceptors
//
// Returns:
//   - error: nil on success; error if database is not configured or registration fails
//
// Behavior:
//   - Uses RegisterServices to automatically register common constructors (logger, database)
//   - Requires database to be configured via DB_DSN environment variable
//   - Fails fast if database is not available (production best practice)
//   - Registers Connect handler with full interceptor stack
//   - Initializes service-to-service clients using clientx
//   - Logs registration progress for observability
//
// Concurrency:
//
//	Called once during service startup, not safe for concurrent use.
func registerServices(app *servicex.App) error {
	// Ensure database is configured (production requirement)
	app.MustDB() // Verifies database is configured
	app.Logger().Info("initializing user service with database-backed repository")

	// Register optional clients using the batch registration system
	clients := servicex.RegisterOptionalClients(app.Logger(), app.InternalToken(), servicex.ClientRegistryConfig{
		ConfigGetter: func() any { return app.Config() },
		Clients: map[string]servicex.ClientConfig{
			"greet": {
				URLKey:       "GreetServiceURL",
				CreateClient: func(url, token string) any { return client.NewGreetClient(url, token) },
				ClientName:   "greet service",
			},
		},
	})
	greetClient, _ := clients["greet"].(*client.GreetClient)

	// Register all constructors using RegisterServices (combines common + custom constructors)
	if err := servicex.RegisterServices(app, map[string]any{
		"repository": func(db *gorm.DB) repository.UserRepository {
			return repository.NewUserRepository(db)
		},
		"service": func(repo repository.UserRepository, logger log.Logger) service.UserService {
			return service.NewUserService(repo, logger, greetClient)
		},
		"handler": func(svc service.UserService, logger log.Logger) *handler.UserHandler {
			return handler.NewUserHandler(svc, logger, app.InternalToken())
		},
	}); err != nil {
		return err
	}

	// Resolve and register Connect handler
	var userHandler *handler.UserHandler
	if err := app.Resolve(&userHandler); err != nil {
		return fmt.Errorf("failed to resolve user handler: %w", err)
	}

	path, connectHandler := userv1connect.NewUserServiceHandler(
		userHandler,
		connect.WithInterceptors(app.Interceptors()...),
	)
	app.Mux().Handle(path, connectHandler)
	app.Logger().Info("user service initialized successfully", "path", path)

	return nil
}
