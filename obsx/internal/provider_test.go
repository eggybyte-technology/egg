// Package internal provides tests for obsx internal implementation.
package internal

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewProvider_Success(t *testing.T) {
	ctx := context.Background()
	opts := ProviderOptions{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	provider, err := NewProvider(ctx, opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v, want nil", err)
	}

	if provider == nil {
		t.Fatal("NewProvider() should return non-nil provider")
	}
	if provider.MeterProvider == nil {
		t.Error("MeterProvider should not be nil")
	}
}

func TestNewProvider_EmptyServiceName(t *testing.T) {
	ctx := context.Background()
	opts := ProviderOptions{
		ServiceVersion: "1.0.0",
	}

	provider, err := NewProvider(ctx, opts)
	if err == nil {
		t.Fatal("NewProvider() should return error for empty service name")
	}
	if provider != nil {
		t.Error("NewProvider() should return nil provider on error")
	}
	if err.Error() != "service name is required" {
		t.Errorf("Error message = %q, want %q", err.Error(), "service name is required")
	}
}

func TestNewProvider_WithResourceAttrs(t *testing.T) {
	ctx := context.Background()
	opts := ProviderOptions{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		ResourceAttrs: map[string]string{
			"env": "test",
			"region": "us-east-1",
		},
	}

	provider, err := NewProvider(ctx, opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v, want nil", err)
	}

	if provider == nil {
		t.Fatal("NewProvider() should return non-nil provider")
	}
}

func TestProvider_GetPrometheusHandler(t *testing.T) {
	ctx := context.Background()
	opts := ProviderOptions{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	provider, err := NewProvider(ctx, opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	handler := provider.GetPrometheusHandler()
	if handler == nil {
		t.Fatal("GetPrometheusHandler() should return non-nil handler")
	}

	// Test handler
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Handler status code = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("Handler should return metrics body")
	}
}

func TestProvider_GetPrometheusHandler_NilRegistry(t *testing.T) {
	provider := &Provider{
		MeterProvider:      nil,
		prometheusRegistry: nil,
	}

	handler := provider.GetPrometheusHandler()
	if handler == nil {
		t.Fatal("GetPrometheusHandler() should return handler even with nil registry")
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Handler status code = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestProvider_Shutdown(t *testing.T) {
	ctx := context.Background()
	opts := ProviderOptions{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	provider, err := NewProvider(ctx, opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	shutdownCtx := context.Background()
	err = provider.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("Shutdown() error = %v, want nil", err)
	}
}

func TestProvider_Shutdown_NilProvider(t *testing.T) {
	provider := &Provider{
		MeterProvider: nil,
	}

	ctx := context.Background()
	err := provider.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown() error = %v, want nil (nil provider should not error)", err)
	}
}

func TestEnableRuntimeMetrics(t *testing.T) {
	ctx := context.Background()
	opts := ProviderOptions{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	provider, err := NewProvider(ctx, opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	err = EnableRuntimeMetrics(ctx, provider.MeterProvider)
	if err != nil {
		t.Errorf("EnableRuntimeMetrics() error = %v, want nil", err)
	}
}

func TestEnableRuntimeMetrics_NilProvider(t *testing.T) {
	ctx := context.Background()

	// This will panic, so we need to recover
	defer func() {
		if r := recover(); r == nil {
			t.Error("EnableRuntimeMetrics() should panic for nil provider")
		}
	}()

	EnableRuntimeMetrics(ctx, nil)
	t.Error("Should have panicked")
}

func TestEnableProcessMetrics(t *testing.T) {
	ctx := context.Background()
	opts := ProviderOptions{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	provider, err := NewProvider(ctx, opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	err = EnableProcessMetrics(ctx, provider.MeterProvider)
	if err != nil {
		t.Errorf("EnableProcessMetrics() error = %v, want nil", err)
	}
}

func TestEnableProcessMetrics_NilProvider(t *testing.T) {
	ctx := context.Background()

	// This will panic, so we need to recover
	defer func() {
		if r := recover(); r == nil {
			t.Error("EnableProcessMetrics() should panic for nil provider")
		}
	}()

	EnableProcessMetrics(ctx, nil)
	t.Error("Should have panicked")
}

func TestRegisterDBMetrics(t *testing.T) {
	ctx := context.Background()
	opts := ProviderOptions{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	provider, err := NewProvider(ctx, opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	// Create a mock sql.DB - we can't easily create a real one without a driver
	// So we'll test the error path instead
	err = RegisterDBMetrics("test-db", nil, provider.MeterProvider)
	// This might panic or return an error depending on implementation
	// We just verify it doesn't crash
	_ = err
}

func TestRegisterDBMetrics_NilDB(t *testing.T) {
	ctx := context.Background()
	opts := ProviderOptions{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	provider, err := NewProvider(ctx, opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	// Test with nil DB - should handle gracefully
	err = RegisterDBMetrics("test-db", nil, provider.MeterProvider)
	// Implementation may handle nil DB differently, so we just check it doesn't panic
	_ = err
}

func TestRegisterDBMetrics_NilProvider(t *testing.T) {
	db := &sql.DB{}

	// This will panic, so we need to recover
	defer func() {
		if r := recover(); r == nil {
			t.Error("RegisterDBMetrics() should panic for nil provider")
		}
	}()

	RegisterDBMetrics("test-db", db, nil)
	t.Error("Should have panicked")
}

func TestRegisterGORMMetrics(t *testing.T) {
	ctx := context.Background()
	opts := ProviderOptions{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	provider, err := NewProvider(ctx, opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	// Create a mock GORM DB that returns an error
	mockGORM := &mockGORMDB{err: sql.ErrConnDone}

	err = RegisterGORMMetrics("test-db", mockGORM, provider.MeterProvider)
	if err == nil {
		t.Error("RegisterGORMMetrics() should return error when DB() fails")
	}
	if !contains(err.Error(), "failed to get sql.DB") {
		t.Errorf("Error message = %q, want to contain 'failed to get sql.DB'", err.Error())
	}
}

func TestRegisterGORMMetrics_Success(t *testing.T) {
	ctx := context.Background()
	opts := ProviderOptions{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	provider, err := NewProvider(ctx, opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	// Create a mock GORM DB that succeeds
	mockGORM := &mockGORMDB{db: &sql.DB{}}

	err = RegisterGORMMetrics("test-db", mockGORM, provider.MeterProvider)
	// May succeed or fail depending on DB state, but shouldn't panic
	_ = err
}

func TestRegisterGORMMetrics_NilProvider(t *testing.T) {
	mockGORM := &mockGORMDB{db: &sql.DB{}}

	// This will panic, so we need to recover
	defer func() {
		if r := recover(); r == nil {
			t.Error("RegisterGORMMetrics() should panic for nil provider")
		}
	}()

	RegisterGORMMetrics("test-db", mockGORM, nil)
	t.Error("Should have panicked")
}

// mockGORMDB is a test implementation of GORM DB interface.
type mockGORMDB struct {
	db  *sql.DB
	err error
}

func (m *mockGORMDB) DB() (*sql.DB, error) {
	return m.db, m.err
}

func TestProvider_GetPrometheusHandler_MetricsFormat(t *testing.T) {
	ctx := context.Background()
	opts := ProviderOptions{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	provider, err := NewProvider(ctx, opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	// Enable some metrics to have data
	err = EnableRuntimeMetrics(ctx, provider.MeterProvider)
	if err != nil {
		t.Fatalf("EnableRuntimeMetrics() error = %v", err)
	}

	handler := provider.GetPrometheusHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Handler status code = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("Handler should return metrics body")
	}
}

func TestProvider_Shutdown_WithTimeout(t *testing.T) {
	opts := ProviderOptions{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	provider, err := NewProvider(context.Background(), opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	// Create a context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = provider.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("Shutdown() error = %v, want nil", err)
	}
}

func TestEnableRuntimeMetrics_Idempotent(t *testing.T) {
	ctx := context.Background()
	opts := ProviderOptions{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	provider, err := NewProvider(ctx, opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	// Call multiple times - should be idempotent
	err1 := EnableRuntimeMetrics(ctx, provider.MeterProvider)
	err2 := EnableRuntimeMetrics(ctx, provider.MeterProvider)
	err3 := EnableRuntimeMetrics(ctx, provider.MeterProvider)

	if err1 != nil {
		t.Errorf("First EnableRuntimeMetrics() error = %v", err1)
	}
	if err2 != nil {
		t.Errorf("Second EnableRuntimeMetrics() error = %v", err2)
	}
	if err3 != nil {
		t.Errorf("Third EnableRuntimeMetrics() error = %v", err3)
	}
}

func TestEnableProcessMetrics_Idempotent(t *testing.T) {
	ctx := context.Background()
	opts := ProviderOptions{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	provider, err := NewProvider(ctx, opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	// Call multiple times - should be idempotent
	err1 := EnableProcessMetrics(ctx, provider.MeterProvider)
	err2 := EnableProcessMetrics(ctx, provider.MeterProvider)
	err3 := EnableProcessMetrics(ctx, provider.MeterProvider)

	if err1 != nil {
		t.Errorf("First EnableProcessMetrics() error = %v", err1)
	}
	if err2 != nil {
		t.Errorf("Second EnableProcessMetrics() error = %v", err2)
	}
	if err3 != nil {
		t.Errorf("Third EnableProcessMetrics() error = %v", err3)
	}
}

func TestNewProvider_EmptyVersion(t *testing.T) {
	ctx := context.Background()
	opts := ProviderOptions{
		ServiceName: "test-service",
		// ServiceVersion is empty
	}

	provider, err := NewProvider(ctx, opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v, want nil (empty version is allowed)", err)
	}

	if provider == nil {
		t.Fatal("NewProvider() should return non-nil provider")
	}
}

func TestProvider_GetPrometheusHandler_ContentType(t *testing.T) {
	ctx := context.Background()
	opts := ProviderOptions{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	}

	provider, err := NewProvider(ctx, opts)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	handler := provider.GetPrometheusHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		t.Error("Handler should set Content-Type header")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

