// Package internal provides tests for configx internal manager implementation.
package internal

import (
	"context"
	"testing"
	"time"

	"go.eggybyte.com/egg/core/log"
)

func TestNewManager_Success(t *testing.T) {
	logger := &mockLogger{}
	sources := []Source{
		NewEnvSource(EnvOptions{}),
	}

	manager, err := NewManager(logger, sources, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewManager() error = %v, want nil", err)
	}

	if manager == nil {
		t.Fatal("NewManager() should return non-nil manager")
	}
	if manager.logger != logger {
		t.Error("Manager logger should be set")
	}
	if len(manager.sources) != 1 {
		t.Errorf("Manager sources len = %d, want %d", len(manager.sources), 1)
	}
}

func TestNewManager_NilLogger(t *testing.T) {
	sources := []Source{
		NewEnvSource(EnvOptions{}),
	}

	manager, err := NewManager(nil, sources, 0)
	if err == nil {
		t.Fatal("NewManager() should return error for nil logger")
	}
	if manager != nil {
		t.Error("NewManager() should return nil manager on error")
	}
	if err.Error() != "logger is required" {
		t.Errorf("Error message = %q, want %q", err.Error(), "logger is required")
	}
}

func TestNewManager_EmptySources(t *testing.T) {
	logger := &mockLogger{}
	sources := []Source{}

	manager, err := NewManager(logger, sources, 0)
	if err == nil {
		t.Fatal("NewManager() should return error for empty sources")
	}
	if manager != nil {
		t.Error("NewManager() should return nil manager on error")
	}
	if err.Error() != "at least one source is required" {
		t.Errorf("Error message = %q, want %q", err.Error(), "at least one source is required")
	}
}

func TestNewManager_DefaultDebounce(t *testing.T) {
	logger := &mockLogger{}
	sources := []Source{
		NewEnvSource(EnvOptions{}),
	}

	manager, err := NewManager(logger, sources, 0)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if manager.debounce != 200*time.Millisecond {
		t.Errorf("Debounce = %v, want %v", manager.debounce, 200*time.Millisecond)
	}
}

func TestManagerImpl_Snapshot(t *testing.T) {
	logger := &mockLogger{}
	sources := []Source{
		NewEnvSource(EnvOptions{}),
	}

	manager, err := NewManager(logger, sources, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	snapshot := manager.Snapshot()
	if snapshot == nil {
		t.Fatal("Snapshot() should return non-nil map")
	}
}

func TestManagerImpl_Value(t *testing.T) {
	logger := &mockLogger{}
	sources := []Source{
		NewEnvSource(EnvOptions{}),
	}

	manager, err := NewManager(logger, sources, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Test with non-existent key
	value, exists := manager.Value("NON_EXISTENT_KEY")
	if exists {
		t.Error("Value() should return false for non-existent key")
	}
	if value != "" {
		t.Errorf("Value() = %q, want empty string", value)
	}
}

func TestManagerImpl_Bind_NilTarget(t *testing.T) {
	logger := &mockLogger{}
	sources := []Source{
		NewEnvSource(EnvOptions{}),
	}

	manager, err := NewManager(logger, sources, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	err = manager.Bind(nil, BindConfig{})
	if err == nil {
		t.Fatal("Bind() should return error for nil target")
	}
	if err.Error() != "target cannot be nil" {
		t.Errorf("Error message = %q, want %q", err.Error(), "target cannot be nil")
	}
}

func TestManagerImpl_OnUpdate(t *testing.T) {
	logger := &mockLogger{}
	sources := []Source{
		NewEnvSource(EnvOptions{}),
	}

	manager, err := NewManager(logger, sources, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	unsubscribe := manager.OnUpdate(func(snapshot map[string]string) {
		// Test callback
	})

	if unsubscribe == nil {
		t.Fatal("OnUpdate() should return unsubscribe function")
	}

	// Unsubscribe
	unsubscribe()

	// Verify subscription was removed
	manager.subsMu.RLock()
	count := len(manager.updateSubs)
	manager.subsMu.RUnlock()

	if count != 0 {
		t.Errorf("Subscription count = %d, want %d", count, 0)
	}
}

func TestManagerImpl_OnUpdate_Multiple(t *testing.T) {
	logger := &mockLogger{}
	sources := []Source{
		NewEnvSource(EnvOptions{}),
	}

	manager, err := NewManager(logger, sources, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	unsub1 := manager.OnUpdate(func(snapshot map[string]string) {})
	unsub2 := manager.OnUpdate(func(snapshot map[string]string) {})

	manager.subsMu.RLock()
	count := len(manager.updateSubs)
	manager.subsMu.RUnlock()

	if count != 2 {
		t.Errorf("Subscription count = %d, want %d", count, 2)
	}

	unsub1()
	unsub2()

	manager.subsMu.RLock()
	count = len(manager.updateSubs)
	manager.subsMu.RUnlock()

	if count != 0 {
		t.Errorf("Subscription count after unsubscribe = %d, want %d", count, 0)
	}
}

func TestManagerImpl_Initialize(t *testing.T) {
	logger := &mockLogger{}
	sources := []Source{
		NewEnvSource(EnvOptions{}),
	}

	manager, err := NewManager(logger, sources, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	ctx := context.Background()
	err = manager.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize() error = %v, want nil", err)
	}
}

func TestMaskSensitiveValue_Empty(t *testing.T) {
	result := maskSensitiveValue("key", "")
	if result != "(empty)" {
		t.Errorf("maskSensitiveValue(empty) = %q, want %q", result, "(empty)")
	}
}

func TestMaskSensitiveValue_Password(t *testing.T) {
	result := maskSensitiveValue("password", "secret123")
	if result == "secret123" {
		t.Error("maskSensitiveValue(password) should mask the value")
	}
	if len(result) <= 8 {
		result = maskSensitiveValue("password", "verylongpassword123")
		if result == "verylongpassword123" {
			t.Error("maskSensitiveValue(long password) should mask the value")
		}
	}
}

func TestMaskSensitiveValue_Secret(t *testing.T) {
	result := maskSensitiveValue("api_secret", "mysecretkey")
	if result == "mysecretkey" {
		t.Error("maskSensitiveValue(secret) should mask the value")
	}
}

func TestMaskSensitiveValue_Token(t *testing.T) {
	result := maskSensitiveValue("auth_token", "token123")
	if result == "token123" {
		t.Error("maskSensitiveValue(token) should mask the value")
	}
}

func TestMaskSensitiveValue_DSN(t *testing.T) {
	dsn := "user:password@host:port/database"
	result := maskSensitiveValue("database_dsn", dsn)
	if result == dsn {
		t.Error("maskSensitiveValue(dsn) should mask the password")
	}
	// DSN masking should produce a masked version
	// The exact format depends on implementation, just verify it's masked
	if len(result) == len(dsn) && result == dsn {
		t.Error("maskSensitiveValue(dsn) should produce different result")
	}
}

func TestMaskSensitiveValue_NonSensitive(t *testing.T) {
	result := maskSensitiveValue("app_name", "myapp")
	if result != "myapp" {
		t.Errorf("maskSensitiveValue(non-sensitive) = %q, want %q", result, "myapp")
	}
}

func TestMaskSensitiveValue_ShortValue(t *testing.T) {
	result := maskSensitiveValue("password", "1234")
	if result != "***" {
		t.Errorf("maskSensitiveValue(short) = %q, want %q", result, "***")
	}
}

// mockLogger is a test implementation of log.Logger.
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, kv ...interface{}) {}
func (m *mockLogger) Info(msg string, kv ...interface{})  {}
func (m *mockLogger) Warn(msg string, kv ...interface{}) {}
func (m *mockLogger) Error(err error, msg string, kv ...interface{}) {}
func (m *mockLogger) With(kv ...interface{}) log.Logger { return m }

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

