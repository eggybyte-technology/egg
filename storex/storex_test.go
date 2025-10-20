// Package storex provides tests for storage interfaces and health check registry.
package storex

import (
	"context"
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

// mockStore is a mock implementation of the Store interface.
type mockStore struct {
	pingErr  error
	closeErr error
}

func (m *mockStore) Ping(ctx context.Context) error {
	return m.pingErr
}

func (m *mockStore) Close() error {
	return m.closeErr
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	// Test empty registry
	if len(registry.List()) != 0 {
		t.Errorf("Expected empty registry, got %d stores", len(registry.List()))
	}

	// Test Ping on empty registry
	ctx := context.Background()
	if err := registry.Ping(ctx); err != nil {
		t.Errorf("Ping() on empty registry error = %v", err)
	}

	// Test Register
	store1 := &mockStore{}
	if err := registry.Register("store1", store1); err != nil {
		t.Errorf("Register() error = %v", err)
	}

	if len(registry.List()) != 1 {
		t.Errorf("Expected 1 store, got %d", len(registry.List()))
	}

	// Test duplicate registration
	if err := registry.Register("store1", store1); err == nil {
		t.Error("Expected error for duplicate registration")
	}

	// Test Get
	retrievedStore, exists := registry.Get("store1")
	if !exists {
		t.Error("Expected store to exist")
	}
	if retrievedStore != store1 {
		t.Error("Retrieved store doesn't match")
	}

	// Test non-existent store
	_, exists = registry.Get("nonexistent")
	if exists {
		t.Error("Expected store to not exist")
	}

	// Test Ping with healthy store
	if err := registry.Ping(ctx); err != nil {
		t.Errorf("Ping() with healthy store error = %v", err)
	}

	// Test Ping with unhealthy store
	unhealthyStore := &mockStore{pingErr: context.DeadlineExceeded}
	if err := registry.Register("unhealthy", unhealthyStore); err != nil {
		t.Errorf("Register() error = %v", err)
	}

	if err := registry.Ping(ctx); err == nil {
		t.Error("Expected error for unhealthy store")
	}

	// Test Unregister
	if err := registry.Unregister("store1"); err != nil {
		t.Errorf("Unregister() error = %v", err)
	}

	if len(registry.List()) != 1 {
		t.Errorf("Expected 1 store after unregister, got %d", len(registry.List()))
	}

	// Test unregister non-existent store
	if err := registry.Unregister("nonexistent"); err == nil {
		t.Error("Expected error for unregistering non-existent store")
	}

	// Test Close
	if err := registry.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestGORMOptions(t *testing.T) {
	opts := GORMOptions{
		DSN:             "test-dsn",
		Driver:          "mysql",
		MaxIdleConns:    5,
		MaxOpenConns:    50,
		ConnMaxLifetime: 30 * time.Minute,
		Logger:          &testLogger{},
	}

	if opts.DSN != "test-dsn" {
		t.Errorf("DSN = %v, want test-dsn", opts.DSN)
	}

	if opts.Driver != "mysql" {
		t.Errorf("Driver = %v, want mysql", opts.Driver)
	}

	if opts.MaxIdleConns != 5 {
		t.Errorf("MaxIdleConns = %v, want 5", opts.MaxIdleConns)
	}

	if opts.MaxOpenConns != 50 {
		t.Errorf("MaxOpenConns = %v, want 50", opts.MaxOpenConns)
	}

	if opts.ConnMaxLifetime != 30*time.Minute {
		t.Errorf("ConnMaxLifetime = %v, want 30m", opts.ConnMaxLifetime)
	}

	if opts.Logger == nil {
		t.Error("Logger is nil")
	}
}

func TestNewGORMStore(t *testing.T) {
	logger := &testLogger{}

	tests := []struct {
		name    string
		opts    GORMOptions
		wantErr bool
	}{
		{
			name: "missing DSN",
			opts: GORMOptions{
				Driver: "mysql",
				Logger: logger,
			},
			wantErr: true,
		},
		{
			name: "missing driver",
			opts: GORMOptions{
				DSN:    "test-dsn",
				Logger: logger,
			},
			wantErr: true,
		},
		{
			name: "unsupported driver",
			opts: GORMOptions{
				DSN:    "test-dsn",
				Driver: "unsupported",
				Logger: logger,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewGORMStore(tt.opts)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewGORMStore() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && store == nil {
				t.Error("Expected store to be created")
			}
		})
	}
}

func TestConvenienceFunctions(t *testing.T) {
	logger := &testLogger{}

	// Test that convenience functions exist and can be called
	// We don't test actual database connections in unit tests
	t.Run("NewMySQLStore", func(t *testing.T) {
		_, err := NewMySQLStore("invalid-dsn", logger)
		if err == nil {
			t.Error("Expected error for invalid DSN")
		}
	})

	t.Run("NewPostgresStore", func(t *testing.T) {
		_, err := NewPostgresStore("invalid-dsn", logger)
		if err == nil {
			t.Error("Expected error for invalid DSN")
		}
	})

	t.Run("NewSQLiteStore", func(t *testing.T) {
		_, err := NewSQLiteStore("", logger)
		if err == nil {
			t.Error("Expected error for empty DSN")
		}
	})
}
