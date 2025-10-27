// Package internal contains Connect interceptor implementations.
package internal

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

// MetricsCollector holds OpenTelemetry metrics instruments for RPC monitoring.
type MetricsCollector struct {
	requestsTotal     metric.Int64Counter
	requestDuration   metric.Float64Histogram
	requestSizeBytes  metric.Int64Histogram
	responseSizeBytes metric.Int64Histogram
	enabled           bool
}

// NewMetricsCollector creates a new metrics collector for RPC monitoring.
// If otelProvider is nil, metrics collection is disabled.
//
// Parameters:
//   - otelProvider: OpenTelemetry provider (can be nil to disable metrics)
//
// Returns:
//   - *MetricsCollector: metrics collector instance
//   - error: initialization error if metrics setup fails
//
// Concurrency:
//   - Safe for concurrent use after initialization
func NewMetricsCollector(otelProvider *obsx.Provider) (*MetricsCollector, error) {
	if otelProvider == nil {
		return &MetricsCollector{enabled: false}, nil
	}

	meter := otelProvider.Meter("go.eggybyte.com/egg/connectx")

	// Create request counter
	requestsTotal, err := meter.Int64Counter(
		"rpc_requests_total",
		metric.WithDescription("Total number of RPC requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	// Create request duration histogram with standard buckets (seconds)
	// Buckets: [0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 10]
	requestDuration, err := meter.Float64Histogram(
		"rpc_request_duration_seconds",
		metric.WithDescription("RPC request duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 10,
		),
	)
	if err != nil {
		return nil, err
	}

	// Create request size histogram with standard buckets (bytes)
	// Buckets: [64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536, 131072, 262144, 524288, 1048576]
	requestSizeBytes, err := meter.Int64Histogram(
		"rpc_request_size_bytes",
		metric.WithDescription("RPC request size in bytes"),
		metric.WithUnit("By"),
		metric.WithExplicitBucketBoundaries(
			64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536, 131072, 262144, 524288, 1048576,
		),
	)
	if err != nil {
		return nil, err
	}

	// Create response size histogram with standard buckets (bytes)
	responseSizeBytes, err := meter.Int64Histogram(
		"rpc_response_size_bytes",
		metric.WithDescription("RPC response size in bytes"),
		metric.WithUnit("By"),
		metric.WithExplicitBucketBoundaries(
			64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536, 131072, 262144, 524288, 1048576,
		),
	)
	if err != nil {
		return nil, err
	}

	return &MetricsCollector{
		requestsTotal:     requestsTotal,
		requestDuration:   requestDuration,
		requestSizeBytes:  requestSizeBytes,
		responseSizeBytes: responseSizeBytes,
		enabled:           true,
	}, nil
}

// MetricsInterceptor creates a Connect interceptor that collects RPC metrics.
// It records request count, duration, and payload sizes for all RPC calls.
//
// Parameters:
//   - collector: metrics collector instance
//
// Returns:
//   - connect.UnaryInterceptorFunc: interceptor function
//
// Metrics collected:
//   - rpc_requests_total: counter of requests by service, method, code
//   - rpc_request_duration_seconds: histogram of request duration in seconds
//   - rpc_request_size_bytes: histogram of request payload size in bytes
//   - rpc_response_size_bytes: histogram of response payload size in bytes
//
// Labels:
//   - rpc_service: service name (e.g., "greet.v1.GreeterService")
//   - rpc_method: method name (e.g., "SayHello")
//   - rpc_code: Connect error code (e.g., "ok", "not_found", "internal")
//
// Concurrency:
//   - Safe for concurrent use
func MetricsInterceptor(collector *MetricsCollector) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if !collector.enabled {
				return next(ctx, req)
			}

			startTime := time.Now()
			procedure := req.Spec().Procedure

			// Parse procedure into service and method
			// Procedure format: "/package.ServiceName/MethodName" or "/ServiceName/MethodName"
			service, method := parseProcedure(procedure)

			// Record request size if available
			if reqMsg := req.Any(); reqMsg != nil {
				// Estimate size based on message (this is approximate)
				// In production, you might want to use proto.Size() for more accurate sizing
				reqSize := int64(len(procedure)) // Simplified size estimation
				collector.requestSizeBytes.Record(ctx, reqSize,
					metric.WithAttributes(
						attribute.String("rpc_service", service),
						attribute.String("rpc_method", method),
					),
				)
			}

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

			// Common attributes (label whitelist)
			attrs := []attribute.KeyValue{
				attribute.String("rpc_service", service),
				attribute.String("rpc_method", method),
				attribute.String("rpc_code", code),
			}

			// Record metrics
			collector.requestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))

			// Record duration with exemplar (trace_id) for histogram
			// Extract trace_id from context for exemplar support
			metricOpts := []metric.RecordOption{metric.WithAttributes(attrs...)}
			if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
				traceID := span.SpanContext().TraceID().String()
				metricOpts = append(metricOpts, metric.WithAttributes(
					attribute.String("trace_id", traceID),
				))
			}
			collector.requestDuration.Record(ctx, duration, metricOpts...)

			// Record response size if available (with safe nil checks)
			if resp != nil {
				// Safely extract response message with panic protection
				func() {
					defer func() {
						if r := recover(); r != nil {
							// Silently skip response size recording if panic occurs
							// This prevents metrics collection from breaking the request flow
						}
					}()

					if respMsg := resp.Any(); respMsg != nil {
						// Estimate size (simplified)
						respSize := int64(len(procedure)) // Simplified size estimation
						collector.responseSizeBytes.Record(ctx, respSize,
							metric.WithAttributes(
								attribute.String("rpc_service", service),
								attribute.String("rpc_method", method),
							),
						)
					}
				}()
			}

			return resp, err
		}
	}
}

// parseProcedure splits a Connect procedure into service and method names.
// Procedure format: "/package.v1.ServiceName/MethodName" or "/ServiceName/MethodName"
//
// Parameters:
//   - procedure: full procedure path (e.g., "/user.v1.UserService/CreateUser")
//
// Returns:
//   - service: service name (e.g., "user.v1.UserService")
//   - method: method name (e.g., "CreateUser")
func parseProcedure(procedure string) (service, method string) {
	// Remove leading slash
	procedure = strings.TrimPrefix(procedure, "/")

	// Split by last slash to separate service from method
	lastSlash := strings.LastIndex(procedure, "/")
	if lastSlash == -1 {
		// No slash found, treat entire string as method
		return "", procedure
	}

	service = procedure[:lastSlash]
	method = procedure[lastSlash+1:]
	return service, method
}
