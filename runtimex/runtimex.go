// Package runtimex provides runtime lifecycle management and unified port strategy.
//
// Overview:
//   - Responsibility: Manage service lifecycle, HTTP/RPC servers, health/metrics endpoints
//   - Key Types: Service interface, Options for configuration, Endpoint for address binding
//   - Concurrency Model: All services run concurrently, graceful shutdown supported
//   - Error Semantics: Start/Stop methods return errors for failure cases
//   - Performance Notes: Supports HTTP/2 and HTTP/2 Cleartext (h2c) for better performance
//
// Usage:
//
//	err := runtimex.Run(ctx, []Service{myService}, runtimex.Options{
//	  Logger: logger,
//	  HTTP: &runtimex.HTTPOptions{Addr: ":8080", H2C: true, Mux: mux},
//	  Health: &runtimex.Endpoint{Addr: ":8081"},
//	  Metrics: &runtimex.Endpoint{Addr: ":9091"},
//	})
package runtimex

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/eggybyte-technology/egg/core/log"
	"github.com/eggybyte-technology/egg/runtimex/internal"
)

// Service defines the interface for services that can be started and stopped.
// Services must be safe for concurrent use and handle context cancellation.
type Service interface {
	// Start begins the service operation.
	// The context should be honored for cancellation.
	// Returns an error if the service fails to start.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the service.
	// The context should be honored for shutdown timeout.
	// Returns an error if the service fails to stop gracefully.
	Stop(ctx context.Context) error
}

// Endpoint represents a network endpoint with an address.
type Endpoint struct {
	Addr string // Network address (e.g., ":8081", "localhost:9091")
}

// HTTPOptions configures the HTTP server.
type HTTPOptions struct {
	Addr string         // Server address (e.g., ":8080")
	H2C  bool           // Enable HTTP/2 Cleartext support
	Mux  *http.ServeMux // HTTP request multiplexer
}

// RPCOptions configures the RPC server (for split port strategy).
type RPCOptions struct {
	Addr string // Server address (e.g., ":9090")
}

// Options holds configuration for the runtime.
type Options struct {
	Logger          log.Logger    // Logger for runtime operations
	HTTP            *HTTPOptions  // HTTP server options (required for single port)
	RPC             *RPCOptions   // RPC server options (optional, for split ports)
	Health          *Endpoint     // Health check endpoint (recommended)
	Metrics         *Endpoint     // Metrics endpoint (recommended)
	ShutdownTimeout time.Duration // Graceful shutdown timeout
}

// Run starts all services and manages their lifecycle.
// This function blocks until the context is cancelled or an error occurs.
// Services are started concurrently and stopped gracefully on shutdown.
//
// Parameters:
//   - ctx: context for lifecycle management
//   - services: list of services to manage
//   - opts: runtime configuration options
//
// Returns:
//   - error: runtime error if any
//
// Concurrency:
//   - Services are started and stopped concurrently
//   - Blocks until context is cancelled
func Run(ctx context.Context, services []Service, opts Options) error {
	if opts.Logger == nil {
		return fmt.Errorf("logger is required")
	}

	// Set default shutdown timeout
	shutdownTimeout := opts.ShutdownTimeout
	if shutdownTimeout == 0 {
		shutdownTimeout = 15 * time.Second
	}

	// Convert services to internal type
	internalServices := make([]internal.Service, len(services))
	for i, service := range services {
		internalServices[i] = service
	}

	// Create runtime instance
	runtime := internal.NewRuntime(opts.Logger, internalServices, shutdownTimeout)

	// Configure servers
	if opts.HTTP != nil {
		httpServer := &http.Server{
			Addr:    opts.HTTP.Addr,
			Handler: opts.HTTP.Mux,
		}
		runtime.SetHTTPServer(httpServer)
	}

	if opts.RPC != nil {
		rpcServer := &http.Server{
			Addr: opts.RPC.Addr,
		}
		runtime.SetRPCServer(rpcServer)
	}

	if opts.Health != nil {
		healthServer := &http.Server{
			Addr: opts.Health.Addr,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			}),
		}
		runtime.SetHealthServer(healthServer)
	}

	if opts.Metrics != nil {
		metricsServer := &http.Server{
			Addr: opts.Metrics.Addr,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("# Metrics endpoint\n"))
			}),
		}
		runtime.SetMetricsServer(metricsServer)
	}

	// Start runtime
	if err := runtime.Start(ctx); err != nil {
		return fmt.Errorf("runtime start failed: %w", err)
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Stop runtime
	if err := runtime.Stop(context.Background()); err != nil {
		return fmt.Errorf("runtime stop failed: %w", err)
	}

	return nil
}

// --- Health check aggregation ---

// HealthChecker defines the interface for health checks.
// Implementations should perform quick checks and honor context deadlines.
type HealthChecker interface {
	// Name returns the name of the health check.
	Name() string
	// Check performs the health check and returns an error if unhealthy.
	Check(ctx context.Context) error
}

// RegisterHealthChecker registers a global health checker.
//
// Parameters:
//   - checker: health checker implementation
//
// Concurrency:
//   - Safe for concurrent use
func RegisterHealthChecker(checker HealthChecker) {
	internal.RegisterHealthChecker(checker)
}

// CheckHealth runs all registered health checkers.
// Returns nil if all checks pass, otherwise returns the first error.
//
// Parameters:
//   - ctx: context with deadline for checks
//
// Returns:
//   - error: first error encountered, or nil if all pass
//
// Concurrency:
//   - Safe for concurrent use
func CheckHealth(ctx context.Context) error {
	return internal.CheckHealth(ctx)
}

// ClearHealthCheckers clears all registered health checkers (intended for testing).
//
// Concurrency:
//   - Safe for concurrent use
func ClearHealthCheckers() {
	internal.ClearHealthCheckers()
}
