// Package connectx provides Connect interceptors and identity injection.
//
// Overview:
//   - Responsibility: Unified interceptor stack for Connect services
//   - Key Types: HeaderMapping for request headers, Options for configuration
//   - Concurrency Model: Interceptors are safe for concurrent use
//   - Error Semantics: Error mapping from core/errors to Connect codes
//   - Performance Notes: Supports configurable payload logging and slow request detection
//
// Usage:
//
//	interceptors := connectx.DefaultInterceptors(connectx.Options{
//	  Logger: logger,
//	  Otel:   otelProvider,
//	})
//	handler := connect.WithInterceptors(interceptors...)
package connectx

import (
	"net/http"

	"connectrpc.com/connect"
	"go.eggybyte.com/egg/connectx/internal"
	"go.eggybyte.com/egg/core/log"
	"go.eggybyte.com/egg/obsx"
)

// HeaderMapping defines how HTTP headers map to identity and metadata fields.
// Default values are set for Higress-style headers.
type HeaderMapping struct {
	RequestID     string // "X-Request-Id"
	InternalToken string // "X-Internal-Token"
	UserID        string // "X-User-Id"
	UserName      string // "X-User-Name"
	Roles         string // "X-User-Roles"
	RealIP        string // "X-Real-IP"
	ForwardedFor  string // "X-Forwarded-For"
	UserAgent     string // "User-Agent"
}

// DefaultHeaderMapping returns the default header mapping for Higress.
func DefaultHeaderMapping() HeaderMapping {
	return HeaderMapping{
		RequestID:     "X-Request-Id",
		InternalToken: "X-Internal-Token",
		UserID:        "X-User-Id",
		UserName:      "X-User-Name",
		Roles:         "X-User-Roles",
		RealIP:        "X-Real-IP",
		ForwardedFor:  "X-Forwarded-For",
		UserAgent:     "User-Agent",
	}
}

// Options holds configuration for Connect interceptors.
type Options struct {
	Logger            log.Logger     // Logger for interceptor operations
	Otel              *obsx.Provider // OpenTelemetry provider (nil disables tracing)
	Headers           HeaderMapping  // Header mapping configuration
	WithRequestBody   bool           // Log request body (default: false for production)
	WithResponseBody  bool           // Log response body (default: false for production)
	SlowRequestMillis int64          // Slow request threshold in milliseconds
	PayloadAccounting bool           // Track inbound/outbound payload sizes
	DefaultTimeoutMs  int64          // Default RPC timeout in milliseconds (0 = no timeout)
	EnableTimeout     bool           // Enable timeout interceptor (default: true)
}

// DefaultInterceptors returns a set of interceptors with the given options.
// The interceptors are ordered for optimal performance and functionality:
// 1. Recovery (panic handling)
// 2. Timeout (service-level + request header override)
// 3. Identity injection (extract headers to context)
// 4. Metrics collection (RPC request metrics)
// 5. Error mapping (core/errors to Connect codes)
// 6. Logging (structured request/response logging)
func DefaultInterceptors(opts Options) []connect.Interceptor {
	// Set default header mapping if not provided
	if opts.Headers.RequestID == "" {
		opts.Headers = DefaultHeaderMapping()
	}

	// Set default slow request threshold
	if opts.SlowRequestMillis == 0 {
		opts.SlowRequestMillis = 1000 // 1 second
	}

	// Set default timeout if not specified
	if opts.DefaultTimeoutMs == 0 {
		opts.DefaultTimeoutMs = 30000 // 30 seconds default
	}

	var interceptors []connect.Interceptor

	// Add recovery interceptor
	if opts.Logger != nil {
		interceptors = append(interceptors, connect.UnaryInterceptorFunc(internal.RecoveryInterceptor(opts.Logger)))
	}

	// Add timeout interceptor (before identity/logging to ensure proper deadline propagation)
	if opts.EnableTimeout || opts.DefaultTimeoutMs > 0 {
		interceptors = append(interceptors, connect.UnaryInterceptorFunc(internal.TimeoutInterceptor(opts.DefaultTimeoutMs)))
	}

	// Add identity injection interceptor
	interceptors = append(interceptors, connect.UnaryInterceptorFunc(internal.IdentityInterceptor(internal.HeaderMapping{
		RequestID:     opts.Headers.RequestID,
		InternalToken: opts.Headers.InternalToken,
		UserID:        opts.Headers.UserID,
		UserName:      opts.Headers.UserName,
		Roles:         opts.Headers.Roles,
		RealIP:        opts.Headers.RealIP,
		ForwardedFor:  opts.Headers.ForwardedFor,
		UserAgent:     opts.Headers.UserAgent,
	})))

	// Add metrics interceptor (if OTEL provider is available)
	if opts.Otel != nil {
		if collector, err := internal.NewMetricsCollector(opts.Otel); err == nil {
			interceptors = append(interceptors, connect.UnaryInterceptorFunc(internal.MetricsInterceptor(collector)))
		}
		// Silently skip metrics if initialization fails
	}

	// Add error mapping interceptor
	interceptors = append(interceptors, connect.UnaryInterceptorFunc(internal.ErrorMappingInterceptor()))

	// Add logging interceptor
	if opts.Logger != nil {
		interceptors = append(interceptors, connect.UnaryInterceptorFunc(internal.LoggingInterceptor(opts.Logger, internal.LoggingOptions{
			WithRequestBody:   opts.WithRequestBody,
			WithResponseBody:  opts.WithResponseBody,
			SlowRequestMillis: opts.SlowRequestMillis,
			PayloadAccounting: opts.PayloadAccounting,
		})))
	}

	return interceptors
}

// Bind is a utility function to bind Connect handlers to HTTP mux.
// This provides a consistent way to mount Connect services.
func Bind(mux *http.ServeMux, path string, handler http.Handler) {
	mux.Handle(path, handler)
}
