// Package configx provides unified configuration management with hot reload
// for the egg microservice framework.
//
// # Overview
//
// configx aggregates multiple configuration sources (env, files, Kubernetes
// ConfigMap, etc.), merges them deterministically, and provides type-safe
// binding into structs with defaults. It also supports debounced hot updates
// and subscription callbacks.
//
// # Features
//
//   - Multiple sources with last-wins merge semantics
//   - Type-safe struct binding via env/default tags
//   - Debounced hot updates with subscription callbacks
//   - Thread-safe reads and update notifications
//   - Minimal footprint and production-grade behavior
//
// # Usage
//
//	sources := []configx.Source{
//		configx.NewEnvSource(configx.EnvOptions{}),
//	}
//	mgr, err := configx.NewManager(ctx, configx.Options{
//		Logger:   logger,
//		Sources:  sources,
//		Debounce: 200 * time.Millisecond,
//	})
//	if err != nil { panic(err) }
//
//	var cfg AppConfig
//	if err := mgr.Bind(&cfg); err != nil { panic(err) }
//
// # Layer
//
// configx belongs to Layer 2 (L2) and depends on core and logx.
//
// # Stability
//
// Stable since v0.1.0. Backward-compatible API changes may occur with minor versions.
package configx
