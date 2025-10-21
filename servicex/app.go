// Package servicex provides a unified microservice initialization framework.
package servicex

import (
	"net/http"

	"connectrpc.com/connect"
	"github.com/eggybyte-technology/egg/core/log"
	"github.com/eggybyte-technology/egg/obsx"
	"gorm.io/gorm"
)

// App provides access to service components during registration.
// This is the only interface exposed to service registration functions.
type App struct {
	mux          *http.ServeMux
	logger       log.Logger
	interceptors []connect.Interceptor
	db           *gorm.DB
	otel         *obsx.Provider
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

// DB returns the database connection.
// Returns nil if database is not configured.
func (a *App) DB() *gorm.DB {
	return a.db
}

// OtelProvider returns the OpenTelemetry provider.
// Returns nil if tracing is disabled.
func (a *App) OtelProvider() *obsx.Provider {
	return a.otel
}
