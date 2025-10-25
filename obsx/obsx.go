// Package obsx provides OpenTelemetry and Prometheus provider initialization.
//
// Overview:
//   - Responsibility: Bootstrap OpenTelemetry tracing and metrics providers
//   - Key Types: Options for configuration, Provider for managing lifecycle
//   - Concurrency Model: Provider is safe for concurrent use
//   - Error Semantics: NewProvider returns error for initialization failures
//   - Performance Notes: Supports configurable sampling and resource attributes
//
// Usage:
//
//	provider, err := obsx.NewProvider(ctx, obsx.Options{
//	  ServiceName: "my-service",
//	  ServiceVersion: "1.0.0",
//	  OTLPEndpoint: "otel-collector:4317",
//	  EnableRuntimeMetrics: true,
//	})
//	defer provider.Shutdown(ctx)
package obsx

import (
	"context"

	"github.com/eggybyte-technology/egg/obsx/internal"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Options holds configuration for the observability provider.
type Options struct {
	ServiceName          string            // Service name for tracing and metrics
	ServiceVersion       string            // Service version
	OTLPEndpoint         string            // OTLP endpoint (e.g., "otel-collector:4317")
	EnableRuntimeMetrics bool              // Enable Go runtime metrics
	ResourceAttrs        map[string]string // Additional resource attributes
	TraceSamplerRatio    float64           // Trace sampling ratio (0.0-1.0)
}

// Provider manages OpenTelemetry tracing and metrics providers.
// The provider must be shut down when no longer needed.
type Provider struct {
	impl *internal.Provider
}

// TracerProvider returns the OpenTelemetry tracer provider.
func (p *Provider) TracerProvider() *sdktrace.TracerProvider {
	return p.impl.TracerProvider
}

// MeterProvider returns the OpenTelemetry meter provider.
func (p *Provider) MeterProvider() *metric.MeterProvider {
	return p.impl.MeterProvider
}

// NewProvider creates a new observability provider with the given options.
// The provider must be shut down when no longer needed.
//
// Parameters:
//   - ctx: context for provider initialization
//   - opts: provider configuration options
//
// Returns:
//   - *Provider: initialized provider instance
//   - error: initialization error if any
//
// Concurrency:
//   - Safe to call from multiple goroutines
//
// Performance:
//   - Default sampling ratio is 10% if not specified
func NewProvider(ctx context.Context, opts Options) (*Provider, error) {
	impl, err := internal.NewProvider(ctx, internal.ProviderOptions{
		ServiceName:       opts.ServiceName,
		ServiceVersion:    opts.ServiceVersion,
		OTLPEndpoint:      opts.OTLPEndpoint,
		ResourceAttrs:     opts.ResourceAttrs,
		TraceSamplerRatio: opts.TraceSamplerRatio,
	})
	if err != nil {
		return nil, err
	}

	return &Provider{impl: impl}, nil
}

// Shutdown gracefully shuts down the provider.
// This should be called when the application is shutting down.
//
// Parameters:
//   - ctx: context with shutdown timeout
//
// Returns:
//   - error: shutdown error if any
//
// Concurrency:
//   - Safe to call from multiple goroutines
//   - Blocks until shutdown completes or timeout
func (p *Provider) Shutdown(ctx context.Context) error {
	return p.impl.Shutdown(ctx)
}
