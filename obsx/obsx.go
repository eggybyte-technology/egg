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
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
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
type Provider struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *metric.MeterProvider
}

// NewProvider creates a new observability provider with the given options.
// The provider must be shut down when no longer needed.
func NewProvider(ctx context.Context, opts Options) (*Provider, error) {
	if opts.ServiceName == "" {
		return nil, fmt.Errorf("service name is required")
	}

	// Set default sampling ratio
	samplerRatio := opts.TraceSamplerRatio
	if samplerRatio <= 0 {
		samplerRatio = 0.1 // Default 10% sampling
	}

	// Create resource
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

	// Create trace exporter if OTLP endpoint is provided
	var traceExporter sdktrace.SpanExporter
	if opts.OTLPEndpoint != "" {
		traceExporter, err = otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(opts.OTLPEndpoint),
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

	// Create metric exporter if OTLP endpoint is provided
	var metricExporter metric.Exporter
	if opts.OTLPEndpoint != "" {
		metricExporter, err = otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(opts.OTLPEndpoint),
			otlpmetricgrpc.WithInsecure(), // In production, use proper TLS
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create metric exporter: %w", err)
		}
	}

	// Create meter provider
	var mp *metric.MeterProvider
	if metricExporter != nil {
		mp = metric.NewMeterProvider(
			metric.WithResource(res),
			metric.WithReader(metric.NewPeriodicReader(metricExporter)),
		)
	} else {
		mp = metric.NewMeterProvider(
			metric.WithResource(res),
		)
	}

	// Set global providers
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)

	return &Provider{
		TracerProvider: tp,
		MeterProvider:  mp,
	}, nil
}

// Shutdown gracefully shuts down the provider.
// This should be called when the application is shutting down.
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
