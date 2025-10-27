// Package internal provides internal implementation details for configx.
package internal

import (
	"context"
	"time"

	"go.eggybyte.com/egg/core/log"
)

// Source describes a configuration source that can load and watch for updates.
// Implementations must be thread-safe and honor context cancellation.
type Source interface {
	// Load reads the current configuration snapshot for initial merge.
	// Returns a map of key-value pairs.
	Load(ctx context.Context) (map[string]string, error)

	// Watch starts monitoring for updates and publishes snapshots via the returned channel.
	// The channel should be closed when the context is cancelled to avoid goroutine leaks.
	Watch(ctx context.Context) (<-chan map[string]string, error)
}

// Manager manages multiple configuration sources and provides unified access.
// The manager merges configurations with later sources taking precedence.
type Manager interface {
	// Snapshot returns a copy of the current merged configuration.
	Snapshot() map[string]string

	// Value returns the value for a key and whether it exists.
	Value(key string) (string, bool)

	// Bind decodes the configuration into a struct with env tags and default values.
	// Supports hot reloading via callback when configuration changes.
	Bind(target any, opts ...BindOption) error

	// OnUpdate subscribes to configuration update events.
	// Returns an unsubscribe function.
	OnUpdate(fn func(snapshot map[string]string)) (unsubscribe func())
}

// Options holds configuration for the manager.
type Options struct {
	Logger   log.Logger    // Logger for configuration operations
	Sources  []Source      // Configuration sources (later sources override earlier ones)
	Debounce time.Duration // Debounce duration for updates (default: 200ms)
}

// BindOption configures binding behavior.
type BindOption interface {
	apply(*bindConfig)
}

type bindConfig struct {
	onUpdate func()
}

type bindOptionFunc func(*bindConfig)

func (f bindOptionFunc) apply(cfg *bindConfig) {
	f(cfg)
}

// WithUpdateCallback sets a callback to be invoked when configuration changes.
func WithUpdateCallback(fn func()) BindOption {
	return bindOptionFunc(func(cfg *bindConfig) {
		cfg.onUpdate = fn
	})
}
