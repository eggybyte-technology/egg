// Package runtimex provides runtime lifecycle orchestration for services,
// including HTTP/RPC servers, health, metrics, and graceful shutdown.
//
// # Overview
//
// runtimex offers production-grade building blocks to start and stop
// servers and background services safely. It centralizes lifecycle
// management and exposes a small, composable API.
//
// # Features
//
//   - Unified lifecycle management with graceful shutdown
//   - HTTP server wiring (H2C optional), health and metrics endpoints
//   - Pluggable service interface for background workers
//   - Structured logging hooks for startup/shutdown events
//
// # Usage
//
//	err := runtimex.Run(ctx, []runtimex.Service{svc}, runtimex.Options{
//		Logger: logger,
//		HTTP:   &runtimex.HTTPOptions{Addr: ":8080", Mux: mux, H2C: true},
//		Health: &runtimex.Endpoint{Addr: ":8081"},
//		Metrics: &runtimex.Endpoint{Addr: ":9091"},
//	})
//
// # Layer
//
// runtimex belongs to Layer 3 (L3) and depends on core/log and obsx.
//
// # Stability
//
// Stable since v0.1.0.
package runtimex
