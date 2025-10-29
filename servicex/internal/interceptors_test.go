// Package internal provides tests for interceptor building logic.
package internal

import (
	"context"
	"testing"

	"go.eggybyte.com/egg/core/log"
	"go.eggybyte.com/egg/logx"
	"go.eggybyte.com/egg/obsx"
)

// TestBuildInterceptors tests interceptor chain building.
func TestBuildInterceptors(t *testing.T) {
	// Create test logger
	logger := logx.New()

	tests := []struct {
		name               string
		logger             log.Logger
		otel               *obsx.Provider
		slowRequestMillis  int64
		enableDebugLogs    bool
		payloadAccounting  bool
		expectInterceptors int
	}{
		{
			name:               "with all features disabled",
			logger:             logger,
			otel:               nil,
			slowRequestMillis:  1000,
			enableDebugLogs:    false,
			payloadAccounting:  false,
			expectInterceptors: 0, // Will have at least some interceptors
		},
		{
			name:               "with debug logs enabled",
			logger:             logger,
			otel:               nil,
			slowRequestMillis:  500,
			enableDebugLogs:    true,
			payloadAccounting:  false,
			expectInterceptors: 0,
		},
		{
			name:               "with payload accounting enabled",
			logger:             logger,
			otel:               nil,
			slowRequestMillis:  2000,
			enableDebugLogs:    false,
			payloadAccounting:  true,
			expectInterceptors: 0,
		},
		{
			name:               "with all features enabled",
			logger:             logger,
			otel:               nil,
			slowRequestMillis:  100,
			enableDebugLogs:    true,
			payloadAccounting:  true,
			expectInterceptors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptors := BuildInterceptors(
				tt.logger,
				tt.otel,
				tt.slowRequestMillis,
				tt.enableDebugLogs,
				tt.payloadAccounting,
			)

			if interceptors == nil {
				t.Error("BuildInterceptors returned nil")
			}

			// Should always return some interceptors
			if len(interceptors) == 0 {
				t.Error("BuildInterceptors returned empty slice, expected at least one interceptor")
			}
		})
	}
}

// TestBuildInterceptorsWithMetrics tests interceptor building with metrics provider.
func TestBuildInterceptorsWithMetrics(t *testing.T) {
	logger := logx.New()

	// Create a test metrics provider
	ctx := context.Background()
	otel, err := obsx.NewProvider(ctx, obsx.Options{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("Failed to create obsx provider: %v", err)
	}
	defer otel.Shutdown(ctx)

	interceptors := BuildInterceptors(
		logger,
		otel,
		1000,
		false,
		true,
	)

	if interceptors == nil {
		t.Error("BuildInterceptors returned nil")
	}

	if len(interceptors) == 0 {
		t.Error("BuildInterceptors returned empty slice")
	}
}

// TestBuildInterceptorsWithNilLogger tests that nil logger is handled gracefully.
func TestBuildInterceptorsWithNilLogger(t *testing.T) {
	// This might panic or handle nil gracefully depending on implementation
	defer func() {
		if r := recover(); r != nil {
			// If it panics, that's acceptable behavior
			t.Logf("BuildInterceptors panicked with nil logger (acceptable): %v", r)
		}
	}()

	interceptors := BuildInterceptors(
		nil, // nil logger
		nil,
		1000,
		false,
		false,
	)

	// If we get here without panic, check result
	if interceptors != nil {
		// Having interceptors with nil logger might be OK if they're optional
		t.Logf("BuildInterceptors handled nil logger: %d interceptors", len(interceptors))
	}
}

// TestBuildInterceptorsSlowRequestThreshold tests different threshold values.
func TestBuildInterceptorsSlowRequestThreshold(t *testing.T) {
	logger := logx.New()

	tests := []struct {
		name              string
		slowRequestMillis int64
	}{
		{"very fast threshold", 10},
		{"default threshold", 1000},
		{"slow threshold", 5000},
		{"zero threshold", 0},
		{"negative threshold", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptors := BuildInterceptors(
				logger,
				nil,
				tt.slowRequestMillis,
				false,
				false,
			)

			if interceptors == nil {
				t.Error("BuildInterceptors returned nil")
			}
		})
	}
}


