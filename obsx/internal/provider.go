// Package internal provides internal implementation for the obsx package.
package internal

import (
	"context"
	"fmt"
	"net/http"
	"time"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// ProviderOptions holds configuration for the metrics provider.
type ProviderOptions struct {
	ServiceName    string
	ServiceVersion string
	ResourceAttrs  map[string]string
}

// Provider manages OpenTelemetry metrics provider with Prometheus export.
type Provider struct {
	MeterProvider      *metric.MeterProvider
	prometheusRegistry *promclient.Registry
}

// NewProvider creates a new metrics provider with Prometheus export.
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
func NewProvider(ctx context.Context, opts ProviderOptions) (*Provider, error) {
	if opts.ServiceName == "" {
		return nil, fmt.Errorf("service name is required")
	}

	// Create resource
	res, err := createResource(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Create meter provider with Prometheus support
	mp, promRegistry, err := createMeterProvider(ctx, res)
	if err != nil {
		return nil, err
	}

	// Set global meter provider
	otel.SetMeterProvider(mp)

	return &Provider{
		MeterProvider:      mp,
		prometheusRegistry: promRegistry,
	}, nil
}

// createResource creates an OpenTelemetry resource with service attributes.
func createResource(ctx context.Context, opts ProviderOptions) (*resource.Resource, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(opts.ServiceName),
			semconv.ServiceVersion(opts.ServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Add custom resource attributes
	if len(opts.ResourceAttrs) > 0 {
		var attrs []attribute.KeyValue
		for k, v := range opts.ResourceAttrs {
			attrs = append(attrs, attribute.String(k, v))
		}
		res, err = resource.Merge(res, resource.NewWithAttributes(semconv.SchemaURL, attrs...))
		if err != nil {
			return nil, fmt.Errorf("failed to add resource attributes: %w", err)
		}
	}

	return res, nil
}

// createMeterProvider creates a meter provider with Prometheus export only.
// It returns the meter provider and a Prometheus registry for HTTP handler.
//
// Parameters:
//   - ctx: context for initialization
//   - res: OpenTelemetry resource with service attributes
//
// Returns:
//   - *metric.MeterProvider: meter provider instance
//   - *promclient.Registry: Prometheus registry for HTTP handler
//   - error: creation error if any
func createMeterProvider(ctx context.Context, res *resource.Resource) (*metric.MeterProvider, *promclient.Registry, error) {
	// Create Prometheus registry and exporter
	promRegistry := promclient.NewRegistry()
	promExporter, err := prometheus.New(
		prometheus.WithRegisterer(promRegistry),
		prometheus.WithoutUnits(),           // Prometheus prefers base units without suffix
		prometheus.WithoutScopeInfo(),       // Remove otel_scope_* labels to reduce cardinality
		prometheus.WithoutCounterSuffixes(), // Remove _total suffix duplication
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	// Create meter provider with Prometheus reader
	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(promExporter),
	)

	return mp, promRegistry, nil
}

// GetPrometheusHandler returns an HTTP handler for the Prometheus metrics endpoint.
// This handler exposes metrics in Prometheus text format at the /metrics path.
//
// Returns:
//   - http.Handler: Prometheus metrics handler
//
// Concurrency:
//   - Safe for concurrent use
func (p *Provider) GetPrometheusHandler() http.Handler {
	if p.prometheusRegistry == nil {
		// Return a no-op handler if Prometheus is not initialized
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("# Prometheus metrics not available\n"))
		})
	}

	return promhttp.HandlerFor(p.prometheusRegistry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// Shutdown gracefully shuts down the metrics provider.
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
	// Create a timeout context for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Shutdown meter provider
	if p.MeterProvider != nil {
		if err := p.MeterProvider.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to shutdown meter provider: %w", err)
		}
	}

	return nil
}
