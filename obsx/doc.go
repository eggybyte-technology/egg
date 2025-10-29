// Package obsx provides Prometheus-based metrics collection for Go applications
// in the egg microservice framework.
//
// # Overview
//
// obsx constructs and configures OpenTelemetry metrics provider with Prometheus
// export. It provides a lightweight, focused metrics solution without distributed
// tracing overhead. The Provider manages lifecycle and exposes /metrics endpoint
// for Prometheus scraping.
//
// # Features
//
//   - Meter provider with Prometheus export only (no remote push)
//   - Runtime metrics (goroutines, GC, memory)
//   - Process metrics (CPU, RSS, uptime)
//   - Database connection pool metrics (GORM/sql.DB)
//   - Graceful shutdown with bounded timeouts
//
// # Usage
//
//	provider, err := obsx.NewProvider(ctx, obsx.Options{
//		ServiceName:    "user-service",
//		ServiceVersion: "1.0.0",
//	})
//	if err != nil { panic(err) }
//
//	// Enable additional metrics
//	provider.EnableRuntimeMetrics(ctx)
//	provider.EnableProcessMetrics(ctx)
//
//	// Expose metrics endpoint
//	http.Handle("/metrics", provider.PrometheusHandler())
//
//	defer provider.Shutdown(ctx)
//
// # Layer
//
// obsx belongs to Layer 2 (L2) and depends on core only.
//
// # Stability
//
// Stable since v0.1.0. Tracing support removed in v0.3.0.
package obsx
