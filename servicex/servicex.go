// Package servicex provides a unified microservice initialization framework with DI.
//
// Overview:
//   - Responsibility: Simplify microservice startup with integrated config, logging, DB, tracing
//   - Key Types: Options for configuration, App for service registration
//   - Concurrency Model: All components are safe for concurrent use
//   - Error Semantics: Initialization errors are returned immediately
//   - Performance Notes: Components are initialized lazily when needed
//
// Usage:
//
//	type AppConfig struct {
//	    configx.BaseConfig
//	    CustomField string `env:"CUSTOM_FIELD" default:"value"`
//	}
//
//	func register(app *servicex.App) error {
//	    handler := myhandler.New(app.Logger())
//	    connectx.Bind(app.Mux(), "/connect/user.v1.UserService/", handler)
//	    return nil
//	}
//
//	func main() {
//	    ctx := context.Background()
//	    cfg := &AppConfig{}
//	    err := servicex.Run(ctx,
//	        servicex.WithConfig(cfg),
//	        servicex.WithDatabase(&cfg.Database),
//	        servicex.WithRegister(register),
//	    )
//	    log.Fatal(err)
//	}
package servicex

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"go.eggybyte.com/egg/configx"
	"go.eggybyte.com/egg/core/log"
	"go.eggybyte.com/egg/obsx"
	"go.eggybyte.com/egg/servicex/internal"
	"gorm.io/gorm"
)

// App provides access to service components during registration.
type App struct {
	mux           *http.ServeMux
	logger        log.Logger
	interceptors  []connect.Interceptor
	otel          *obsx.Provider
	container     *internal.Container
	shutdownHooks []func(context.Context) error
	db            *gorm.DB
}

// Mux returns the HTTP mux for handler registration.
func (a *App) Mux() *http.ServeMux { return a.mux }

// Logger returns the logger instance.
func (a *App) Logger() log.Logger { return a.logger }

// Interceptors returns the configured Connect interceptors.
func (a *App) Interceptors() []connect.Interceptor { return a.interceptors }

// OtelProvider returns the OpenTelemetry provider (may be nil if disabled).
func (a *App) OtelProvider() *obsx.Provider { return a.otel }

// Provide registers a constructor in the DI container.
func (a *App) Provide(constructor any) error { return a.container.Provide(constructor) }

// Resolve resolves a dependency from the DI container.
func (a *App) Resolve(target any) error { return a.container.Resolve(target) }

// AddShutdownHook registers a shutdown hook (executed in LIFO order at shutdown).
func (a *App) AddShutdownHook(hook func(context.Context) error) {
	a.shutdownHooks = append(a.shutdownHooks, hook)
}

// DB returns the GORM database instance or nil if not configured.
func (a *App) DB() *gorm.DB { return a.db }

// MustDB returns the GORM database instance or panics if not configured.
func (a *App) MustDB() *gorm.DB {
	if a.db == nil {
		panic(fmt.Errorf("database not configured; use WithDatabase option"))
	}
	return a.db
}

// Option is a functional option for configuring the service.
type Option func(*internal.ServiceConfig)

// WithService sets the service name and version.
func WithService(name, version string) Option {
	return func(c *internal.ServiceConfig) {
		c.ServiceName = name
		c.ServiceVersion = version
	}
}

// WithConfig sets the configuration struct.
// If the config struct embeds configx.BaseConfig or has a Database field,
// it will automatically be used for database configuration.
func WithConfig(cfg any) Option {
	return func(c *internal.ServiceConfig) {
		c.Config = cfg

		// Auto-detect database configuration from BaseConfig or embedded Database
		if c.DBConfig == nil {
			// Try to extract BaseConfig first
			if baseCfg, ok := internal.ExtractBaseConfig(cfg); ok {
				dbCfg := &DatabaseConfig{
					Driver:          baseCfg.Database.Driver,
					DSN:             baseCfg.Database.DSN,
					MaxIdleConns:    baseCfg.Database.MaxIdle,
					MaxOpenConns:    baseCfg.Database.MaxOpen,
					ConnMaxLifetime: baseCfg.Database.MaxLifetime,
					PingTimeout:     5 * time.Second,
				}
				// Only set if DSN is provided
				if dbCfg.DSN != "" {
					c.DBConfig = &internal.DatabaseConfig{
						Driver:          dbCfg.Driver,
						DSN:             dbCfg.DSN,
						MaxIdleConns:    dbCfg.MaxIdleConns,
						MaxOpenConns:    dbCfg.MaxOpenConns,
						ConnMaxLifetime: dbCfg.ConnMaxLifetime,
						PingTimeout:     dbCfg.PingTimeout,
					}
				}
			}
		}
	}
}

// WithLogger sets the logger.
func WithLogger(logger log.Logger) Option {
	return func(c *internal.ServiceConfig) {
		c.Logger = logger
	}
}

// WithMetrics enables metrics.
func WithMetrics(enabled bool) Option {
	return func(c *internal.ServiceConfig) {
		c.EnableMetrics = enabled
	}
}

// WithMetricsConfig enables fine-grained metrics configuration.
// It automatically enables EnableMetrics if any metric type is enabled.
func WithMetricsConfig(runtime, process, db, client bool) Option {
	return func(c *internal.ServiceConfig) {
		c.MetricsConfig = &internal.MetricsConfig{
			EnableRuntime: runtime,
			EnableProcess: process,
			EnableDB:      db,
			EnableClient:  client,
		}
		// Auto-enable metrics if any metric type is enabled
		if runtime || process || db || client {
			c.EnableMetrics = true
		}
	}
}

// WithRegister sets the service registration function.
func WithRegister(fn func(*App) error) Option {
	return func(c *internal.ServiceConfig) {
		c.RegisterFn = func(app interface{}) error {
			// Convert internal.App to servicex.App
			internalApp := app.(*internal.App)
			servicexApp := &App{
				mux:           internalApp.Mux,
				logger:        internalApp.Logger,
				interceptors:  internalApp.Interceptors,
				otel:          internalApp.OtelProvider,
				container:     internalApp.Container,
				shutdownHooks: internalApp.ShutdownHooks,
				db:            internalApp.DB,
			}
			err := fn(servicexApp)
			// Copy shutdown hooks back to internal app after registration
			internalApp.ShutdownHooks = servicexApp.shutdownHooks
			return err
		}
	}
}

// WithTimeout sets the default RPC timeout in milliseconds.
func WithTimeout(timeoutMs int64) Option {
	return func(c *internal.ServiceConfig) {
		c.DefaultTimeoutMs = timeoutMs
	}
}

// WithSlowRequestThreshold sets the slow request threshold in milliseconds.
func WithSlowRequestThreshold(millis int64) Option {
	return func(c *internal.ServiceConfig) {
		c.SlowRequestMillis = millis
	}
}

// WithShutdownTimeout sets the graceful shutdown timeout.
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(c *internal.ServiceConfig) {
		c.ShutdownTimeout = timeout
	}
}

// WithDebugLogs enables debug-level logging.
// Deprecated: Use LOG_LEVEL environment variable instead for more control.
// This option is kept for backward compatibility.
func WithDebugLogs(enabled bool) Option {
	return func(c *internal.ServiceConfig) {
		c.EnableDebug = enabled
	}
}

// WithDatabase enables database support for the service.
// If cfg is nil, it will automatically read configuration from environment variables via configx.
func WithDatabase(cfg *DatabaseConfig) Option {
	return func(c *internal.ServiceConfig) {
		if cfg == nil {
			// Use configx to read database configuration from environment
			dbCfg := &configx.DatabaseConfig{}
			// Create a temporary manager to read env vars
			// Note: This is a simplified approach - in production, configx should be initialized
			dbCfg.Driver = os.Getenv("DB_DRIVER")
			if dbCfg.Driver == "" {
				dbCfg.Driver = "mysql"
			}
			dbCfg.DSN = os.Getenv("DB_DSN")
			if maxIdle := os.Getenv("DB_MAX_IDLE"); maxIdle != "" {
				if val, err := strconv.Atoi(maxIdle); err == nil {
					dbCfg.MaxIdle = val
				}
			}
			if maxOpen := os.Getenv("DB_MAX_OPEN"); maxOpen != "" {
				if val, err := strconv.Atoi(maxOpen); err == nil {
					dbCfg.MaxOpen = val
				}
			}
			if maxLifetime := os.Getenv("DB_MAX_LIFETIME"); maxLifetime != "" {
				if val, err := time.ParseDuration(maxLifetime); err == nil {
					dbCfg.MaxLifetime = val
				}
			}
			cfg = &DatabaseConfig{
				Driver:          dbCfg.Driver,
				DSN:             dbCfg.DSN,
				MaxIdleConns:    dbCfg.MaxIdle,
				MaxOpenConns:    dbCfg.MaxOpen,
				ConnMaxLifetime: dbCfg.MaxLifetime,
				PingTimeout:     5 * time.Second,
			}
		}
		c.DBConfig = &internal.DatabaseConfig{
			Driver:          cfg.Driver,
			DSN:             cfg.DSN,
			MaxIdleConns:    cfg.MaxIdleConns,
			MaxOpenConns:    cfg.MaxOpenConns,
			ConnMaxLifetime: cfg.ConnMaxLifetime,
			PingTimeout:     cfg.PingTimeout,
		}
	}
}

// WithAutoMigrate specifies database models to auto-migrate during startup.
func WithAutoMigrate(models ...any) Option {
	return func(c *internal.ServiceConfig) {
		c.AutoMigrateModels = models
	}
}

// WithAppConfig is a convenience function that combines WithConfig and WithDatabase.
// It automatically detects database configuration from the provided config struct.
// This simplifies the common pattern of using BaseConfig with database.
//
// Example:
//
//	cfg := &MyConfig{configx.BaseConfig{}} // MyConfig embeds BaseConfig
//	servicex.Run(ctx,
//	    servicex.WithService("my-service", "1.0.0"),
//	    servicex.WithLogger(logger),
//	    servicex.WithAppConfig(cfg), // Automatically handles database config
//	    servicex.WithAutoMigrate(&MyModel{}),
//	    servicex.WithRegister(register),
//	)
func WithAppConfig(cfg any) Option {
	return func(c *internal.ServiceConfig) {
		WithConfig(cfg)(c)
		// Database config is already extracted by WithConfig if BaseConfig is embedded
	}
}

// Run starts the service with the given options.
//
// Parameters:
//   - ctx: context for service lifecycle
//   - opts: functional options for service configuration
//
// Returns:
//   - error: service error if any
//
// Concurrency:
//   - Blocks until context is cancelled
//   - All components run concurrently
func Run(ctx context.Context, opts ...Option) error {
	cfg := internal.NewServiceConfig()

	for _, opt := range opts {
		opt(cfg)
	}

	runtime, err := internal.NewServiceRuntime(cfg)
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}

	return runtime.Run(ctx)
}

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
	Driver          string        `env:"DB_DRIVER" default:"mysql"`
	DSN             string        `env:"DB_DSN" default:""`
	MaxIdleConns    int           `env:"DB_MAX_IDLE" default:"10"`
	MaxOpenConns    int           `env:"DB_MAX_OPEN" default:"100"`
	ConnMaxLifetime time.Duration `env:"DB_MAX_LIFETIME" default:"1h"`
	PingTimeout     time.Duration `env:"DB_PING_TIMEOUT" default:"5s"`
}

// DatabaseMigrator defines a function that performs database migrations.
type DatabaseMigrator func(db *gorm.DB) error

// ServiceRegistrar defines a function that registers services with the application.
type ServiceRegistrar func(app *App) error

// Options documents high-level configuration shape for one-call startup.
type Options struct {
	ServiceName       string `env:"SERVICE_NAME" default:"app"`
	ServiceVersion    string `env:"SERVICE_VERSION" default:"0.0.0"`
	Config            any
	Database          *DatabaseConfig
	Migrate           DatabaseMigrator
	Register          ServiceRegistrar
	EnableHealthCheck bool          `env:"ENABLE_HEALTH_CHECK" default:"true"`
	EnableMetrics     bool          `env:"ENABLE_METRICS" default:"true"`
	EnableDebugLogs   bool          `env:"ENABLE_DEBUG_LOGS" default:"false"`
	SlowRequestMillis int64         `env:"SLOW_REQUEST_MILLIS" default:"1000"`
	PayloadAccounting bool          `env:"PAYLOAD_ACCOUNTING" default:"true"`
	ShutdownTimeout   time.Duration `env:"SHUTDOWN_TIMEOUT" default:"15s"`
	Logger            log.Logger
}

// FromBaseConfig creates a DatabaseConfig from configx.DatabaseConfig.
func FromBaseConfig(dbCfg *configx.DatabaseConfig) *DatabaseConfig {
	if dbCfg == nil {
		return nil
	}
	return &DatabaseConfig{
		Driver:          dbCfg.Driver,
		DSN:             dbCfg.DSN,
		MaxIdleConns:    dbCfg.MaxIdle,
		MaxOpenConns:    dbCfg.MaxOpen,
		ConnMaxLifetime: dbCfg.MaxLifetime,
		PingTimeout:     5 * time.Second,
	}
}
