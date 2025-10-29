// Package obsx provides Prometheus-based metrics collection for Go applications.
//
// Overview:
//   - Responsibility: Bootstrap OpenTelemetry metrics provider with Prometheus export
//   - Key Types: Options for configuration, Provider for managing lifecycle
//   - Concurrency Model: Provider is safe for concurrent use
//   - Error Semantics: NewProvider returns error for initialization failures
//   - Performance Notes: Lightweight metrics collection without tracing overhead
//
// Usage:
//
//	provider, err := obsx.NewProvider(ctx, obsx.Options{
//	  ServiceName: "my-service",
//	  ServiceVersion: "1.0.0",
//	})
//	if err := provider.EnableRuntimeMetrics(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer provider.Shutdown(ctx)
package obsx

import (
	"context"
	"database/sql"
	"net/http"

	"go.eggybyte.com/egg/obsx/internal"
	api "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)

// Options holds configuration for the metrics provider.
type Options struct {
	ServiceName    string            // Service name for metrics
	ServiceVersion string            // Service version
	ResourceAttrs  map[string]string // Additional resource attributes
}

// Provider manages OpenTelemetry metrics provider with Prometheus export.
// The provider must be shut down when no longer needed.
type Provider struct {
	impl *internal.Provider
}

// MeterProvider returns the OpenTelemetry meter provider.
func (p *Provider) MeterProvider() *metric.MeterProvider {
	return p.impl.MeterProvider
}

// PrometheusHandler returns an HTTP handler for the Prometheus metrics endpoint.
// This handler exposes metrics in Prometheus text format suitable for scraping.
//
// Returns:
//   - http.Handler: handler that serves Prometheus metrics at any path
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Metrics are collected on-demand when the endpoint is scraped
//
// Example:
//
//	mux := http.NewServeMux()
//	mux.Handle("/metrics", provider.PrometheusHandler())
func (p *Provider) PrometheusHandler() http.Handler {
	return p.impl.GetPrometheusHandler()
}

// Meter returns an OpenTelemetry Meter for creating custom metrics.
// The meter name should be the service or component name.
//
// Parameters:
//   - name: meter name (e.g., "user-service", "payment-processor")
//
// Returns:
//   - api/metric.Meter: meter instance for creating counters, histograms, and gauges
//
// Concurrency:
//   - Safe for concurrent use
//
// Example:
//
//	meter := provider.Meter("user-service")
//	counter, _ := meter.Int64Counter("user.registrations.total")
//	counter.Add(ctx, 1)
func (p *Provider) Meter(name string) api.Meter {
	return p.impl.MeterProvider.Meter(name)
}

// NewProvider creates a new metrics provider with Prometheus export.
// The provider must be shut down when no longer needed.
//
// Parameters:
//   - ctx: context for provider initialization
//   - opts: provider configuration options
//
// Returns:
//   - *Provider: initialized provider instance
//   - error: initialization error if any
//
// Concurrency:
//   - Safe to call from multiple goroutines
//
// Performance:
//   - Lightweight metrics collection with local Prometheus scraping only
func NewProvider(ctx context.Context, opts Options) (*Provider, error) {
	impl, err := internal.NewProvider(ctx, internal.ProviderOptions{
		ServiceName:    opts.ServiceName,
		ServiceVersion: opts.ServiceVersion,
		ResourceAttrs:  opts.ResourceAttrs,
	})
	if err != nil {
		return nil, err
	}

	return &Provider{impl: impl}, nil
}

// Shutdown gracefully shuts down the provider.
// This should be called when the application is shutting down.
//
// Parameters:
//   - ctx: context with shutdown timeout
//
// Returns:
//   - error: shutdown error if any
//
// Concurrency:
//   - Safe to call from multiple goroutines
//   - Blocks until shutdown completes or timeout
func (p *Provider) Shutdown(ctx context.Context) error {
	return p.impl.Shutdown(ctx)
}

// EnableRuntimeMetrics starts collecting Go runtime metrics.
// It registers metrics for goroutines, GC, and memory usage.
//
// Metrics collected:
//   - process_runtime_go_goroutines: Current number of goroutines
//   - process_runtime_go_gc_count_total: Total number of GC cycles
//   - process_runtime_go_memory_heap_bytes: Heap memory in bytes
//   - process_runtime_go_memory_stack_bytes: Stack memory in bytes
//
// Parameters:
//   - ctx: context for initialization
//
// Returns:
//   - error: initialization error if any
//
// Concurrency:
//   - Safe to call multiple times (idempotent)
//
// Performance:
//   - Metrics collected on scrape by OpenTelemetry SDK
//
// Example:
//
//	provider, _ := obsx.NewProvider(ctx, obsx.Options{...})
//	if err := provider.EnableRuntimeMetrics(ctx); err != nil {
//	    log.Fatal(err)
//	}
func (p *Provider) EnableRuntimeMetrics(ctx context.Context) error {
	return internal.EnableRuntimeMetrics(ctx, p.impl.MeterProvider)
}

// EnableProcessMetrics starts collecting process-level metrics.
// It registers metrics for CPU, memory, and process uptime.
//
// Metrics collected:
//   - process_cpu_seconds_total: Total CPU time consumed
//   - process_memory_rss_bytes: Resident memory size
//   - process_start_time_seconds: Process start time as Unix timestamp
//   - process_uptime_seconds: Process uptime in seconds
//
// Parameters:
//   - ctx: context for initialization
//
// Returns:
//   - error: initialization error if any
//
// Concurrency:
//   - Safe to call multiple times (idempotent)
//
// Performance:
//   - Metrics collected on scrape by OpenTelemetry SDK
//
// Example:
//
//	provider, _ := obsx.NewProvider(ctx, obsx.Options{...})
//	if err := provider.EnableProcessMetrics(ctx); err != nil {
//	    log.Fatal(err)
//	}
func (p *Provider) EnableProcessMetrics(ctx context.Context) error {
	return internal.EnableProcessMetrics(ctx, p.impl.MeterProvider)
}

// RegisterDBMetrics registers metrics for a database connection pool.
// Metrics are collected from sql.DBStats periodically.
//
// Metrics collected:
//   - db_pool_open_connections: Number of established connections
//   - db_pool_in_use: Number of connections currently in use
//   - db_pool_idle: Number of idle connections
//   - db_pool_wait_count_total: Total number of connections waited for
//   - db_pool_wait_seconds_total: Total time blocked waiting for connections
//   - db_pool_max_open: Maximum number of open connections
//
// Parameters:
//   - name: database instance name for labeling (e.g., "main", "cache")
//   - db: sql.DB instance to monitor
//
// Returns:
//   - error: registration error if any
//
// Concurrency:
//   - Safe to call multiple times with different names
//
// Performance:
//   - Stats collected on scrape by OpenTelemetry SDK
//
// Example:
//
//	provider, _ := obsx.NewProvider(ctx, obsx.Options{...})
//	sqlDB, _ := sql.Open("mysql", dsn)
//	if err := provider.RegisterDBMetrics("main", sqlDB); err != nil {
//	    log.Fatal(err)
//	}
func (p *Provider) RegisterDBMetrics(name string, db *sql.DB) error {
	return internal.RegisterDBMetrics(name, db, p.impl.MeterProvider)
}

// RegisterGORMMetrics registers metrics for a GORM database connection pool.
// This is a convenience wrapper around RegisterDBMetrics.
//
// Parameters:
//   - name: database instance name for labeling
//   - gormDB: gorm.DB instance to monitor
//
// Returns:
//   - error: registration error if any
//
// Example:
//
//	provider, _ := obsx.NewProvider(ctx, obsx.Options{...})
//	db, _ := gorm.Open(mysql.Open(dsn))
//	if err := provider.RegisterGORMMetrics("main", db); err != nil {
//	    log.Fatal(err)
//	}
func (p *Provider) RegisterGORMMetrics(name string, gormDB interface{ DB() (*sql.DB, error) }) error {
	return internal.RegisterGORMMetrics(name, gormDB, p.impl.MeterProvider)
}
