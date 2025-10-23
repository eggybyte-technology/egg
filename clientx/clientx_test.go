package clientx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient("https://api.example.com")
	if client == nil {
		t.Fatal("NewHTTPClient should return non-nil client")
	}

	if client.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", client.Timeout)
	}
}

func TestWithTimeout(t *testing.T) {
	client := NewHTTPClient("https://api.example.com",
		WithTimeout(5*time.Second),
	)

	if client.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", client.Timeout)
	}
}

func TestWithRetry(t *testing.T) {
	client := NewHTTPClient("https://api.example.com",
		WithRetry(5),
	)

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	// Verify transport is retryTransport
	rt, ok := client.Transport.(*retryTransport)
	if !ok {
		t.Fatal("Expected retryTransport")
	}

	if rt.maxRetries != 5 {
		t.Errorf("Expected maxRetries 5, got %d", rt.maxRetries)
	}
}

func TestRetryOn5xx(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL,
		WithRetry(3),
		WithTimeout(5*time.Second),
		WithCircuitBreaker(false), // Disable circuit breaker for test
	)

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestNoRetryOn4xx(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL,
		WithRetry(3),
		WithTimeout(5*time.Second),
		WithCircuitBreaker(false),
	)

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	// Should not retry on 4xx
	if attempts != 1 {
		t.Errorf("Expected 1 attempt (no retry on 4xx), got %d", attempts)
	}
}

func TestCircuitBreakerEnabled(t *testing.T) {
	client := NewHTTPClient("https://api.example.com",
		WithCircuitBreaker(true),
	)

	rt, ok := client.Transport.(*retryTransport)
	if !ok {
		t.Fatal("Expected retryTransport")
	}

	if rt.cb == nil {
		t.Error("Circuit breaker should be enabled")
	}
}

func TestCircuitBreakerDisabled(t *testing.T) {
	client := NewHTTPClient("https://api.example.com",
		WithCircuitBreaker(false),
	)

	rt, ok := client.Transport.(*retryTransport)
	if !ok {
		t.Fatal("Expected retryTransport")
	}

	if rt.cb != nil {
		t.Error("Circuit breaker should be disabled")
	}
}

func TestWithIdempotencyKey(t *testing.T) {
	client := NewHTTPClient("https://api.example.com",
		WithIdempotencyKey("X-Request-ID"),
	)

	if client == nil {
		t.Fatal("Client should not be nil")
	}
}

func BenchmarkRetryTransport(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL,
		WithRetry(3),
		WithCircuitBreaker(false),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
		resp, err := client.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
