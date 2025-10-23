// Package connectx provides the unified Connect-RPC interceptor stack
// for the egg microservice framework.
//
// # Overview
//
// connectx defines a composable set of Connect interceptors for timeouts,
// logging, metrics, identity injection, and structured error mapping.
// It ensures consistent RPC observability and governance across egg-based
// microservices with minimal business intrusion.
//
// # Features
//
//   - Per-RPC timeout control (global default with optional overrides)
//   - Unified structured logging with request/trace correlation
//   - Error mapping between core/errors and Connect/HTTP codes
//   - Identity extraction from headers and context propagation
//   - Extensible interceptor chaining (platform + business layers)
//   - Optional payload accounting and slow-request logging
//
// # Usage
//
//	mux := http.NewServeMux()
//	path, handler := myv1connect.NewMyServiceHandler(
//		myHandler,
//		connect.WithInterceptors(connectx.DefaultInterceptors(connectx.Options{
//			Logger:            logger,
//			SlowRequestMillis: 1000,
//			PayloadAccounting: true,
//		})...),
//	)
//	mux.Handle(path, handler)
//
// # Layer
//
// connectx belongs to Layer 3 (L3) and may depend on: core, logx, obsx, configx.
//
// # Stability
//
// Stable since v0.1.0. Backward-compatible API changes only occur with a minor version bump.
package connectx
