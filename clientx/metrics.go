// Package clientx provides client-side metrics collection.
package clientx

import (
	"context"
	"strings"
	"time"

	"connectrpc.com/connect"
	"go.eggybyte.com/egg/obsx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// ClientMetricsCollector holds OpenTelemetry metrics instruments for client-side RPC monitoring.
type ClientMetricsCollector struct {
	requestsTotal   metric.Int64Counter
	requestDuration metric.Float64Histogram
	enabled         bool
}

// NewClientMetricsCollector creates a new metrics collector for client-side RPC monitoring.
// If otelProvider is nil, metrics collection is disabled.
//
// Parameters:
//   - otelProvider: OpenTelemetry provider (can be nil to disable metrics)
//
// Returns:
//   - *ClientMetricsCollector: metrics collector instance
//   - error: initialization error if metrics setup fails
//
// Concurrency:
//   - Safe for concurrent use after initialization
func NewClientMetricsCollector(otelProvider *obsx.Provider) (*ClientMetricsCollector, error) {
	if otelProvider == nil {
		return &ClientMetricsCollector{enabled: false}, nil
	}

	meter := otelProvider.Meter("go.eggybyte.com/egg/clientx")

	// Create client request counter
	requestsTotal, err := meter.Int64Counter(
		"rpc_client_requests_total",
		metric.WithDescription("Total number of client RPC requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	// Create client request duration histogram
	requestDuration, err := meter.Float64Histogram(
		"rpc_client_request_duration_seconds",
		metric.WithDescription("Client RPC request duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 10,
		),
	)
	if err != nil {
		return nil, err
	}

	return &ClientMetricsCollector{
		requestsTotal:   requestsTotal,
		requestDuration: requestDuration,
		enabled:         true,
	}, nil
}

// ClientMetricsInterceptor creates a Connect client interceptor that collects outbound RPC metrics.
// It records request count and duration for all outbound RPC calls.
//
// Parameters:
//   - collector: client metrics collector instance
//
// Returns:
//   - connect.UnaryInterceptorFunc: client interceptor function
//
// Metrics collected:
//   - rpc_client_requests_total: counter of outbound requests by service, method, code
//   - rpc_client_request_duration_seconds: histogram of outbound request duration
//
// Labels:
//   - rpc_service: target service name
//   - rpc_method: target method name
//   - rpc_code: Connect error code
//
// Concurrency:
//   - Safe for concurrent use
func ClientMetricsInterceptor(collector *ClientMetricsCollector) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if !collector.enabled {
				return next(ctx, req)
			}

			startTime := time.Now()
			procedure := req.Spec().Procedure

			// Parse procedure into service and method
			service, method := parseClientProcedure(procedure)

			// Call next handler
			resp, err := next(ctx, req)

			// Calculate duration in seconds
			duration := time.Since(startTime).Seconds()

			// Determine error code
			var code string
			if err != nil {
				if connectErr, ok := err.(*connect.Error); ok {
					code = connectErr.Code().String()
				} else {
					code = "unknown"
				}
			} else {
				code = "ok"
			}

			// Common attributes
			attrs := []attribute.KeyValue{
				attribute.String("rpc_service", service),
				attribute.String("rpc_method", method),
				attribute.String("rpc_code", code),
			}

			// Record metrics
			collector.requestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))

			// Record duration with exemplar (trace_id) for histogram
			metricOpts := []metric.RecordOption{metric.WithAttributes(attrs...)}
			if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
				traceID := span.SpanContext().TraceID().String()
				metricOpts = append(metricOpts, metric.WithAttributes(
					attribute.String("trace_id", traceID),
				))
			}
			collector.requestDuration.Record(ctx, duration, metricOpts...)

			return resp, err
		}
	}
}

// parseClientProcedure splits a Connect procedure into service and method names.
// Same logic as server-side but kept separate for clarity.
func parseClientProcedure(procedure string) (service, method string) {
	procedure = strings.TrimPrefix(procedure, "/")
	lastSlash := strings.LastIndex(procedure, "/")
	if lastSlash == -1 {
		return "", procedure
	}
	service = procedure[:lastSlash]
	method = procedure[lastSlash+1:]
	return service, method
}
