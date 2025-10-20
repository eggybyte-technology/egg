// Package runtimex provides tests for runtime lifecycle management.
package runtimex

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/eggybyte-technology/egg/core/log"
)

// testLogger is a test logger implementation.
type testLogger struct {
	logs []string
}

func (l *testLogger) With(kv ...any) log.Logger              { return l }
func (l *testLogger) Debug(msg string, kv ...any)            { l.logs = append(l.logs, "DEBUG: "+msg) }
func (l *testLogger) Info(msg string, kv ...any)             { l.logs = append(l.logs, "INFO: "+msg) }
func (l *testLogger) Warn(msg string, kv ...any)             { l.logs = append(l.logs, "WARN: "+msg) }
func (l *testLogger) Error(err error, msg string, kv ...any) { l.logs = append(l.logs, "ERROR: "+msg) }

// mockService is a mock implementation of the Service interface.
type mockService struct {
	startCalled bool
	stopCalled  bool
	startErr    error
	stopErr     error
}

func (m *mockService) Start(ctx context.Context) error {
	m.startCalled = true
	return m.startErr
}

func (m *mockService) Stop(ctx context.Context) error {
	m.stopCalled = true
	return m.stopErr
}

func TestOptions(t *testing.T) {
	logger := &testLogger{}
	mux := http.NewServeMux()

	opts := Options{
		Logger: logger,
		HTTP: &HTTPOptions{
			Addr: ":8080",
			H2C:  true,
			Mux:  mux,
		},
		Health: &Endpoint{
			Addr: ":8081",
		},
		Metrics: &Endpoint{
			Addr: ":9091",
		},
		ShutdownTimeout: 15 * time.Second,
	}

	if opts.Logger == nil {
		t.Error("Logger should not be nil")
	}

	if opts.HTTP == nil {
		t.Error("HTTP options should not be nil")
	}

	if opts.HTTP.Addr != ":8080" {
		t.Errorf("HTTP address = %v, want :8080", opts.HTTP.Addr)
	}

	if !opts.HTTP.H2C {
		t.Error("H2C should be enabled")
	}

	if opts.Health == nil {
		t.Error("Health endpoint should not be nil")
	}

	if opts.Metrics == nil {
		t.Error("Metrics endpoint should not be nil")
	}

	if opts.ShutdownTimeout != 15*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 15s", opts.ShutdownTimeout)
	}
}

func TestService(t *testing.T) {
	service := &mockService{}

	ctx := context.Background()

	// Test Start
	if err := service.Start(ctx); err != nil {
		t.Errorf("Start() error = %v", err)
	}

	if !service.startCalled {
		t.Error("Start should have been called")
	}

	// Test Stop
	if err := service.Stop(ctx); err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	if !service.stopCalled {
		t.Error("Stop should have been called")
	}
}

func TestEndpoint(t *testing.T) {
	tests := []struct {
		name string
		addr string
	}{
		{
			name: "port only",
			addr: ":8080",
		},
		{
			name: "host and port",
			addr: "localhost:8080",
		},
		{
			name: "IP and port",
			addr: "127.0.0.1:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := &Endpoint{
				Addr: tt.addr,
			}

			if endpoint.Addr != tt.addr {
				t.Errorf("Addr = %v, want %v", endpoint.Addr, tt.addr)
			}
		})
	}
}

func TestHTTPOptions(t *testing.T) {
	mux := http.NewServeMux()

	opts := &HTTPOptions{
		Addr: ":8080",
		H2C:  true,
		Mux:  mux,
	}

	if opts.Addr != ":8080" {
		t.Errorf("Addr = %v, want :8080", opts.Addr)
	}

	if !opts.H2C {
		t.Error("H2C should be enabled")
	}

	if opts.Mux == nil {
		t.Error("Mux should not be nil")
	}
}

func TestRPCOptions(t *testing.T) {
	opts := &RPCOptions{
		Addr: ":9090",
	}

	if opts.Addr != ":9090" {
		t.Errorf("Addr = %v, want :9090", opts.Addr)
	}
}

func TestRun_MissingLogger(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := Run(ctx, nil, Options{
		HTTP: &HTTPOptions{
			Addr: ":8080",
			Mux:  http.NewServeMux(),
		},
	})

	if err == nil {
		t.Error("Expected error for missing logger")
	}
}

func TestRun_WithShutdown(t *testing.T) {
	logger := &testLogger{}
	mux := http.NewServeMux()

	// Add a simple handler
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	opts := Options{
		Logger: logger,
		HTTP: &HTTPOptions{
			Addr: ":18080", // Use a different port to avoid conflicts
			H2C:  true,
			Mux:  mux,
		},
		ShutdownTimeout: 1 * time.Second,
	}

	// Run in a goroutine as it blocks
	errChan := make(chan error, 1)
	go func() {
		errChan <- Run(ctx, nil, opts)
	}()

	// Wait for context to be cancelled
	<-ctx.Done()

	// Wait for Run to complete
	select {
	case err := <-errChan:
		if err != nil {
			t.Logf("Run() completed with expected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Run() did not complete in time")
	}
}

func TestRun_WithServices(t *testing.T) {
	logger := &testLogger{}
	service := &mockService{}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	opts := Options{
		Logger: logger,
		HTTP: &HTTPOptions{
			Addr: ":18081", // Use a different port
			Mux:  http.NewServeMux(),
		},
		ShutdownTimeout: 1 * time.Second,
	}

	// Run in a goroutine
	go func() {
		Run(ctx, []Service{service}, opts)
	}()

	// Wait for context to be cancelled
	<-ctx.Done()

	// Give it some time to stop
	time.Sleep(200 * time.Millisecond)

	if !service.startCalled {
		t.Error("Service Start should have been called")
	}

	if !service.stopCalled {
		t.Error("Service Stop should have been called")
	}
}
