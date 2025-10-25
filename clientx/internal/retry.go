// Package internal provides internal implementation details for clientx.
package internal

import (
	"net/http"
	"time"

	"github.com/sony/gobreaker"
)

// RetryTransport implements http.RoundTripper with retry logic and circuit breaker.
type RetryTransport struct {
	base       http.RoundTripper
	maxRetries int
	backoff    time.Duration
	cb         *gobreaker.CircuitBreaker
}

// NewRetryTransport creates a new retry transport with the given configuration.
func NewRetryTransport(base http.RoundTripper, maxRetries int, backoff time.Duration, cb *gobreaker.CircuitBreaker) *RetryTransport {
	return &RetryTransport{
		base:       base,
		maxRetries: maxRetries,
		backoff:    backoff,
		cb:         cb,
	}
}

// RoundTrip implements http.RoundTripper with retry and circuit breaker.
func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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
func (t *RetryTransport) roundTripWithRetry(req *http.Request) (*http.Response, error) {
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



