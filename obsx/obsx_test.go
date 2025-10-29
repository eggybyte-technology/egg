// Package obsx provides tests for observability provider.
package obsx

import (
	"context"
	"testing"
	"time"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantErr bool
	}{
		{
			name: "valid options",
			opts: Options{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "missing service name",
			opts: Options{
				ServiceVersion: "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "with resource attributes",
			opts: Options{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				ResourceAttrs: map[string]string{
					"environment": "test",
					"region":      "us-west-2",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(context.Background(), tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if provider == nil {
					t.Error("NewProvider() returned nil provider")
					return
				}

				if provider.MeterProvider() == nil {
					t.Error("MeterProvider is nil")
				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				if err := provider.Shutdown(ctx); err != nil {
					t.Errorf("Shutdown() error = %v", err)
				}
			}
		})
	}
}

func TestProviderShutdown(t *testing.T) {
	provider, err := NewProvider(context.Background(), Options{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	// Test normal shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	// Test shutdown on already shutdown provider
	// This might return an error, which is acceptable
	ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel2()

	if err := provider.Shutdown(ctx2); err != nil {
		t.Logf("Second Shutdown() error (expected): %v", err)
	}
}

func TestProviderWithRuntimeMetrics(t *testing.T) {
	provider, err := NewProvider(context.Background(), Options{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("Provider is nil")
	}

	// Enable runtime metrics
	ctx := context.Background()
	if err := provider.EnableRuntimeMetrics(ctx); err != nil {
		t.Errorf("EnableRuntimeMetrics() error = %v", err)
	}

	// Test shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.Shutdown(shutdownCtx); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}
