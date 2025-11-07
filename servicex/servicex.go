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
	internalToken string
	config        any
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
//
// This is a convenience wrapper around ProvideTyped for backward compatibility.
// Prefer ProvideTyped for better error messages.
func (a *App) Provide(constructor any) error { return a.container.Provide(constructor) }

// Resolve resolves a dependency from the DI container.
//
// This stores the resolved instance in the provided pointer.
// Prefer ResolveTyped for type-safe resolution without type assertions.
func (a *App) Resolve(target any) error { return a.container.Resolve(target) }

// ProvideTyped registers a constructor in the DI container with improved error messages.
//
// This is the recommended way to register constructors. It provides better error messages
// for common mistakes and validates the constructor signature before registration.
//
// Parameters:
//   - constructor: Function that returns one value (and optionally an error)
//
// Returns:
//   - error: nil on success; descriptive error if validation fails
//
// Usage:
//
//	err := app.ProvideTyped(func(logger log.Logger, db *gorm.DB) *MyService {
//	    return NewMyService(logger, db)
//	})
func (a *App) ProvideTyped(constructor any) error {
	return a.container.ProvideTyped(constructor)
}

// MustResolve resolves a dependency from the DI container and panics on error.
//
// This is a convenience method for startup-time dependency resolution where
// errors should cause the service to fail fast. Use Resolve for runtime resolution.
//
// Parameters:
//   - target: Pointer to the target type (must be a pointer)
//
// Panics:
//   - If resolution fails (e.g., dependency not registered)
//
// Usage:
//
//	var service *MyService
//	app.MustResolve(&service)
func (a *App) MustResolve(target any) {
	if err := a.container.Resolve(target); err != nil {
		panic(fmt.Sprintf("MustResolve failed: %v", err))
	}
}

// ResolveTyped resolves a dependency of the specified type using generics.
//
// This is a type-safe alternative to Resolve that eliminates the need for type assertions.
// The type parameter T must have been registered via Provide or ProvideTyped.
//
// Parameters:
//   - app: Application instance
//
// Returns:
//   - T: resolved instance of the requested type
//   - error: nil on success; error if resolution fails
//
// Usage:
//
//	service, err := ResolveTyped[*MyService](app)
//	if err != nil {
//	    return err
//	}
func ResolveTyped[T any](app *App) (T, error) {
	return internal.ResolveTyped[T](app.container)
}

// CallService wraps a service method call with automatic error handling and Connect response conversion.
//
// This is the simplest way to create Connect handlers from service methods.
// It automatically handles:
// - Converting Connect request to proto message (req.Msg)
// - Calling the service method
// - Converting proto response to Connect response
// - Error handling
// - Debug logging
//
// Parameters:
//   - serviceMethod: Service method that takes proto request pointer and returns proto response pointer
//   - logger: Logger instance for logging
//   - methodName: Name of the method (for logging purposes)
//
// Returns:
//   - HandlerFunc: Wrapped handler function ready for Connect
//
// Usage:
//
//	func (h *UserHandler) CreateUser(ctx context.Context, req *connect.Request[userv1.CreateUserRequest]) (*connect.Response[userv1.CreateUserResponse], error) {
//	    return CallService(h.service.CreateUser, h.logger, "CreateUser")(ctx, req)
//	}
func CallService[TReq, TResp any](serviceMethod func(ctx context.Context, req *TReq) (*TResp, error), logger log.Logger, methodName string) internal.HandlerFunc[TReq, TResp] {
	return internal.CallService(serviceMethod, logger, methodName)
}

// CallServiceWithToken wraps a service method call with token validation.
//
// This combines CallService with internal token validation for admin operations.
func CallServiceWithToken[TReq, TResp any](serviceMethod func(ctx context.Context, req *TReq) (*TResp, error), logger log.Logger, internalToken string, methodName string) internal.HandlerFunc[TReq, TResp] {
	return internal.CallServiceWithToken(serviceMethod, logger, internalToken, methodName)
}

// ResolveAndRegister resolves a dependency from the container and registers handlers.
//
// This is a simpler variant that assumes constructors have already been registered.
func ResolveAndRegister[T any](app *App, registerFn func(T, *App) error) error {
	dep, err := ResolveTyped[T](app)
	if err != nil {
		return fmt.Errorf("failed to resolve dependency: %w", err)
	}
	return registerFn(dep, app)
}

// RegisterServices registers common constructors and custom constructors in one call.
//
// This is the recommended way to register services. It combines ProvideCommonConstructors
// and ProvideMany into a single call, reducing boilerplate.
//
// Parameters:
//   - app: Application instance
//   - constructors: Optional map of constructor names to constructor functions (may be nil)
//
// Returns:
//   - error: nil on success; error if registration fails
//
// Usage:
//
//	if err := servicex.RegisterServices(app, map[string]any{
//	    "repository": func(db *gorm.DB) repository.UserRepository {
//	        return repository.NewUserRepository(db)
//	    },
//	    "service": func(repo repository.UserRepository, logger log.Logger) service.UserService {
//	        return service.NewUserService(repo, logger)
//	    },
//	}); err != nil {
//	    return err
//	}
func RegisterServices(app *App, constructors map[string]any) error {
	return internal.RegisterServices(app, constructors)
}

// ProvideCommonConstructors registers common service constructors in the DI container.
//
// This helper registers standard framework dependencies (logger, database) that
// are commonly needed by services. This reduces boilerplate in service registration.
func ProvideCommonConstructors(app *App) error {
	return internal.ProvideCommonConstructors(app)
}

// ProvideMany registers multiple constructors, stopping on first error.
//
// This helper simplifies bulk registration of constructors with unified error handling.
// If any constructor fails to register, it returns an error with a descriptive message.
//
// Parameters:
//   - app: Application instance
//   - constructors: Map of constructor names to constructor functions
//
// Returns:
//   - error: nil on success; error with constructor name if registration fails
//
// Usage:
//
//	if err := ProvideMany(app, map[string]any{
//	    "repository": func(db *gorm.DB) repository.UserRepository {
//	        return repository.NewUserRepository(db)
//	    },
//	    "service": func(repo repository.UserRepository, logger log.Logger) service.UserService {
//	        return service.NewUserService(repo, logger, nil)
//	    },
//	}); err != nil {
//	    return err
//	}
func ProvideMany(app *App, constructors map[string]any) error {
	return internal.ProvideMany(app, constructors)
}

// OptionalClientConfig holds configuration for creating an optional client.
type OptionalClientConfig[T any] = internal.OptionalClientConfig[T]

// CreateOptionalClient creates an optional client from configuration and logs the result.
//
// This helper simplifies the common pattern of creating optional service clients
// with consistent logging. If the URL is not configured, it returns zero value and logs
// that the client is not configured.
func CreateOptionalClient[T any](logger log.Logger, internalToken string, config OptionalClientConfig[T]) T {
	return internal.CreateOptionalClient(logger, internalToken, config)
}

// ClientRegistryConfig holds configuration for registering multiple optional clients.
type ClientRegistryConfig = internal.ClientRegistryConfig

// ClientConfig holds configuration for a single client in the registry.
type ClientConfig = internal.ClientConfig

// RegisterOptionalClients registers multiple optional clients from configuration.
//
// This helper simplifies batch registration of clients by automatically extracting
// URLs from the config struct using reflection. It supports multiple clients and
// provides unified logging.
func RegisterOptionalClients(logger log.Logger, internalToken string, config ClientRegistryConfig) map[string]any {
	return internal.RegisterOptionalClients(logger, internalToken, config)
}

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

// InternalToken returns the configured internal token from environment.
// Returns empty string if INTERNAL_TOKEN is not set.
func (a *App) InternalToken() string { return a.internalToken }

// Config returns the configuration struct that was passed to WithConfig.
// Returns nil if no config was provided.
//
// Usage:
//
//	cfg := app.Config().(*MyConfig)
//	if cfg != nil {
//	    // Use config fields
//	}
func (a *App) Config() any { return a.config }

// RegisterConnectHandler registers a Connect service handler with automatic interceptor injection.
//
// This is a convenience method that simplifies Connect handler registration by automatically
// applying the configured interceptors. It logs the registration for observability.
//
// Parameters:
//   - handler: Connect service handler implementation
//   - newHandler: Function that creates the Connect handler (e.g., userv1connect.NewUserServiceHandler)
//
// Returns:
//   - error: nil on success; error if registration fails
//
// Usage:
//
//	path, connectHandler := userv1connect.NewUserServiceHandler(
//	    userHandler,
//	    connect.WithInterceptors(app.Interceptors()...),
//	)
//	app.Mux().Handle(path, connectHandler)
//
// Simplified:
//
//	err := app.RegisterConnectHandler(userHandler, func(handler any, opts ...connect.HandlerOption) (string, http.Handler) {
//	    return userv1connect.NewUserServiceHandler(handler.(userv1connect.UserServiceHandler), opts...)
//	})
func (a *App) RegisterConnectHandler(handler any, newHandler func(handler any, opts ...connect.HandlerOption) (string, http.Handler)) error {
	path, connectHandler := newHandler(handler, connect.WithInterceptors(a.interceptors...))
	a.mux.Handle(path, connectHandler)
	a.logger.Info("registered Connect handler", "path", path)
	return nil
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
				internalToken: internalApp.InternalToken,
				config:        internalApp.Config,
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
