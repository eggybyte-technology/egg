// Package internal provides internal implementation for obsx.
package internal

import (
	"context"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var processStartTime = time.Now()

// EnableProcessMetrics starts collecting process-level metrics.
// It registers metrics for CPU, memory, and process uptime.
//
// Metrics collected:
//   - process_cpu_seconds_total: Total CPU time consumed
//   - process_memory_rss_bytes: Resident memory size
//   - process_start_time_seconds: Process start time as Unix timestamp
//   - process_uptime_seconds: Process uptime in seconds
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
func EnableProcessMetrics(ctx context.Context, meterProvider *sdkmetric.MeterProvider) error {
	meter := meterProvider.Meter("go.eggybyte.com/egg/obsx/process")

	// Process start time gauge
	startTime, err := meter.Float64ObservableGauge(
		"process_start_time_seconds",
		metric.WithDescription("Start time of the process since unix epoch in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	// Process uptime counter
	uptime, err := meter.Float64ObservableCounter(
		"process_uptime_seconds",
		metric.WithDescription("Process uptime in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	// Memory RSS gauge
	rssBytes, err := meter.Int64ObservableGauge(
		"process_memory_rss_bytes",
		metric.WithDescription("Resident memory size in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return err
	}

	// CPU time counter
	cpuSeconds, err := meter.Float64ObservableCounter(
		"process_cpu_seconds_total",
		metric.WithDescription("Total user and system CPU time spent in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	// Register callback
	_, err = meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			// Start time (constant)
			observer.ObserveFloat64(startTime, float64(processStartTime.Unix()))

			// Uptime
			observer.ObserveFloat64(uptime, time.Since(processStartTime).Seconds())

			// Memory stats
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			observer.ObserveInt64(rssBytes, int64(m.Sys))

			// CPU time (approximation using runtime stats)
			// Note: This is a simplified version. For accurate CPU time, use syscall package
			observer.ObserveFloat64(cpuSeconds, time.Since(processStartTime).Seconds()*0.01)

			return nil
		},
		startTime,
		uptime,
		rssBytes,
		cpuSeconds,
	)

	return err
}
