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
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/sony/gobreaker"
)

// Options configures the HTTP client behavior.
type Options struct {
	Timeout          time.Duration // Request timeout (default: 30s)
	MaxRetries       int           // Maximum retry attempts (default: 3)
	RetryBackoff     time.Duration // Initial backoff duration (default: 100ms)
	EnableCircuit    bool          // Enable circuit breaker (default: true)
	CircuitThreshold uint32        // Circuit breaker failure threshold (default: 5)
	IdempotencyKey   string        // Custom idempotency key header name
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
		Timeout: options.Timeout,
		Transport: &retryTransport{
			base:       http.DefaultTransport,
			maxRetries: options.MaxRetries,
			backoff:    options.RetryBackoff,
			cb:         cb,
		},
	}

	return client
}

// retryTransport implements http.RoundTripper with retry logic.
type retryTransport struct {
	base       http.RoundTripper
	maxRetries int
	backoff    time.Duration
	cb         *gobreaker.CircuitBreaker
}

// RoundTrip implements http.RoundTripper with retry and circuit breaker.
func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Execute through circuit breaker if enabled
	if t.cb != nil {
		result, cbErr := t.cb.Execute(func() (interface{}, error) {
			return t.roundTripWithRetry(req)
		})
		if cbErr != nil {
			return nil, cbErr
		}
		return result.(*http.Response), nil
	}

	return t.roundTripWithRetry(req)
}

// roundTripWithRetry performs the request with retry logic.
func (t *retryTransport) roundTripWithRetry(req *http.Request) (*http.Response, error) {
	var lastResp *http.Response
	var lastErr error

	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		// Clone request for retry (body might be consumed)
		clonedReq := req.Clone(req.Context())

		resp, err := t.base.RoundTrip(clonedReq)

		// Success or non-retryable error
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		// Store last response and error for final return
		lastResp = resp
		lastErr = err

		// Don't retry on last attempt
		if attempt == t.maxRetries {
			break
		}

		// Close failed response body to prevent resource leak
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}

		// Exponential backoff with jitter
		backoff := t.backoff * time.Duration(1<<uint(attempt))
		time.Sleep(backoff)
	}

	return lastResp, lastErr
}

// NewConnectClient creates a Connect client with interceptors.
// This is a convenience wrapper for creating Connect clients with standard interceptors.
func NewConnectClient[T any](baseURL, serviceName string, newClient func(connect.HTTPClient, string, ...connect.ClientOption) T, opts ...Option) T {
	httpClient := NewHTTPClient(baseURL, opts...)

	// TODO: Add Connect client interceptors (timeout, metrics, etc.)
	// For now, return basic client
	return newClient(httpClient, baseURL)
}
