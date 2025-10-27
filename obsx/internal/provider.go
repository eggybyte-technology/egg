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
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// ProviderOptions holds configuration for the observability provider.
type ProviderOptions struct {
	ServiceName       string
	ServiceVersion    string
	OTLPEndpoint      string
	ResourceAttrs     map[string]string
	TraceSamplerRatio float64
}

// Provider manages OpenTelemetry tracing and metrics providers.
type Provider struct {
	TracerProvider     *sdktrace.TracerProvider
	MeterProvider      *metric.MeterProvider
	prometheusRegistry *promclient.Registry
}

// NewProvider creates a new observability provider with the given options.
func NewProvider(ctx context.Context, opts ProviderOptions) (*Provider, error) {
	if opts.ServiceName == "" {
		return nil, fmt.Errorf("service name is required")
	}

	// Set default sampling ratio
	samplerRatio := opts.TraceSamplerRatio
	if samplerRatio <= 0 {
		samplerRatio = 0.1 // Default 10% sampling
	}

	// Create resource
	res, err := createResource(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Create trace provider
	tp, err := createTracerProvider(ctx, res, opts.OTLPEndpoint, samplerRatio)
	if err != nil {
		return nil, err
	}

	// Create meter provider with Prometheus support
	mp, promRegistry, err := createMeterProvider(ctx, res, opts.OTLPEndpoint)
	if err != nil {
		return nil, err
	}

	// Set global providers
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)

	return &Provider{
		TracerProvider:     tp,
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

// createTracerProvider creates a tracer provider with optional OTLP export.
func createTracerProvider(ctx context.Context, res *resource.Resource, otlpEndpoint string, samplerRatio float64) (*sdktrace.TracerProvider, error) {
	// Create trace exporter if OTLP endpoint is provided
	var traceExporter sdktrace.SpanExporter
	if otlpEndpoint != "" {
		var err error
		traceExporter, err = otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(otlpEndpoint),
			otlptracegrpc.WithInsecure(), // In production, use proper TLS
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create trace exporter: %w", err)
		}
	}

	// Create tracer provider
	var tp *sdktrace.TracerProvider
	if traceExporter != nil {
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sdktrace.TraceIDRatioBased(samplerRatio)),
			sdktrace.WithBatcher(traceExporter),
		)
	} else {
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sdktrace.TraceIDRatioBased(samplerRatio)),
		)
	}

	return tp, nil
}

// createMeterProvider creates a meter provider with Prometheus and optional OTLP export.
// It returns the meter provider and a Prometheus registry for HTTP handler.
func createMeterProvider(ctx context.Context, res *resource.Resource, otlpEndpoint string) (*metric.MeterProvider, *promclient.Registry, error) {
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

	// Build meter provider options
	meterOpts := []metric.Option{
		metric.WithResource(res),
		metric.WithReader(promExporter),
	}

	// Create OTLP metric exporter if endpoint is provided
	if otlpEndpoint != "" {
		otlpExporter, err := otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(otlpEndpoint),
			otlpmetricgrpc.WithInsecure(), // In production, use proper TLS
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create otlp metric exporter: %w", err)
		}
		meterOpts = append(meterOpts, metric.WithReader(metric.NewPeriodicReader(otlpExporter)))
	}

	// Create meter provider with all readers
	mp := metric.NewMeterProvider(meterOpts...)

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

// Shutdown gracefully shuts down the provider.
func (p *Provider) Shutdown(ctx context.Context) error {
	// Create a timeout context for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var errors []error

	// Shutdown tracer provider
	if p.TracerProvider != nil {
		if err := p.TracerProvider.Shutdown(shutdownCtx); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown tracer provider: %w", err))
		}
	}

	// Shutdown meter provider
	if p.MeterProvider != nil {
		if err := p.MeterProvider.Shutdown(shutdownCtx); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown meter provider: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}
