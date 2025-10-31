// Package internal provides tests for runtimex internal implementation.
package internal

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"go.eggybyte.com/egg/core/log"
)

func TestRegisterHealthChecker(t *testing.T) {
	ClearHealthCheckers()

	checker := &mockHealthChecker{name: "test-checker"}
	RegisterHealthChecker(checker)

	ctx := context.Background()
	err := CheckHealth(ctx)
	if err != nil {
		t.Errorf("CheckHealth() error = %v, want nil", err)
	}
}

func TestCheckHealth_NoCheckers(t *testing.T) {
	ClearHealthCheckers()

	ctx := context.Background()
	err := CheckHealth(ctx)
	if err != nil {
		t.Errorf("CheckHealth() error = %v, want nil (no checkers)", err)
	}
}

func TestCheckHealth_SingleChecker(t *testing.T) {
	ClearHealthCheckers()

	checker := &mockHealthChecker{name: "test-checker"}
	RegisterHealthChecker(checker)

	ctx := context.Background()
	err := CheckHealth(ctx)
	if err != nil {
		t.Errorf("CheckHealth() error = %v, want nil", err)
	}
	if !checker.checked {
		t.Error("Checker should have been checked")
	}
}

func TestCheckHealth_MultipleCheckers(t *testing.T) {
	ClearHealthCheckers()

	checker1 := &mockHealthChecker{name: "checker1"}
	checker2 := &mockHealthChecker{name: "checker2"}
	checker3 := &mockHealthChecker{name: "checker3"}

	RegisterHealthChecker(checker1)
	RegisterHealthChecker(checker2)
	RegisterHealthChecker(checker3)

	ctx := context.Background()
	err := CheckHealth(ctx)
	if err != nil {
		t.Errorf("CheckHealth() error = %v, want nil", err)
	}

	if !checker1.checked || !checker2.checked || !checker3.checked {
		t.Error("All checkers should have been checked")
	}
}

func TestCheckHealth_Failure(t *testing.T) {
	ClearHealthCheckers()

	checker1 := &mockHealthChecker{name: "checker1"}
	checker2 := &mockHealthChecker{name: "checker2", shouldFail: true, failErr: errors.New("health check failed")}
	checker3 := &mockHealthChecker{name: "checker3"}

	RegisterHealthChecker(checker1)
	RegisterHealthChecker(checker2)
	RegisterHealthChecker(checker3)

	ctx := context.Background()
	err := CheckHealth(ctx)
	if err == nil {
		t.Fatal("CheckHealth() should return error when checker fails")
	}

	if err.Error() != "health check failed" {
		t.Errorf("CheckHealth() error = %v, want 'health check failed'", err)
	}

	// First checker should have been checked
	if !checker1.checked {
		t.Error("First checker should have been checked")
	}

	// Second checker should have been checked and failed
	if !checker2.checked {
		t.Error("Second checker should have been checked")
	}

	// Third checker should not have been checked (fail fast)
	if checker3.checked {
		t.Error("Third checker should not have been checked (fail fast)")
	}
}

func TestCheckHealth_ContextCancellation(t *testing.T) {
	ClearHealthCheckers()

	checker := &mockHealthChecker{
		name:      "slow-checker",
		delay:     100 * time.Millisecond,
		shouldFail: true,
		failErr:   context.DeadlineExceeded,
	}

	RegisterHealthChecker(checker)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := CheckHealth(ctx)
	if err == nil {
		t.Fatal("CheckHealth() should return error when context is cancelled")
	}
}

func TestCheckHealth_Concurrency(t *testing.T) {
	ClearHealthCheckers()

	const numCheckers = 10
	for i := 0; i < numCheckers; i++ {
		checker := &mockHealthChecker{name: "checker"}
		RegisterHealthChecker(checker)
	}

	ctx := context.Background()
	err := CheckHealth(ctx)
	if err != nil {
		t.Errorf("CheckHealth() error = %v, want nil", err)
	}
}

func TestClearHealthCheckers(t *testing.T) {
	ClearHealthCheckers()

	checker := &mockHealthChecker{name: "test-checker"}
	RegisterHealthChecker(checker)

	ClearHealthCheckers()

	ctx := context.Background()
	err := CheckHealth(ctx)
	if err != nil {
		t.Errorf("CheckHealth() error = %v, want nil (checkers cleared)", err)
	}

	if checker.checked {
		t.Error("Checker should not have been checked after clearing")
	}
}

// mockHealthChecker is a test implementation of HealthChecker.
type mockHealthChecker struct {
	name       string
	checked    bool
	shouldFail bool
	failErr    error
	delay      time.Duration
}

func (m *mockHealthChecker) Name() string {
	return m.name
}

func (m *mockHealthChecker) Check(ctx context.Context) error {
	m.checked = true

	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if m.shouldFail {
		return m.failErr
	}

	return nil
}

func TestNewRuntime(t *testing.T) {
	logger := &mockLogger{}
	services := []Service{}
	timeout := 30 * time.Second

	runtime := NewRuntime(logger, services, timeout)

	if runtime == nil {
		t.Fatal("NewRuntime() should return non-nil runtime")
	}
	if runtime.logger != logger {
		t.Error("Runtime logger should be set")
	}
	if len(runtime.services) != 0 {
		t.Error("Runtime services should be empty")
	}
	if runtime.shutdownTimeout != timeout {
		t.Errorf("ShutdownTimeout = %v, want %v", runtime.shutdownTimeout, timeout)
	}
}

func TestRuntime_SetHTTPServer(t *testing.T) {
	runtime := NewRuntime(&mockLogger{}, nil, 30*time.Second)
	server := &http.Server{Addr: ":8080"}

	runtime.SetHTTPServer(server)

	if runtime.httpServer != server {
		t.Error("HTTP server should be set")
	}
}

func TestRuntime_SetRPCServer(t *testing.T) {
	runtime := NewRuntime(&mockLogger{}, nil, 30*time.Second)
	server := &http.Server{Addr: ":8081"}

	runtime.SetRPCServer(server)

	if runtime.rpcServer != server {
		t.Error("RPC server should be set")
	}
}

func TestRuntime_SetHealthServer(t *testing.T) {
	runtime := NewRuntime(&mockLogger{}, nil, 30*time.Second)
	server := &http.Server{Addr: ":8082"}

	runtime.SetHealthServer(server)

	if runtime.healthServer != server {
		t.Error("Health server should be set")
	}
}

func TestRuntime_SetMetricsServer(t *testing.T) {
	runtime := NewRuntime(&mockLogger{}, nil, 30*time.Second)
	server := &http.Server{Addr: ":8083"}

	runtime.SetMetricsServer(server)

	if runtime.metricsServer != server {
		t.Error("Metrics server should be set")
	}
}

// mockLogger is a test implementation of log.Logger.
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, kv ...interface{}) {}
func (m *mockLogger) Info(msg string, kv ...interface{})  {}
func (m *mockLogger) Warn(msg string, kv ...interface{})  {}
func (m *mockLogger) Error(err error, msg string, kv ...interface{}) {}
func (m *mockLogger) With(kv ...interface{}) log.Logger { return m }

