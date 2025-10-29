// Package servicex provides a unified microservice initialization framework.
package servicex

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"go.eggybyte.com/egg/configx"
	"go.eggybyte.com/egg/core/log"
	"go.eggybyte.com/egg/servicex/internal"
	"gorm.io/gorm"
)

// MockLogger is a test implementation of log.Logger
type MockLogger struct {
	debugs []string
	infos  []string
	warns  []string
	errors []string
}

func (m *MockLogger) With(kv ...any) log.Logger              { return m }
func (m *MockLogger) Debug(msg string, kv ...any)            { m.debugs = append(m.debugs, msg) }
func (m *MockLogger) Info(msg string, kv ...any)             { m.infos = append(m.infos, msg) }
func (m *MockLogger) Warn(msg string, kv ...any)             { m.warns = append(m.warns, msg) }
func (m *MockLogger) Error(err error, msg string, kv ...any) { m.errors = append(m.errors, msg) }

// setupTestPorts sets up random ports for testing to avoid conflicts
func setupTestPorts(t *testing.T) func() {
	t.Helper()

	// Save original values
	originalHTTP := os.Getenv("HTTP_PORT")
	originalHealth := os.Getenv("HEALTH_PORT")
	originalMetrics := os.Getenv("METRICS_PORT")

	// Set to "0" to allocate random ports
	os.Setenv("HTTP_PORT", "0")
	os.Setenv("HEALTH_PORT", "0")
	os.Setenv("METRICS_PORT", "0")

	// Return cleanup function
	return func() {
		if originalHTTP != "" {
			os.Setenv("HTTP_PORT", originalHTTP)
		} else {
			os.Unsetenv("HTTP_PORT")
		}
		if originalHealth != "" {
			os.Setenv("HEALTH_PORT", originalHealth)
		} else {
			os.Unsetenv("HEALTH_PORT")
		}
		if originalMetrics != "" {
			os.Setenv("METRICS_PORT", originalMetrics)
		} else {
			os.Unsetenv("METRICS_PORT")
		}
	}
}

func TestOptions(t *testing.T) {
	tests := []struct {
		name    string
		options Options
	}{
		{
			name: "valid options",
			options: Options{
				ServiceName: "test-service",
				Config:      &struct{}{},
			},
		},
		{
			name: "with database config",
			options: Options{
				ServiceName: "test",
				Database: &DatabaseConfig{
					Driver: "mysql",
					DSN:    "test-dsn",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.options.ServiceName == "" {
				t.Error("ServiceName should not be empty")
			}
		})
	}
}

func TestApp_Mux(t *testing.T) {
	mux := http.NewServeMux()
	app := &App{mux: mux}

	if app.Mux() != mux {
		t.Errorf("App.Mux() = %v, want %v", app.Mux(), mux)
	}
}

func TestApp_Logger(t *testing.T) {
	logger := &MockLogger{}
	app := &App{logger: logger}

	if app.Logger() != logger {
		t.Errorf("App.Logger() = %v, want %v", app.Logger(), logger)
	}
}

// TestApp_Provide tests the DI container Provide method.
func TestApp_Provide(t *testing.T) {
	app := &App{
		container: internal.NewContainer(),
	}

	// Provide a simple constructor
	err := app.Provide(func() string { return "test" })
	if err != nil {
		t.Fatalf("Provide failed: %v", err)
	}

	var result string
	err = app.Resolve(&result)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if result != "test" {
		t.Errorf("Expected 'test', got %q", result)
	}
}

func TestApp_Interceptors(t *testing.T) {
	// Note: We can't easily create mock connect.Interceptor instances for testing
	// This test verifies the method exists and returns the expected type
	app := &App{interceptors: []connect.Interceptor{}}

	result := app.Interceptors()
	if result == nil {
		t.Error("App.Interceptors() should not return nil")
	}
	if len(result) != 0 {
		t.Errorf("App.Interceptors() length = %d, want 0", len(result))
	}
}

func TestDatabaseConfig(t *testing.T) {
	config := DatabaseConfig{
		Driver:          "mysql",
		DSN:             "test-dsn",
		MaxIdleConns:    5,
		MaxOpenConns:    50,
		ConnMaxLifetime: 30 * time.Minute,
		PingTimeout:     5 * time.Second,
	}

	if config.Driver != "mysql" {
		t.Errorf("DatabaseConfig.Driver = %s, want mysql", config.Driver)
	}
	if config.DSN != "test-dsn" {
		t.Errorf("DatabaseConfig.DSN = %s, want test-dsn", config.DSN)
	}
	if config.MaxIdleConns != 5 {
		t.Errorf("DatabaseConfig.MaxIdleConns = %d, want 5", config.MaxIdleConns)
	}
	if config.MaxOpenConns != 50 {
		t.Errorf("DatabaseConfig.MaxOpenConns = %d, want 50", config.MaxOpenConns)
	}
	if config.ConnMaxLifetime != 30*time.Minute {
		t.Errorf("DatabaseConfig.ConnMaxLifetime = %v, want %v", config.ConnMaxLifetime, 30*time.Minute)
	}
	if config.PingTimeout != 5*time.Second {
		t.Errorf("DatabaseConfig.PingTimeout = %v, want %v", config.PingTimeout, 5*time.Second)
	}
}

func TestServiceRegistrar(t *testing.T) {
	var called bool
	var receivedApp *App

	registrar := func(app *App) error {
		called = true
		receivedApp = app
		return nil
	}

	testApp := &App{}
	err := registrar(testApp)

	if !called {
		t.Error("ServiceRegistrar was not called")
	}
	if receivedApp != testApp {
		t.Errorf("ServiceRegistrar received app = %v, want %v", receivedApp, testApp)
	}
	if err != nil {
		t.Errorf("ServiceRegistrar returned error = %v, want nil", err)
	}
}

func TestDatabaseMigrator(t *testing.T) {
	var called bool
	var receivedDB *gorm.DB

	migrator := func(db *gorm.DB) error {
		called = true
		receivedDB = db
		return nil
	}

	testDB := &gorm.DB{}
	err := migrator(testDB)

	if !called {
		t.Error("DatabaseMigrator was not called")
	}
	if receivedDB != testDB {
		t.Errorf("DatabaseMigrator received db = %v, want %v", receivedDB, testDB)
	}
	if err != nil {
		t.Errorf("DatabaseMigrator returned error = %v, want nil", err)
	}
}

// ========================================
// Integration Tests
// ========================================

// TestServiceLifecycle tests the full service lifecycle.
func TestServiceLifecycle(t *testing.T) {
	cleanup := setupTestPorts(t)
	defer cleanup()

	tests := []struct {
		name        string
		opts        []Option
		wantErr     bool
		errContains string
	}{
		{
			name: "minimal service with defaults",
			opts: []Option{
				WithService("test-service", "1.0.0"),
				WithRegister(func(app *App) error {
					app.Mux().HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte("OK"))
					})
					return nil
				}),
			},
			wantErr: false,
		},
		{
			name: "service with config",
			opts: []Option{
				WithService("test-service", "1.0.0"),
				WithConfig(&configx.BaseConfig{}),
				WithRegister(func(app *App) error {
					return nil
				}),
			},
			wantErr: false,
		},
		{
			name: "service with metrics enabled",
			opts: []Option{
				WithService("test-service", "1.0.0"),
				WithMetrics(true),
				WithRegister(func(app *App) error {
					return nil
				}),
			},
			wantErr: false,
		},
		{
			name: "service with metrics config",
			opts: []Option{
				WithService("test-service", "1.0.0"),
				WithMetricsConfig(true, true, false, false),
				WithRegister(func(app *App) error {
					return nil
				}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create cancellable context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// Run service in goroutine
			errChan := make(chan error, 1)
			go func() {
				errChan <- Run(ctx, tt.opts...)
			}()

			// Wait for service to start
			time.Sleep(200 * time.Millisecond)

			// Cancel context to trigger shutdown
			cancel()

			// Wait for service to stop
			select {
			case err := <-errChan:
				if tt.wantErr && err == nil {
					t.Errorf("Expected error but got nil")
				}
				if !tt.wantErr && err != nil && err != context.Canceled {
					t.Errorf("Unexpected error: %v", err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("Service did not shut down in time")
			}
		})
	}
}

// TestServiceWithCustomTimeout tests service with custom timeout settings.
func TestServiceWithCustomTimeout(t *testing.T) {
	cleanup := setupTestPorts(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- Run(ctx,
			WithService("test-service", "1.0.0"),
			WithTimeout(5000), // 5 seconds
			WithSlowRequestThreshold(100),
			WithRegister(func(app *App) error {
				return nil
			}),
		)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			t.Errorf("Unexpected error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Service did not shut down in time")
	}
}

// TestServiceWithShutdownHook tests shutdown hook execution.
func TestServiceWithShutdownHook(t *testing.T) {
	cleanup := setupTestPorts(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	hookCalled := false

	errChan := make(chan error, 1)
	go func() {
		errChan <- Run(ctx,
			WithService("test-service", "1.0.0"),
			WithRegister(func(app *App) error {
				app.AddShutdownHook(func(ctx context.Context) error {
					hookCalled = true
					return nil
				})
				return nil
			}),
		)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case <-errChan:
		// Wait a bit for shutdown to complete
		time.Sleep(100 * time.Millisecond)
		if !hookCalled {
			t.Error("Shutdown hook was not called")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Service did not shut down in time")
	}
}

// TestServiceRegistrationError tests error handling during service registration.
func TestServiceRegistrationError(t *testing.T) {
	cleanup := setupTestPorts(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	expectedErr := fmt.Errorf("registration error")

	errChan := make(chan error, 1)
	go func() {
		errChan <- Run(ctx,
			WithService("test-service", "1.0.0"),
			WithRegister(func(app *App) error {
				return expectedErr
			}),
		)
	}()

	select {
	case err := <-errChan:
		if err == nil {
			t.Error("Expected error but got nil")
		}
		// Error should contain "registration" or similar
	case <-time.After(3 * time.Second):
		t.Fatal("Service did not return error in time")
	}
}

// TestServiceWithShutdownTimeout tests graceful shutdown timeout.
func TestServiceWithShutdownTimeout(t *testing.T) {
	cleanup := setupTestPorts(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- Run(ctx,
			WithService("test-service", "1.0.0"),
			WithShutdownTimeout(1*time.Second),
			WithRegister(func(app *App) error {
				return nil
			}),
		)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case <-errChan:
		// Shutdown completed
	case <-time.After(5 * time.Second):
		t.Fatal("Service did not shut down in time")
	}
}

// TestServiceEnvironmentVariables tests service configuration via environment.
func TestServiceEnvironmentVariables(t *testing.T) {
	// Save original env vars
	originalHTTP := os.Getenv("HTTP_PORT")
	originalHealth := os.Getenv("HEALTH_PORT")
	originalMetrics := os.Getenv("METRICS_PORT")
	defer func() {
		os.Setenv("HTTP_PORT", originalHTTP)
		os.Setenv("HEALTH_PORT", originalHealth)
		os.Setenv("METRICS_PORT", originalMetrics)
	}()

	// Set test env vars (use high ports to avoid conflicts)
	os.Setenv("HTTP_PORT", "18080")
	os.Setenv("HEALTH_PORT", "18081")
	os.Setenv("METRICS_PORT", "19091")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- Run(ctx,
			WithService("test-service", "1.0.0"),
			WithConfig(&configx.BaseConfig{}),
			WithRegister(func(app *App) error {
				return nil
			}),
		)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			t.Errorf("Unexpected error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Service did not shut down in time")
	}
}

// TestServiceAppMethods tests App methods during registration.
func TestServiceAppMethods(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var capturedApp *App

	errChan := make(chan error, 1)
	go func() {
		errChan <- Run(ctx,
			WithService("test-service", "1.0.0"),
			WithRegister(func(app *App) error {
				capturedApp = app

				// Test Mux
				if app.Mux() == nil {
					return fmt.Errorf("Mux() returned nil")
				}

				// Test Logger
				if app.Logger() == nil {
					return fmt.Errorf("Logger() returned nil")
				}

				// Test Interceptors
				if app.Interceptors() == nil {
					return fmt.Errorf("Interceptors() returned nil")
				}

				// Test DB (should be nil without database config)
				if app.DB() != nil {
					return fmt.Errorf("DB() should be nil without database config")
				}

				return nil
			}),
		)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			t.Errorf("Unexpected error: %v", err)
		}
		if capturedApp == nil {
			t.Error("App was not captured during registration")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Service did not complete in time")
	}
}
