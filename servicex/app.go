// Package servicex provides a unified microservice initialization framework.
package servicex

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/eggybyte-technology/egg/core/log"
	"github.com/eggybyte-technology/egg/obsx"
	"gorm.io/gorm"
)

// App provides access to service components during registration.
// This is the only interface exposed to service registration functions.
type App struct {
	mux           *http.ServeMux
	logger        log.Logger
	interceptors  []connect.Interceptor
	otel          *obsx.Provider
	container     *container
	shutdownHooks []func(context.Context) error
	db            *gorm.DB // Database connection (optional)
}

// Mux returns the HTTP mux for handler registration.
func (a *App) Mux() *http.ServeMux {
	return a.mux
}

// Logger returns the logger instance.
func (a *App) Logger() log.Logger {
	return a.logger
}

// Interceptors returns the configured Connect interceptors.
func (a *App) Interceptors() []connect.Interceptor {
	return a.interceptors
}

// OtelProvider returns the OpenTelemetry provider.
// Returns nil if tracing is disabled.
func (a *App) OtelProvider() *obsx.Provider {
	return a.otel
}

// Provide registers a constructor in the DI container.
func (a *App) Provide(constructor any) error {
	return a.container.Provide(constructor)
}

// Resolve resolves a dependency from the DI container.
func (a *App) Resolve(target any) error {
	return a.container.Resolve(target)
}

// AddShutdownHook registers a shutdown hook.
// Hooks are called in LIFO order during graceful shutdown.
func (a *App) AddShutdownHook(hook func(context.Context) error) {
	a.shutdownHooks = append(a.shutdownHooks, hook)
}

// DB returns the GORM database instance.
// Returns nil if no database was configured via WithDatabase option.
//
// Returns:
//   - *gorm.DB: Database instance, or nil if not configured.
//   - error: Error if database was configured but initialization failed.
//
// Concurrency:
//   - The returned *gorm.DB is safe for concurrent use.
//
// Example:
//
//	db := app.DB()
//	if db == nil {
//	    return errors.New("database not configured")
//	}
//	// Use db for queries...
func (a *App) DB() *gorm.DB {
	return a.db
}

// MustDB returns the GORM database instance or panics if not configured.
// This is useful in registration functions where database is required.
//
// Returns:
//   - *gorm.DB: Database instance.
//
// Panics:
//   - If database was not configured via WithDatabase option.
//
// Example:
//
//	db := app.MustDB()
//	repo := repository.NewUserRepository(db)
func (a *App) MustDB() *gorm.DB {
	if a.db == nil {
		panic(fmt.Errorf("database not configured; use WithDatabase option"))
	}
	return a.db
}
