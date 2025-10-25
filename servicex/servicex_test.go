// Package servicex provides a unified microservice initialization framework.
package servicex

import (
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/eggybyte-technology/egg/core/log"
	"github.com/eggybyte-technology/egg/servicex/internal"
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
