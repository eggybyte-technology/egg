// Package testingx provides testing helpers and fakes for egg modules.
//
// # Overview
//
// testingx contains small utilities to speed up unit tests, including a
// mock logger with capture capabilities and helpers to construct contexts
// with identity and request metadata.
//
// # Features
//
//   - MockLogger with in-memory capture and assertions
//   - Context helpers for identity and request metadata
//   - Error assertion helpers for core/errors codes
//
// # Usage
//
//	logger := testingx.NewMockLogger(t)
//	ctx := testingx.NewContextWithIdentity(t, &identity.UserInfo{UserID: "u-1"})
//
// # Layer
//
// testingx is an auxiliary module for tests only and depends on core modules.
//
// # Stability
//
// Stable since v0.1.0.
package testingx
