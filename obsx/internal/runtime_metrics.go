// Package internal provides internal implementation for obsx.
package internal

import (
	"context"
	"runtime"

	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// EnableRuntimeMetrics starts collecting Go runtime metrics.
// It registers metrics for goroutines, GC, and memory usage.
//
// Metrics collected:
//   - process_runtime_go_goroutines: Current number of goroutines
//   - process_runtime_go_gc_count_total: Total number of GC cycles
//   - process_runtime_go_memory_heap_bytes: Heap memory in bytes
//   - process_runtime_go_memory_stack_bytes: Stack memory in bytes
//
// Parameters:
//   - ctx: context for initialization
//   - meterProvider: OpenTelemetry meter provider
//
// Returns:
//   - error: initialization error if any
//
// Concurrency:
//   - Safe to call multiple times (idempotent)
//
// Performance:
//   - Metrics collected on scrape by OpenTelemetry SDK
func EnableRuntimeMetrics(ctx context.Context, meterProvider *sdkmetric.MeterProvider) error {
	meter := meterProvider.Meter("go.eggybyte.com/egg/obsx/runtime")

	// Goroutines gauge
	goroutines, err := meter.Int64ObservableGauge(
		"process_runtime_go_goroutines",
		metric.WithDescription("Number of goroutines that currently exist"),
		metric.WithUnit("{goroutine}"),
	)
	if err != nil {
		return err
	}

	// Memory heap gauge
	heapBytes, err := meter.Int64ObservableGauge(
		"process_runtime_go_memory_heap_bytes",
		metric.WithDescription("Heap memory in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return err
	}

	// Memory stack gauge
	stackBytes, err := meter.Int64ObservableGauge(
		"process_runtime_go_memory_stack_bytes",
		metric.WithDescription("Stack memory in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return err
	}

	// GC count counter
	gcCount, err := meter.Int64ObservableCounter(
		"process_runtime_go_gc_count_total",
		metric.WithDescription("Total number of GC cycles completed"),
		metric.WithUnit("{gc}"),
	)
	if err != nil {
		return err
	}

	// Register callback to collect metrics
	_, err = meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			// Collect goroutines
			observer.ObserveInt64(goroutines, int64(runtime.NumGoroutine()))

			// Collect memory stats
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			observer.ObserveInt64(heapBytes, int64(m.HeapAlloc))
			observer.ObserveInt64(stackBytes, int64(m.StackInuse))
			observer.ObserveInt64(gcCount, int64(m.NumGC))

			return nil
		},
		goroutines,
		heapBytes,
		stackBytes,
		gcCount,
	)

	return err
}
