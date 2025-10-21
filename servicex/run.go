// Package servicex provides a unified microservice initialization framework.
package servicex

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/eggybyte-technology/egg/configx"
	"github.com/eggybyte-technology/egg/core/log"
	"github.com/eggybyte-technology/egg/obsx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Run starts a microservice with the given options.
// This is the main entry point for servicex - it handles all initialization
// and provides a single function call to start a complete microservice.
func Run(ctx context.Context, opts Options) error {
	// Validate options
	if err := opts.validate(); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	// Set default logger if not provided
	if opts.Logger == nil {
		opts.Logger = &defaultLogger{}
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Initialize configuration management
	configMgr, err := configx.DefaultManager(ctx, opts.Logger)
	if err != nil {
		return fmt.Errorf("config initialization failed: %w", err)
	}

	// Bind configuration
	if err := configMgr.Bind(opts.Config); err != nil {
		return fmt.Errorf("failed to bind configuration: %w", err)
	}

	// Set up configuration hot reloading
	configMgr.OnUpdate(func(snapshot map[string]string) {
		opts.Logger.Info("Configuration updated", log.Int("keys", len(snapshot)))
	})

	// Initialize observability
	var otel *obsx.Provider
	if opts.EnableTracing {
		otel, err = obsx.NewProvider(ctx, obsx.Options{
			ServiceName:    opts.ServiceName,
			ServiceVersion: opts.ServiceVersion,
		})
		if err != nil {
			return fmt.Errorf("observability initialization failed: %w", err)
		}
	}

	// Initialize database (if configured)
	var db *gorm.DB
	var sqlDB *sql.DB
	if opts.Database != nil && opts.Database.DSN != "" {
		db, err = gorm.Open(mysql.Open(opts.Database.DSN), &gorm.Config{})
		if err != nil {
			return fmt.Errorf("database initialization failed: %w", err)
		}

		sqlDB, err = db.DB()
		if err != nil {
			return fmt.Errorf("failed to get underlying sql.DB: %w", err)
		}

		sqlDB.SetMaxIdleConns(opts.Database.MaxIdle)
		sqlDB.SetMaxOpenConns(opts.Database.MaxOpen)
		sqlDB.SetConnMaxLifetime(opts.Database.MaxLifetime)
	}

	// Run database migrations if configured
	if opts.Migrate != nil && db != nil {
		if err := opts.Migrate(db); err != nil {
			return fmt.Errorf("database migration failed: %w", err)
		}
	}

	// Initialize HTTP server
	mux := http.NewServeMux()

	// Add health check endpoint
	if opts.EnableHealthCheck {
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			if sqlDB != nil {
				if err := sqlDB.PingContext(r.Context()); err != nil {
					w.WriteHeader(http.StatusServiceUnavailable)
					w.Write([]byte("Database unhealthy"))
					return
				}
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Healthy"))
		})
	}

	// Add metrics endpoint
	if opts.EnableMetrics {
		mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("# Service metrics\n"))
		})
	}

	// Build interceptors
	interceptors := buildInterceptors(&opts, opts.Logger, otel)

	// Create app instance for registration
	app := &App{
		mux:          mux,
		logger:       opts.Logger,
		interceptors: interceptors,
		db:           db,
		otel:         otel,
	}

	// Run service registration
	if opts.Register != nil {
		if err := opts.Register(app); err != nil {
			return fmt.Errorf("service registration failed: %w", err)
		}
	}

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		opts.Logger.Info("Received shutdown signal")
		cancel()
	}()

	// Start HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			opts.Logger.Error(err, "HTTP server failed")
		}
	}()

	opts.Logger.Info("Service started successfully",
		log.Str("service_name", opts.ServiceName),
		log.Str("service_version", opts.ServiceVersion))

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), opts.ShutdownTimeout)
	defer shutdownCancel()

	opts.Logger.Info("Shutting down service")

	// Stop HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		opts.Logger.Error(err, "Failed to shutdown HTTP server")
	}

	// Stop database connection
	if sqlDB != nil {
		if err := sqlDB.Close(); err != nil {
			opts.Logger.Error(err, "Failed to close database connection")
		}
	}

	// Stop OpenTelemetry provider
	if otel != nil {
		if err := otel.Shutdown(shutdownCtx); err != nil {
			opts.Logger.Error(err, "Failed to shutdown OpenTelemetry provider")
		}
	}

	opts.Logger.Info("Service stopped gracefully")
	return nil
}

// defaultLogger provides a simple console logger implementation.
type defaultLogger struct{}

func (l *defaultLogger) With(kv ...any) log.Logger   { return l }
func (l *defaultLogger) Debug(msg string, kv ...any) { fmt.Printf("[DEBUG] %s %v\n", msg, kv) }
func (l *defaultLogger) Info(msg string, kv ...any)  { fmt.Printf("[INFO] %s %v\n", msg, kv) }
func (l *defaultLogger) Warn(msg string, kv ...any)  { fmt.Printf("[WARN] %s %v\n", msg, kv) }
func (l *defaultLogger) Error(err error, msg string, kv ...any) {
	if err != nil {
		fmt.Printf("[ERROR] %s: %v %v\n", msg, err, kv)
	} else {
		fmt.Printf("[ERROR] %s %v\n", msg, kv)
	}
}
