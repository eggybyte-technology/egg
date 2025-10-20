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
			name: "with OTLP endpoint",
			opts: Options{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				OTLPEndpoint:   "localhost:4317",
			},
			wantErr: false,
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
		{
			name: "with custom sampling ratio",
			opts: Options{
				ServiceName:       "test-service",
				ServiceVersion:    "1.0.0",
				TraceSamplerRatio: 0.5,
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

				if provider.TracerProvider == nil {
					t.Error("TracerProvider is nil")
				}

				if provider.MeterProvider == nil {
					t.Error("MeterProvider is nil")
				}

				// Test shutdown with shorter timeout for OTLP endpoint tests
				shutdownTimeout := 5 * time.Second
				if tt.opts.OTLPEndpoint != "" {
					shutdownTimeout = 1 * time.Second // Shorter timeout for OTLP tests
				}

				ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
				defer cancel()

				if err := provider.Shutdown(ctx); err != nil {
					// Ignore connection errors for OTLP endpoint tests
					if tt.opts.OTLPEndpoint != "" {
						t.Logf("Shutdown() error (expected for OTLP test): %v", err)
					} else {
						t.Errorf("Shutdown() error = %v", err)
					}
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

func TestProviderWithOTLPEndpoint(t *testing.T) {
	// This test might fail if there's no OTLP collector running
	// but it should still create the provider successfully
	provider, err := NewProvider(context.Background(), Options{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   "localhost:4317",
	})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("Provider is nil")
	}

	// Test shutdown with shorter timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := provider.Shutdown(ctx); err != nil {
		t.Logf("Shutdown() error (expected for OTLP test): %v", err)
	}
}

func TestProviderWithRuntimeMetrics(t *testing.T) {
	provider, err := NewProvider(context.Background(), Options{
		ServiceName:          "test-service",
		ServiceVersion:       "1.0.0",
		EnableRuntimeMetrics: true,
	})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("Provider is nil")
	}

	// Test shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}
