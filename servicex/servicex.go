// Package servicex provides a unified microservice initialization framework with DI.
package servicex

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/eggybyte-technology/egg/configx"
	"github.com/eggybyte-technology/egg/connectx"
	"github.com/eggybyte-technology/egg/core/log"
	"github.com/eggybyte-technology/egg/logx"
	"github.com/eggybyte-technology/egg/obsx"
	"github.com/eggybyte-technology/egg/runtimex"
	"gorm.io/gorm"
)

// Option is a functional option for configuring the service.
type Option func(*serviceConfig)

// serviceConfig holds the service configuration.
type serviceConfig struct {
	serviceName    string
	serviceVersion string
	config         any
	logger         log.Logger
	enableTracing  bool
	enableMetrics  bool
	registerFn     func(*App) error

	// Server ports
	httpPort    string
	healthPort  string
	metricsPort string

	// Connect options
	defaultTimeoutMs  int64
	slowRequestMillis int64

	// Database
	dbConfig          *DatabaseConfig
	autoMigrateModels []any

	// Shutdown
	shutdownTimeout time.Duration
	shutdownHooks   []func(context.Context) error
}

// WithService sets the service name and version.
func WithService(name, version string) Option {
	return func(c *serviceConfig) {
		c.serviceName = name
		c.serviceVersion = version
	}
}

// WithConfig sets the configuration struct.
func WithConfig(cfg any) Option {
	return func(c *serviceConfig) {
		c.config = cfg
	}
}

// WithLogger sets the logger.
func WithLogger(logger log.Logger) Option {
	return func(c *serviceConfig) {
		c.logger = logger
	}
}

// WithTracing enables tracing.
func WithTracing(enabled bool) Option {
	return func(c *serviceConfig) {
		c.enableTracing = enabled
	}
}

// WithMetrics enables metrics.
func WithMetrics(enabled bool) Option {
	return func(c *serviceConfig) {
		c.enableMetrics = enabled
	}
}

// WithRegister sets the service registration function.
func WithRegister(fn func(*App) error) Option {
	return func(c *serviceConfig) {
		c.registerFn = fn
	}
}

// WithTimeout sets the default RPC timeout in milliseconds.
func WithTimeout(timeoutMs int64) Option {
	return func(c *serviceConfig) {
		c.defaultTimeoutMs = timeoutMs
	}
}

// WithSlowRequestThreshold sets the slow request threshold in milliseconds.
func WithSlowRequestThreshold(millis int64) Option {
	return func(c *serviceConfig) {
		c.slowRequestMillis = millis
	}
}

// WithShutdownTimeout sets the graceful shutdown timeout.
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(c *serviceConfig) {
		c.shutdownTimeout = timeout
	}
}

// Run starts the service with the given options.
func Run(ctx context.Context, opts ...Option) error {
	// Build configuration
	cfg := &serviceConfig{
		serviceName:       "app",
		serviceVersion:    "0.0.0",
		enableTracing:     true,
		enableMetrics:     true,
		httpPort:          ":8080",
		healthPort:        ":8081",
		metricsPort:       ":9091",
		defaultTimeoutMs:  30000,
		slowRequestMillis: 1000,
		shutdownTimeout:   15 * time.Second,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Create or use provided logger
	if cfg.logger == nil {
		// Use logx as default logger with logfmt format
		cfg.logger = logx.New(
			logx.WithFormat(logx.FormatLogfmt),
			logx.WithLevel(slog.LevelInfo),
			logx.WithColor(false),
		)
	}

	// Log startup banner with service identity (only at initialization)
	cfg.logger.Info("starting service",
		"service", cfg.serviceName,
		"version", cfg.serviceVersion,
	)

	// Initialize configuration if provided
	var configMgr configx.Manager
	var err error
	if cfg.config != nil {
		configMgr, err = configx.DefaultManager(ctx, cfg.logger)
		if err != nil {
			return fmt.Errorf("config init failed: %w", err)
		}
		if err := configMgr.Bind(cfg.config); err != nil {
			return fmt.Errorf("config bind failed: %w", err)
		}

		// Extract port configuration from BaseConfig if available
		if baseGetter, ok := cfg.config.(interface {
			GetHTTPPort() string
			GetHealthPort() string
			GetMetricsPort() string
		}); ok {
			if port := baseGetter.GetHTTPPort(); port != "" {
				cfg.httpPort = port
			}
			if port := baseGetter.GetHealthPort(); port != "" {
				cfg.healthPort = port
			}
			if port := baseGetter.GetMetricsPort(); port != "" {
				cfg.metricsPort = port
			}
		}
	}

	// Initialize database if configured
	var db any // Using any to avoid import cycle; will be *gorm.DB
	if cfg.dbConfig != nil {
		cfg.logger.Info("initializing database", "driver", cfg.dbConfig.Driver)
		db, err = initDatabase(ctx, cfg.dbConfig, cfg.logger)
		if err != nil {
			return fmt.Errorf("database init failed: %w", err)
		}

		// Run auto-migration if models provided
		if len(cfg.autoMigrateModels) > 0 {
			if gormDB, ok := db.(interface{ AutoMigrate(...interface{}) error }); ok {
				if err := autoMigrate(gormDB.(*gorm.DB), cfg.logger, cfg.autoMigrateModels); err != nil {
					return fmt.Errorf("auto-migration failed: %w", err)
				}
			}
		}
	}

	// Initialize observability
	var otelProvider *obsx.Provider
	if cfg.enableTracing {
		otelProvider, err = obsx.NewProvider(ctx, obsx.Options{
			ServiceName:    cfg.serviceName,
			ServiceVersion: cfg.serviceVersion,
		})
		if err != nil {
			cfg.logger.Error(err, "otel init failed, continuing without tracing")
		}
	}

	// Create HTTP mux for application routes
	mux := http.NewServeMux()

	// Create separate health check mux
	healthMux := http.NewServeMux()

	// Register health check endpoints on health mux
	healthMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check all registered health checkers
		ctx := r.Context()
		if err := runtimex.CheckHealth(ctx); err != nil {
			cfg.logger.Error(err, "health check failed")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unhealthy","error":"%s"}`, err.Error())
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"healthy"}`)
	})

	healthMux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Readiness is same as health for now
		ctx := r.Context()
		if err := runtimex.CheckHealth(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"not_ready","error":"%s"}`, err.Error())
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ready"}`)
	})

	healthMux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"alive"}`)
	})

	// Build default interceptors
	interceptors := connectx.DefaultInterceptors(connectx.Options{
		Logger:            cfg.logger,
		Otel:              otelProvider,
		DefaultTimeoutMs:  cfg.defaultTimeoutMs,
		EnableTimeout:     true,
		SlowRequestMillis: cfg.slowRequestMillis,
	})

	// Create App instance for DI and registration
	app := &App{
		mux:           mux,
		logger:        cfg.logger,
		interceptors:  interceptors,
		otel:          otelProvider,
		container:     newContainer(),
		shutdownHooks: cfg.shutdownHooks,
	}

	// Set database if initialized
	if db != nil {
		if gormDB, ok := db.(*gorm.DB); ok {
			app.db = gormDB
		}
	}

	// Register services
	if cfg.registerFn != nil {
		if err := cfg.registerFn(app); err != nil {
			return fmt.Errorf("service registration failed: %w", err)
		}
	}

	// Start HTTP server for application routes
	httpServer := &http.Server{
		Addr:    cfg.httpPort,
		Handler: mux,
	}

	// Start health check server
	healthServer := &http.Server{
		Addr:    cfg.healthPort,
		Handler: healthMux,
	}

	// Start HTTP server
	go func() {
		cfg.logger.Info("HTTP server listening", "addr", cfg.httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cfg.logger.Error(err, "HTTP server error")
		}
	}()

	// Start health check server
	go func() {
		cfg.logger.Info("health check server listening", "addr", cfg.healthPort)
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cfg.logger.Error(err, "health server error")
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	cfg.logger.Info("shutting down service")

	// Execute shutdown hooks in LIFO order
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.shutdownTimeout)
	defer cancel()

	for i := len(app.shutdownHooks) - 1; i >= 0; i-- {
		if err := app.shutdownHooks[i](shutdownCtx); err != nil {
			cfg.logger.Error(err, "shutdown hook failed", "index", i)
		}
	}

	// Shutdown health check server
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		cfg.logger.Error(err, "health server shutdown failed")
	}

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		cfg.logger.Error(err, "HTTP server shutdown failed")
	}

	// Shutdown observability
	if otelProvider != nil {
		if err := otelProvider.Shutdown(shutdownCtx); err != nil {
			cfg.logger.Error(err, "otel shutdown failed")
		}
	}

	cfg.logger.Info("service stopped")
	return nil
}
