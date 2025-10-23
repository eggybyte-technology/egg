// Package clientx provides Connect HTTP client construction with
// retry, circuit breaker, idempotency headers, and timeouts.
//
// # Overview
//
// clientx offers production-grade HTTP clients suitable for Connect-based
// services. It includes exponential backoff retry, optional circuit breaker,
// and request timeouts while keeping APIs minimal and composable.
//
// # Features
//
//   - Exponential backoff retries for transient 5xx errors
//   - Optional circuit breaker to prevent cascade failures
//   - Request timeouts and idempotency key injection
//   - Generic helper for constructing typed Connect clients
//
// # Usage
//
//	client := clientx.NewHTTPClient("https://api.example.com",
//		clientx.WithTimeout(5*time.Second),
//		clientx.WithRetry(3),
//		clientx.WithCircuitBreaker(true),
//	)
//
// # Layer
//
// clientx belongs to Layer 3 (L3) and depends on core/log, connectx (optionally).
//
// # Stability
//
// Stable since v0.1.0. Minor versions may introduce backward-compatible improvements.
package clientx
