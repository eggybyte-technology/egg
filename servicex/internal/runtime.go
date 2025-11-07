// Package internal provides internal implementation for the servicex package.
package internal

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"go.eggybyte.com/egg/configx"
	"go.eggybyte.com/egg/core/log"
	"go.eggybyte.com/egg/logx"
	"go.eggybyte.com/egg/obsx"
	"go.eggybyte.com/egg/storex"
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
	metricsServer *http.Server
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
	// Pre-bind basic configuration to get log level before logger initialization
	// This ensures logger uses the correct level from BaseConfig.LogLevel (via configx)
	if err := r.preBindBaseConfig(ctx); err != nil {
		return err
	}

	// Initialize logger with log level from BaseConfig
	if err := r.initializeLogger(); err != nil {
		return err
	}

	r.logger.Info("starting service",
		"service", r.config.ServiceName,
		"version", r.config.ServiceVersion,
		"build_time", BuildTime,
	)

	// Initialize full configuration with manager
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

// preBindBaseConfig performs lightweight pre-binding of critical BaseConfig fields.
//
// This method extracts and populates BaseConfig.LogLevel from environment before logger
// initialization, ensuring the logger uses the correct level. Uses a minimal approach
// that reads only what's needed, avoiding full config manager overhead.
//
// Design principles:
//   - Simple and focused: only handles LogLevel for logger initialization
//   - Type-safe: uses interface-based extraction (no reflection)
//   - Fail-safe: continues with defaults if environment read fails
func (r *ServiceRuntime) preBindBaseConfig(ctx context.Context) error {
	if r.config.Config == nil {
		return nil
	}

	// Extract BaseConfig using type-safe interface
	baseCfg, ok := ExtractBaseConfig(r.config.Config)
	if !ok {
		return nil // No BaseConfig available, skip pre-binding
	}

	// Load environment variables through configx (unified configuration source)
	envSource := configx.NewEnvSource(configx.EnvOptions{})
	snapshot, err := envSource.Load(ctx)
	if err != nil {
		// Fail-safe: use default log level if env loading fails
		baseCfg.LogLevel = "info"
		return nil
	}

	// Populate LogLevel from environment with fallback to default
	if logLevel, exists := snapshot["LOG_LEVEL"]; exists && logLevel != "" {
		baseCfg.LogLevel = logLevel
	} else {
		baseCfg.LogLevel = "info"
	}

	return nil
}

// initializeLogger creates or uses the provided logger.
//
// If no logger is provided via WithLogger(), creates a default logger with:
//   - Console format (human-readable, colored output for development)
//   - Log level from BaseConfig.LogLevel (populated via configx)
//   - Color enabled for better readability
//
// This ensures all configuration is unified through configx, not direct environment reads.
func (r *ServiceRuntime) initializeLogger() error {
	if r.config.Logger == nil {
		logLevelStr := "info" // default

		// Extract log level from BaseConfig (populated by preBindBaseConfig via configx)
		if r.config.Config != nil {
			if baseCfg, ok := ExtractBaseConfig(r.config.Config); ok {
				if baseCfg.LogLevel != "" {
					logLevelStr = baseCfg.LogLevel
				}
			}
		}

		// Parse and create logger with configured level
		logLevel := logx.ParseLevel(logLevelStr)
		r.logger = logx.New(
			logx.WithFormat(logx.FormatConsole),
			logx.WithLevel(logLevel),
			logx.WithColor(true),
		)
		r.config.Logger = r.logger

		r.logger.Info("logger initialized", "log_level", logLevelStr, "source", "configx")
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
			r.config.HTTPPort = parsePort(port, 8080)
		}
		if port := baseGetter.GetHealthPort(); port != "" {
			r.config.HealthPort = parsePort(port, 8081)
		}
		if port := baseGetter.GetMetricsPort(); port != "" {
			r.config.MetricsPort = parsePort(port, 9091)
		}
	}

	// Extract database configuration from BaseConfig after binding environment variables
	// This ensures DSN is populated from environment variables
	if r.config.DBConfig == nil {
		if baseCfg, ok := ExtractBaseConfig(r.config.Config); ok {
			if baseCfg.Database.DSN != "" {
				r.config.DBConfig = &DatabaseConfig{
					Driver:          baseCfg.Database.Driver,
					DSN:             baseCfg.Database.DSN,
					MaxIdleConns:    baseCfg.Database.MaxIdle,
					MaxOpenConns:    baseCfg.Database.MaxOpen,
					ConnMaxLifetime: baseCfg.Database.MaxLifetime,
					PingTimeout:     5 * time.Second,
				}
				r.logger.Info("database configuration detected from BaseConfig",
					"driver", baseCfg.Database.Driver,
					"dsn_preview", maskDSN(baseCfg.Database.DSN))
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

// initializeObservability initializes metrics provider.
func (r *ServiceRuntime) initializeObservability(ctx context.Context) error {
	// Skip if metrics is disabled
	if !r.config.EnableMetrics {
		return nil
	}

	otelProvider, err := obsx.NewProvider(ctx, obsx.Options{
		ServiceName:    r.config.ServiceName,
		ServiceVersion: r.config.ServiceVersion,
	})
	if err != nil {
		r.logger.Error(err, "metrics provider init failed, continuing without observability")
		return nil // Non-fatal
	}

	r.otelProvider = otelProvider
	r.logger.Info("metrics provider initialized")

	// Enable additional metrics based on MetricsConfig
	if r.config.MetricsConfig != nil {
		if r.config.MetricsConfig.EnableRuntime {
			if err := otelProvider.EnableRuntimeMetrics(ctx); err != nil {
				r.logger.Error(err, "failed to enable runtime metrics")
			} else {
				r.logger.Info("runtime metrics enabled")
			}
		}

		if r.config.MetricsConfig.EnableProcess {
			if err := otelProvider.EnableProcessMetrics(ctx); err != nil {
				r.logger.Error(err, "failed to enable process metrics")
			} else {
				r.logger.Info("process metrics enabled")
			}
		}

		if r.config.MetricsConfig.EnableDB && r.db != nil {
			sqlDB, err := r.db.DB()
			if err == nil {
				if err := otelProvider.RegisterDBMetrics(r.config.ServiceName, sqlDB); err != nil {
					r.logger.Error(err, "failed to register database metrics")
				} else {
					r.logger.Info("database metrics enabled")
				}
			}
		}
	}

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

	// Read internal token from environment
	internalToken := ""
	if r.config.Config != nil {
		if baseCfg, ok := ExtractBaseConfig(r.config.Config); ok {
			internalToken = baseCfg.Security.InternalToken
		}
	}

	app := &App{
		Mux:           mux,
		Logger:        r.logger,
		Interceptors:  interceptors,
		OtelProvider:  r.otelProvider,
		Container:     NewContainer(),
		ShutdownHooks: []func(context.Context) error{},
		DB:            r.db,
		InternalToken: internalToken,
		Config:        r.config.Config,
	}

	return app, nil
}

// startServers starts HTTP and health check servers.
func (r *ServiceRuntime) startServers(ctx context.Context, app *App) error {
	// Create separate health check mux
	healthMux := http.NewServeMux()
	SetupHealthEndpoints(healthMux, r.logger)

	// Start servers
	httpAddr := fmt.Sprintf(":%d", r.config.HTTPPort)
	healthAddr := fmt.Sprintf(":%d", r.config.HealthPort)
	r.httpServer = &http.Server{Addr: httpAddr, Handler: app.Mux}
	r.healthServer = &http.Server{Addr: healthAddr, Handler: healthMux}

	go func() {
		r.logger.Info("HTTP server listening", "port", r.config.HTTPPort)
		if err := r.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			r.logger.Error(err, "HTTP server error")
		}
	}()

	go func() {
		r.logger.Info("health check server listening", "port", r.config.HealthPort)
		if err := r.healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			r.logger.Error(err, "health server error")
		}
	}()

	// Start metrics server if enabled and observability is initialized
	if r.config.EnableMetrics && r.otelProvider != nil {
		metricsMux := http.NewServeMux()
		metricsMux.Handle("/metrics", r.otelProvider.PrometheusHandler())

		metricsAddr := fmt.Sprintf(":%d", r.config.MetricsPort)
		r.metricsServer = &http.Server{Addr: metricsAddr, Handler: metricsMux}

		go func() {
			r.logger.Info("metrics server listening", "port", r.config.MetricsPort)
			if err := r.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				r.logger.Error(err, "metrics server error")
			}
		}()
	}

	return nil
}

// parsePort extracts port number from string like ":8080" or "8080" and returns as int.
// Returns defaultPort if parsing fails.
func parsePort(portStr string, defaultPort int) int {
	// Remove leading colon if present
	trimmed := strings.TrimPrefix(portStr, ":")
	if trimmed == "" {
		return defaultPort
	}

	// Try to parse as integer
	if port, err := strconv.Atoi(trimmed); err == nil {
		return port
	}

	return defaultPort
}

// maskDSN masks sensitive information in DSN for logging.
func maskDSN(dsn string) string {
	// Mask password in DSN
	// Example: user:password@tcp(host:port)/db -> user:***@tcp(host:port)/db
	if idx := strings.Index(dsn, "@"); idx > 0 {
		if colonIdx := strings.LastIndex(dsn[:idx], ":"); colonIdx > 0 {
			return dsn[:colonIdx+1] + "***" + dsn[idx:]
		}
	}
	return dsn
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
	if r.metricsServer != nil {
		if err := r.metricsServer.Shutdown(shutdownCtx); err != nil {
			r.logger.Error(err, "metrics server shutdown failed")
		}
	}

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
	InternalToken string
	Config        any
}
