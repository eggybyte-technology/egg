// Package servicex provides a unified microservice initialization framework.
package servicex

import (
	"time"

	"github.com/eggybyte-technology/egg/core/log"
	"gorm.io/gorm"
)

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
	Driver      string        `env:"DB_DRIVER" default:"mysql"`
	DSN         string        `env:"DB_DSN" default:""`
	MaxIdle     int           `env:"DB_MAX_IDLE" default:"10"`
	MaxOpen     int           `env:"DB_MAX_OPEN" default:"100"`
	MaxLifetime time.Duration `env:"DB_MAX_LIFETIME" default:"1h"`
}

// DatabaseMigrator defines a function that performs database migrations.
type DatabaseMigrator func(db *gorm.DB) error

// ServiceRegistrar defines a function that registers services with the application.
type ServiceRegistrar func(app *App) error

// Options holds configuration for service initialization.
type Options struct {
	// Service identification
	ServiceName    string `env:"SERVICE_NAME" default:"app"`
	ServiceVersion string `env:"SERVICE_VERSION" default:"0.0.0"`

	// Configuration
	Config any // Configuration struct that embeds configx.BaseConfig

	// Database (optional)
	Database *DatabaseConfig  // Database configuration
	Migrate  DatabaseMigrator // Database migration function

	// Service registration
	Register ServiceRegistrar // Service registration function

	// Observability
	EnableTracing bool `env:"ENABLE_TRACING" default:"true"`

	// Feature flags
	EnableHealthCheck bool `env:"ENABLE_HEALTH_CHECK" default:"true"`
	EnableMetrics     bool `env:"ENABLE_METRICS" default:"true"`
	EnableDebugLogs   bool `env:"ENABLE_DEBUG_LOGS" default:"false"`

	// Connect interceptor options
	SlowRequestMillis int64 `env:"SLOW_REQUEST_MILLIS" default:"1000"`
	PayloadAccounting bool  `env:"PAYLOAD_ACCOUNTING" default:"true"`

	// Shutdown timeout
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" default:"15s"`

	// Logger (optional, will create default if nil)
	Logger log.Logger
}

// validate validates the options and sets defaults.
func (o *Options) validate() error {
	if o.ServiceName == "" {
		o.ServiceName = "app"
	}
	if o.ServiceVersion == "" {
		o.ServiceVersion = "0.0.0"
	}
	if o.ShutdownTimeout == 0 {
		o.ShutdownTimeout = 15 * time.Second
	}
	if o.SlowRequestMillis == 0 {
		o.SlowRequestMillis = 1000
	}
	return nil
}
