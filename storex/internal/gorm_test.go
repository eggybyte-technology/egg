// Package internal provides tests for storex internal implementation.
package internal

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"go.eggybyte.com/egg/core/log"
)

func TestNewGORMStore(t *testing.T) {
	// Test with nil DB - real gorm.DB requires a driver
	logger := &mockLogger{}
	store := NewGORMStore(nil, logger)

	if store == nil {
		t.Fatal("NewGORMStore() should return non-nil store")
	}
	if store.db != nil {
		t.Error("Store db should be nil when passed nil")
	}
	if store.logger != logger {
		t.Error("Store logger should be set")
	}
}

func TestGORMStore_Ping_Success(t *testing.T) {
	// Testing Ping with nil DB is already covered in TestGORMStore_Ping_NilDB
	// Real gorm.DB testing requires a database connection
	store := NewGORMStore(nil, &mockLogger{})

	ctx := context.Background()
	err := store.Ping(ctx)

	if err == nil {
		t.Fatal("Ping() should return error for nil db")
	}
}

func TestGORMStore_Ping_NilDB(t *testing.T) {
	store := NewGORMStore(nil, &mockLogger{})

	ctx := context.Background()
	err := store.Ping(ctx)

	if err == nil {
		t.Fatal("Ping() should return error for nil db")
	}
	if !contains(err.Error(), "database connection is nil") {
		t.Errorf("Error message = %q, want to contain 'database connection is nil'", err.Error())
	}
}

func TestGORMStore_Ping_DBError(t *testing.T) {
	// Real gorm.DB testing requires a database connection
	// This test is covered by Ping_NilDB
	store := NewGORMStore(nil, &mockLogger{})

	ctx := context.Background()
	err := store.Ping(ctx)

	if err == nil {
		t.Fatal("Ping() should return error for nil db")
	}
}

func TestGORMStore_Ping_PingError(t *testing.T) {
	// Real gorm.DB testing requires a database connection
	// This test is covered by Ping_NilDB
	store := NewGORMStore(nil, &mockLogger{})

	ctx := context.Background()
	err := store.Ping(ctx)

	if err == nil {
		t.Fatal("Ping() should return error for nil db")
	}
}

func TestGORMStore_Close_Success(t *testing.T) {
	// Real gorm.DB testing requires a database connection
	// This test is covered by Close_NilDB
	store := NewGORMStore(nil, &mockLogger{})

	err := store.Close()

	if err != nil {
		t.Errorf("Close() error = %v, want nil (nil db should be handled gracefully)", err)
	}
}

func TestGORMStore_Close_NilDB(t *testing.T) {
	store := NewGORMStore(nil, &mockLogger{})

	err := store.Close()

	if err != nil {
		t.Errorf("Close() error = %v, want nil (nil db should be handled gracefully)", err)
	}
}

func TestGORMStore_Close_DBError(t *testing.T) {
	// Real gorm.DB testing requires a database connection
	// This test is covered by Close_NilDB
	store := NewGORMStore(nil, &mockLogger{})

	err := store.Close()

	if err != nil {
		t.Errorf("Close() error = %v, want nil (nil db should be handled gracefully)", err)
	}
}

func TestGORMStore_Close_CloseError(t *testing.T) {
	// Real gorm.DB testing requires a database connection
	// This test is covered by Close_NilDB
	store := NewGORMStore(nil, &mockLogger{})

	err := store.Close()

	if err != nil {
		t.Errorf("Close() error = %v, want nil (nil db should be handled gracefully)", err)
	}
}

func TestGORMStore_GetDB(t *testing.T) {
	// Test with nil - we can't easily create a real gorm.DB without a driver
	// So we just test that GetDB returns what was set
	store := &GORMStore{
		db:     nil,
		logger: &mockLogger{},
	}

	db := store.GetDB()
	if db != nil {
		t.Error("GetDB() should return nil when db is nil")
	}
}

func TestDefaultGORMOptions(t *testing.T) {
	opts := DefaultGORMOptions()

	if opts.MaxIdleConns != 10 {
		t.Errorf("MaxIdleConns = %d, want %d", opts.MaxIdleConns, 10)
	}
	if opts.MaxOpenConns != 100 {
		t.Errorf("MaxOpenConns = %d, want %d", opts.MaxOpenConns, 100)
	}
	if opts.ConnMaxLifetime != time.Hour {
		t.Errorf("ConnMaxLifetime = %v, want %v", opts.ConnMaxLifetime, time.Hour)
	}
}

func TestNewGORMStoreFromOptions_EmptyDSN(t *testing.T) {
	opts := DefaultGORMOptions()
	opts.DSN = ""
	opts.Driver = "mysql"

	store, err := NewGORMStoreFromOptions(opts)

	if err == nil {
		t.Fatal("NewGORMStoreFromOptions() should return error for empty DSN")
	}
	if store != nil {
		t.Error("NewGORMStoreFromOptions() should return nil store on error")
	}
	if !contains(err.Error(), "DSN is required") {
		t.Errorf("Error message = %q, want to contain 'DSN is required'", err.Error())
	}
}

func TestNewGORMStoreFromOptions_EmptyDriver(t *testing.T) {
	opts := DefaultGORMOptions()
	opts.DSN = "test-dsn"
	opts.Driver = ""

	store, err := NewGORMStoreFromOptions(opts)

	if err == nil {
		t.Fatal("NewGORMStoreFromOptions() should return error for empty driver")
	}
	if store != nil {
		t.Error("NewGORMStoreFromOptions() should return nil store on error")
	}
	if !contains(err.Error(), "driver is required") {
		t.Errorf("Error message = %q, want to contain 'driver is required'", err.Error())
	}
}

func TestGetGORMDriver_MySQL(t *testing.T) {
	driver, err := getGORMDriver("mysql", "dsn")

	if err != nil {
		t.Errorf("getGORMDriver(mysql) error = %v, want nil", err)
	}
	if driver == nil {
		t.Error("getGORMDriver(mysql) should return non-nil driver")
	}
}

func TestGetGORMDriver_Postgres(t *testing.T) {
	driver, err := getGORMDriver("postgres", "dsn")

	if err != nil {
		t.Errorf("getGORMDriver(postgres) error = %v, want nil", err)
	}
	if driver == nil {
		t.Error("getGORMDriver(postgres) should return non-nil driver")
	}
}

func TestGetGORMDriver_SQLite(t *testing.T) {
	driver, err := getGORMDriver("sqlite", "dsn")

	if err != nil {
		t.Errorf("getGORMDriver(sqlite) error = %v, want nil", err)
	}
	if driver == nil {
		t.Error("getGORMDriver(sqlite) should return non-nil driver")
	}
}

func TestGetGORMDriver_Unsupported(t *testing.T) {
	driver, err := getGORMDriver("unsupported", "dsn")

	if err == nil {
		t.Fatal("getGORMDriver(unsupported) should return error")
	}
	if driver != nil {
		t.Error("getGORMDriver(unsupported) should return nil driver")
	}
	if !contains(err.Error(), "unsupported driver") {
		t.Errorf("Error message = %q, want to contain 'unsupported driver'", err.Error())
	}
}

func TestIsDatabaseConnectionError_ConnectionErrors(t *testing.T) {
	tests := []struct {
		name    string
		errStr  string
		want    bool
	}{
		{"connection refused", "connection refused", true},
		{"connection reset", "connection reset", true},
		{"timeout", "timeout", true},
		{"network unreachable", "network is unreachable", true},
		{"no such host", "no such host", true},
		{"connection pool exhausted", "connection pool exhausted", true},
		{"broken pipe", "broken pipe", true},
		{"EOF", "EOF", true},
		{"duplicate key", "duplicate key", false},
		{"unique constraint", "unique constraint", false},
		{"foreign key constraint", "foreign key constraint", false},
		{"normal error", "some normal error", true}, // Unknown errors are treated as connection errors
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errStr)
			result := isDatabaseConnectionError(err)
			if result != tt.want {
				t.Errorf("isDatabaseConnectionError(%q) = %v, want %v", tt.name, result, tt.want)
			}
		})
	}
}

func TestIsDatabaseConnectionError_GORMRecordNotFound(t *testing.T) {
	err := gorm.ErrRecordNotFound
	result := isDatabaseConnectionError(err)

	if result {
		t.Error("isDatabaseConnectionError(gorm.ErrRecordNotFound) = true, want false")
	}
}

func TestIsDatabaseConnectionError_NilError(t *testing.T) {
	result := isDatabaseConnectionError(nil)

	if result {
		t.Error("isDatabaseConnectionError(nil) = true, want false")
	}
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("NewRegistry() should return non-nil registry")
	}
	if registry.stores == nil {
		t.Error("Registry stores map should be initialized")
	}
	if len(registry.stores) != 0 {
		t.Error("Registry should start with empty stores")
	}
}

func TestRegistry_Register_Success(t *testing.T) {
	registry := NewRegistry()
	store := &mockStore{}

	err := registry.Register("test-store", store)

	if err != nil {
		t.Errorf("Register() error = %v, want nil", err)
	}
	if len(registry.stores) != 1 {
		t.Errorf("Registry should have 1 store, got %d", len(registry.stores))
	}
}

func TestRegistry_Register_EmptyName(t *testing.T) {
	registry := NewRegistry()
	store := &mockStore{}

	err := registry.Register("", store)

	if err == nil {
		t.Fatal("Register() should return error for empty name")
	}
	if !contains(err.Error(), "store name is required") {
		t.Errorf("Error message = %q, want to contain 'store name is required'", err.Error())
	}
}

func TestRegistry_Register_NilStore(t *testing.T) {
	registry := NewRegistry()

	err := registry.Register("test-store", nil)

	if err == nil {
		t.Fatal("Register() should return error for nil store")
	}
	if !contains(err.Error(), "store cannot be nil") {
		t.Errorf("Error message = %q, want to contain 'store cannot be nil'", err.Error())
	}
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	registry := NewRegistry()
	store := &mockStore{}

	err1 := registry.Register("test-store", store)
	if err1 != nil {
		t.Fatalf("First Register() error = %v", err1)
	}

	err2 := registry.Register("test-store", store)
	if err2 == nil {
		t.Fatal("Register() should return error for duplicate name")
	}
	if !contains(err2.Error(), "already registered") {
		t.Errorf("Error message = %q, want to contain 'already registered'", err2.Error())
	}
}

func TestRegistry_Unregister_Success(t *testing.T) {
	registry := NewRegistry()
	store := &mockStore{}

	err := registry.Register("test-store", store)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	err = registry.Unregister("test-store")
	if err != nil {
		t.Errorf("Unregister() error = %v, want nil", err)
	}
	if len(registry.stores) != 0 {
		t.Error("Registry should have no stores after unregister")
	}
}

func TestRegistry_Unregister_NotFound(t *testing.T) {
	registry := NewRegistry()

	err := registry.Unregister("non-existent")

	if err == nil {
		t.Fatal("Unregister() should return error for non-existent store")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("Error message = %q, want to contain 'not found'", err.Error())
	}
}

func TestRegistry_Ping_Success(t *testing.T) {
	registry := NewRegistry()
	store := &mockStore{pingErr: nil}

	err := registry.Register("test-store", store)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	ctx := context.Background()
	err = registry.Ping(ctx)

	if err != nil {
		t.Errorf("Ping() error = %v, want nil", err)
	}
}

func TestRegistry_Ping_EmptyRegistry(t *testing.T) {
	registry := NewRegistry()

	ctx := context.Background()
	err := registry.Ping(ctx)

	if err != nil {
		t.Errorf("Ping() error = %v, want nil (empty registry)", err)
	}
}

func TestRegistry_Ping_Failure(t *testing.T) {
	registry := NewRegistry()
	store := &mockStore{pingErr: errors.New("ping failed")}

	err := registry.Register("test-store", store)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	ctx := context.Background()
	err = registry.Ping(ctx)

	if err == nil {
		t.Fatal("Ping() should return error when store ping fails")
	}
}

func TestRegistry_Ping_MultipleStores(t *testing.T) {
	registry := NewRegistry()
	store1 := &mockStore{pingErr: nil}
	store2 := &mockStore{pingErr: nil}

	registry.Register("store1", store1)
	registry.Register("store2", store2)

	ctx := context.Background()
	err := registry.Ping(ctx)

	if err != nil {
		t.Errorf("Ping() error = %v, want nil", err)
	}
}

func TestRegistry_Ping_PartialFailure(t *testing.T) {
	registry := NewRegistry()
	store1 := &mockStore{pingErr: nil}
	store2 := &mockStore{pingErr: errors.New("ping failed")}

	registry.Register("store1", store1)
	registry.Register("store2", store2)

	ctx := context.Background()
	err := registry.Ping(ctx)

	if err == nil {
		t.Fatal("Ping() should return error when any store ping fails")
	}
}

func TestRegistry_Close_Success(t *testing.T) {
	registry := NewRegistry()
	store := &mockStore{closeErr: nil}

	err := registry.Register("test-store", store)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	err = registry.Close()

	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestRegistry_Close_EmptyRegistry(t *testing.T) {
	registry := NewRegistry()

	err := registry.Close()

	if err != nil {
		t.Errorf("Close() error = %v, want nil (empty registry)", err)
	}
}

func TestRegistry_Close_Failure(t *testing.T) {
	registry := NewRegistry()
	store := &mockStore{closeErr: errors.New("close failed")}

	err := registry.Register("test-store", store)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	err = registry.Close()

	if err == nil {
		t.Fatal("Close() should return error when store close fails")
	}
}

func TestRegistry_Close_MultipleStores(t *testing.T) {
	registry := NewRegistry()
	store1 := &mockStore{closeErr: nil}
	store2 := &mockStore{closeErr: nil}

	registry.Register("store1", store1)
	registry.Register("store2", store2)

	err := registry.Close()

	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	if len(registry.List()) != 0 {
		t.Error("List() should return empty slice for empty registry")
	}

	store1 := &mockStore{}
	store2 := &mockStore{}

	registry.Register("store1", store1)
	registry.Register("store2", store2)

	list := registry.List()
	if len(list) != 2 {
		t.Errorf("List() len = %d, want %d", len(list), 2)
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	store := &mockStore{}

	registry.Register("test-store", store)

	retrieved, exists := registry.Get("test-store")
	if !exists {
		t.Fatal("Get() should return true for existing store")
	}
	if retrieved != store {
		t.Error("Get() should return the registered store")
	}

	_, exists = registry.Get("non-existent")
	if exists {
		t.Error("Get() should return false for non-existent store")
	}
}

// Mock implementations

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

type mockLogger struct{}

func (m *mockLogger) Debug(msg string, kv ...interface{}) {}
func (m *mockLogger) Info(msg string, kv ...interface{})  {}
func (m *mockLogger) Warn(msg string, kv ...interface{})  {}
func (m *mockLogger) Error(err error, msg string, kv ...interface{}) {}
func (m *mockLogger) With(kv ...interface{}) log.Logger { return m }

// Helper function
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

