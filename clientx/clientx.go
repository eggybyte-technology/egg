// Package clientx provides Connect client factory with retry, circuit breaker, and timeouts.
//
// Overview:
//   - Responsibility: Create Connect HTTP clients with production-ready interceptors
//   - Key Types: Options for client configuration, interceptors for resilience
//   - Concurrency Model: Clients are safe for concurrent use
//   - Error Semantics: Retry only on transient/idempotent errors
//   - Performance Notes: Circuit breaker prevents cascade failures
//
// Usage:
//
//	client := clientx.NewHTTPClient("https://api.example.com",
//	  clientx.WithTimeout(5*time.Second),
//	  clientx.WithRetry(3),
//	)
package clientx

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/sony/gobreaker"
	"go.eggybyte.com/egg/clientx/internal"
)

// Options configures the HTTP client behavior.
type Options struct {
	Timeout            time.Duration // Request timeout (default: 30s)
	MaxRetries         int           // Maximum retry attempts (default: 3)
	RetryBackoff       time.Duration // Initial backoff duration (default: 100ms)
	EnableCircuit      bool          // Enable circuit breaker (default: true)
	CircuitThreshold   uint32        // Circuit breaker failure threshold (default: 5)
	IdempotencyKey     string        // Custom idempotency key header name
	InternalToken      string        // Internal service token
	InternalTokenHeader string       // Header name for internal token
}

// Option is a functional option for configuring the client.
type Option func(*Options)

// WithTimeout sets the client timeout.
func WithTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.Timeout = d
	}
}

// WithRetry sets the maximum retry attempts.
func WithRetry(maxRetries int) Option {
	return func(o *Options) {
		o.MaxRetries = maxRetries
	}
}

// WithCircuitBreaker enables or disables the circuit breaker.
func WithCircuitBreaker(enabled bool) Option {
	return func(o *Options) {
		o.EnableCircuit = enabled
	}
}

// WithIdempotencyKey sets the idempotency key header name.
func WithIdempotencyKey(key string) Option {
	return func(o *Options) {
		o.IdempotencyKey = key
	}
}

// WithInternalToken sets the internal token for service-to-service authentication.
// The token is automatically added to all outgoing requests.
func WithInternalToken(token string) Option {
	return func(o *Options) {
		o.InternalToken = token
		if o.InternalTokenHeader == "" {
			o.InternalTokenHeader = "X-Internal-Token"
		}
	}
}

// WithInternalTokenHeader sets the header name for internal token.
func WithInternalTokenHeader(header string) Option {
	return func(o *Options) {
		o.InternalTokenHeader = header
	}
}

// NewHTTPClient creates a new HTTP client with Connect interceptors.
func NewHTTPClient(baseURL string, opts ...Option) *http.Client {
	options := Options{
		Timeout:          30 * time.Second,
		MaxRetries:       3,
		RetryBackoff:     100 * time.Millisecond,
		EnableCircuit:    true,
		CircuitThreshold: 5,
		IdempotencyKey:   "X-Idempotency-Key",
	}

	for _, opt := range opts {
		opt(&options)
	}

	// Create circuit breaker if enabled
	var cb *gobreaker.CircuitBreaker
	if options.EnableCircuit {
		cb = gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:        "connect-client",
			MaxRequests: options.CircuitThreshold,
			Timeout:     60 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures > options.CircuitThreshold
			},
		})
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout:   options.Timeout,
		Transport: internal.NewRetryTransport(http.DefaultTransport, options.MaxRetries, options.RetryBackoff, cb),
	}

	return client
}

// NewConnectClient creates a Connect client with interceptors.
// This is a convenience wrapper for creating Connect clients with standard interceptors.
func NewConnectClient[T any](baseURL, serviceName string, newClient func(connect.HTTPClient, string, ...connect.ClientOption) T, opts ...Option) T {
	// Apply options
	options := Options{
		Timeout:            30 * time.Second,
		MaxRetries:         3,
		RetryBackoff:       100 * time.Millisecond,
		EnableCircuit:      true,
		CircuitThreshold:   5,
		IdempotencyKey:     "X-Idempotency-Key",
		InternalTokenHeader: "X-Internal-Token",
	}
	for _, opt := range opts {
		opt(&options)
	}

	httpClient := NewHTTPClient(baseURL, opts...)

	// Build client options
	var clientOpts []connect.ClientOption

	// Add internal token interceptor if token is provided
	if options.InternalToken != "" {
		clientOpts = append(clientOpts, connect.WithInterceptors(
			internalTokenInterceptor(options.InternalToken, options.InternalTokenHeader),
		))
	}

	return newClient(httpClient, baseURL, clientOpts...)
}

// internalTokenInterceptor creates a client-side interceptor that adds internal token to requests.
func internalTokenInterceptor(token, headerName string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if token != "" && req.Header() != nil {
				req.Header().Set(headerName, token)
			}
			return next(ctx, req)
		}
	}
}
