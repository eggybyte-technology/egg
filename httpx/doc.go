// Package httpx provides HTTP helpers for binding, validation,
// security headers, CORS, and standard error responses.
//
// # Overview
//
// httpx contains thin utilities to keep HTTP adapters minimal and consistent.
// It helps decode/validate JSON requests, write structured errors, and apply
// sensible default security headers and CORS policies.
//
// # Features
//
//   - Bind and validate JSON requests (validator integration)
//   - Standard JSON error responses (404/405/custom)
//   - Security headers middleware with sane defaults
//   - CORS middleware with configurable options
//
// # Usage
//
//	var req CreateUserRequest
//	if err := httpx.BindAndValidate(r, &req); err != nil {
//		_ = httpx.WriteError(w, err, http.StatusBadRequest)
//		return
//	}
//
// # Layer
//
// httpx belongs to Layer 2 (L2) and depends on core/log minimally.
//
// # Stability
//
// Stable since v0.1.0. API evolves conservatively.
package httpx
