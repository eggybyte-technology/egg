// Package internal provides internal implementation for the servicex package.
package internal

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/eggybyte-technology/egg/configx"
	"github.com/eggybyte-technology/egg/core/log"
	"github.com/eggybyte-technology/egg/logx"
	"github.com/eggybyte-technology/egg/obsx"
	"github.com/eggybyte-technology/egg/storex"
	"gorm.io/gorm"
)

// ServiceRuntime manages the service lifecycle and components.
type ServiceRuntime struct {
	config        *ServiceConfig
	logger        log.Logger
	configMgr     configx.Manager
	otelProvider  *obsx.Provider
	db            *gorm.DB
	store         storex.GORMStore
	httpServer    *http.Server
	healthServer  *http.Server
	shutdownHooks []func(context.Context) error
}

// NewServiceRuntime creates a new service runtime instance.
func NewServiceRuntime(config *ServiceConfig) (*ServiceRuntime, error) {
	return &ServiceRuntime{
		config:        config,
		shutdownHooks: []func(context.Context) error{},
	}, nil
}

// Run starts the service with all components.
func (r *ServiceRuntime) Run(ctx context.Context) error {
	// Initialize logger first
	if err := r.initializeLogger(); err != nil {
		return err
	}

	r.logger.Info("starting service",
		"service", r.config.ServiceName,
		"version", r.config.ServiceVersion,
	)

	// Initialize configuration
	if err := r.initializeConfig(ctx); err != nil {
		return err
	}

	// Initialize database if configured
	if err := r.initializeDatabase(ctx); err != nil {
		return err
	}

	// Initialize observability
	if err := r.initializeObservability(ctx); err != nil {
		return err
	}

	// Build application components
	app, err := r.buildApp()
	if err != nil {
		return err
	}

	// Register services
	if r.config.RegisterFn != nil {
		if err := r.config.RegisterFn(app); err != nil {
			return fmt.Errorf("service registration failed: %w", err)
		}
	}

	// Start servers
	if err := r.startServers(ctx, app); err != nil {
		return err
	}

	// Wait for context cancellation
	<-ctx.Done()
	r.logger.Info("shutting down service")

	// Perform graceful shutdown
	return r.gracefulShutdown(app)
}

// initializeLogger creates or uses the provided logger.
func (r *ServiceRuntime) initializeLogger() error {
	if r.config.Logger == nil {
		level := slog.LevelInfo
		if r.config.EnableDebug {
			level = slog.LevelDebug
		}
		r.logger = logx.New(
			logx.WithFormat(logx.FormatLogfmt),
			logx.WithLevel(level),
			logx.WithColor(false),
		)
		r.config.Logger = r.logger
	} else {
		r.logger = r.config.Logger
	}
	return nil
}

// initializeConfig loads and binds configuration.
func (r *ServiceRuntime) initializeConfig(ctx context.Context) error {
	if r.config.Config == nil {
		return nil
	}

	configMgr, err := configx.DefaultManager(ctx, r.logger)
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	if err := configMgr.Bind(r.config.Config); err != nil {
		return fmt.Errorf("failed to bind config: %w", err)
	}

	r.configMgr = configMgr
	r.logger.Info("configuration loaded", "keys", len(configMgr.Snapshot()))

	// Extract port configuration from BaseConfig if available
	if baseGetter, ok := r.config.Config.(interface {
		GetHTTPPort() string
		GetHealthPort() string
		GetMetricsPort() string
	}); ok {
		if port := baseGetter.GetHTTPPort(); port != "" {
			if port[0] != ':' {
				r.config.HTTPPort = ":" + port
			} else {
				r.config.HTTPPort = port
			}
		}
		if port := baseGetter.GetHealthPort(); port != "" {
			if port[0] != ':' {
				r.config.HealthPort = ":" + port
			} else {
				r.config.HealthPort = port
			}
		}
		if port := baseGetter.GetMetricsPort(); port != "" {
			if port[0] != ':' {
				r.config.MetricsPort = ":" + port
			} else {
				r.config.MetricsPort = port
			}
		}
	}

	return nil
}

// initializeDatabase initializes the database connection and performs migrations.
func (r *ServiceRuntime) initializeDatabase(ctx context.Context) error {
	if r.config.DBConfig == nil {
		return nil
	}

	r.logger.Info("initializing database", "driver", r.config.DBConfig.Driver)

	store, err := storex.NewGORMStore(storex.GORMOptions{
		DSN:             r.config.DBConfig.DSN,
		Driver:          r.config.DBConfig.Driver,
		MaxIdleConns:    r.config.DBConfig.MaxIdleConns,
		MaxOpenConns:    r.config.DBConfig.MaxOpenConns,
		ConnMaxLifetime: r.config.DBConfig.ConnMaxLifetime,
		Logger:          r.logger,
	})
	if err != nil {
		return fmt.Errorf("database init failed: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, r.config.DBConfig.PingTimeout)
	defer cancel()
	if err := store.Ping(pingCtx); err != nil {
		store.Close()
		return fmt.Errorf("database ping failed: %w", err)
	}

	r.db = store.GetDB()
	r.store = store
	r.logger.Info("database connection established successfully")

	// Run auto-migration if models provided
	if len(r.config.AutoMigrateModels) > 0 {
		if err := AutoMigrate(r.db, r.logger, r.config.AutoMigrateModels); err != nil {
			return fmt.Errorf("auto-migration failed: %w", err)
		}
	}

	// Register store in health check registry
	registry := storex.NewRegistry()
	if err := registry.Register("database", store); err != nil {
		return fmt.Errorf("failed to register database health check: %w", err)
	}

	return nil
}

// initializeObservability initializes OpenTelemetry providers.
func (r *ServiceRuntime) initializeObservability(ctx context.Context) error {
	if !r.config.EnableTracing {
		return nil
	}

	otelProvider, err := obsx.NewProvider(ctx, obsx.Options{
		ServiceName:    r.config.ServiceName,
		ServiceVersion: r.config.ServiceVersion,
	})
	if err != nil {
		r.logger.Error(err, "otel init failed, continuing without tracing")
		return nil // Non-fatal
	}

	r.otelProvider = otelProvider
	return nil
}

// buildApp creates the App instance with all components.
func (r *ServiceRuntime) buildApp() (*App, error) {
	mux := http.NewServeMux()

	// Build default interceptors
	interceptors := BuildInterceptors(
		r.logger,
		r.otelProvider,
		r.config.SlowRequestMillis,
		r.config.EnableDebug,
		true, // payloadAccounting enabled by default
	)

	app := &App{
		Mux:           mux,
		Logger:        r.logger,
		Interceptors:  interceptors,
		OtelProvider:  r.otelProvider,
		Container:     NewContainer(),
		ShutdownHooks: []func(context.Context) error{},
		DB:            r.db,
	}

	return app, nil
}

// startServers starts HTTP and health check servers.
func (r *ServiceRuntime) startServers(ctx context.Context, app *App) error {
	// Create separate health check mux
	healthMux := http.NewServeMux()
	SetupHealthEndpoints(healthMux, r.logger)

	// Start servers
	r.httpServer = &http.Server{Addr: r.config.HTTPPort, Handler: app.Mux}
	r.healthServer = &http.Server{Addr: r.config.HealthPort, Handler: healthMux}

	go func() {
		r.logger.Info("HTTP server listening", "addr", r.config.HTTPPort)
		if err := r.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			r.logger.Error(err, "HTTP server error")
		}
	}()

	go func() {
		r.logger.Info("health check server listening", "addr", r.config.HealthPort)
		if err := r.healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			r.logger.Error(err, "health server error")
		}
	}()

	return nil
}

// gracefulShutdown performs graceful shutdown of all components.
func (r *ServiceRuntime) gracefulShutdown(app *App) error {
	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), r.config.ShutdownTimeout)
	defer cancel()

	// Execute shutdown hooks in LIFO order
	for i := len(app.ShutdownHooks) - 1; i >= 0; i-- {
		if err := app.ShutdownHooks[i](shutdownCtx); err != nil {
			r.logger.Error(err, "shutdown hook failed", "index", i)
		}
	}

	// Shutdown servers
	if r.healthServer != nil {
		if err := r.healthServer.Shutdown(shutdownCtx); err != nil {
			r.logger.Error(err, "health server shutdown failed")
		}
	}

	if r.httpServer != nil {
		if err := r.httpServer.Shutdown(shutdownCtx); err != nil {
			r.logger.Error(err, "HTTP server shutdown failed")
		}
	}

	// Shutdown observability
	if r.otelProvider != nil {
		if err := r.otelProvider.Shutdown(shutdownCtx); err != nil {
			r.logger.Error(err, "otel shutdown failed")
		}
	}

	// Close database connection
	if r.store != nil {
		if err := r.store.Close(); err != nil {
			r.logger.Error(err, "database close failed")
		}
	}

	r.logger.Info("service stopped")
	return nil
}

// App provides access to service components during registration.
type App struct {
	Mux           *http.ServeMux
	Logger        log.Logger
	Interceptors  []connect.Interceptor
	OtelProvider  *obsx.Provider
	Container     *Container
	ShutdownHooks []func(context.Context) error
	DB            *gorm.DB
}

