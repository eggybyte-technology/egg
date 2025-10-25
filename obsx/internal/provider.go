// Package internal provides internal implementation for the obsx package.
package internal

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
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *metric.MeterProvider
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

	// Create meter provider
	mp, err := createMeterProvider(ctx, res, opts.OTLPEndpoint)
	if err != nil {
		return nil, err
	}

	// Set global providers
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)

	return &Provider{
		TracerProvider: tp,
		MeterProvider:  mp,
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

// createMeterProvider creates a meter provider with optional OTLP export.
func createMeterProvider(ctx context.Context, res *resource.Resource, otlpEndpoint string) (*metric.MeterProvider, error) {
	// Create metric exporter if OTLP endpoint is provided
	var metricExporter metric.Exporter
	if otlpEndpoint != "" {
		var err error
		metricExporter, err = otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(otlpEndpoint),
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

	return mp, nil
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

