// Package obsx initializes OpenTelemetry tracing and metrics providers
// for the egg microservice framework.
//
// # Overview
//
// obsx constructs and configures OpenTelemetry providers (tracer, meter)
// with sensible defaults and optional OTLP exporters. It exposes a single
// Provider that manages lifecycle and integrates with the global OTel.
//
// # Features
//
//   - Tracer and meter providers with resource attributes
//   - Optional OTLP exporters for traces and metrics
//   - Configurable sampling ratio and runtime metrics
//   - Graceful shutdown with bounded timeouts
//
// # Usage
//
//	provider, err := obsx.NewProvider(ctx, obsx.Options{
//		ServiceName:    "user-service",
//		ServiceVersion: "1.0.0",
//		OTLPEndpoint:   "otel-collector:4317",
//	})
//	if err != nil { panic(err) }
//	defer provider.Shutdown(ctx)
//
// # Layer
//
// obsx belongs to Layer 2 (L2) and depends on core only.
//
// # Stability
//
// Stable since v0.1.0.
package obsx
