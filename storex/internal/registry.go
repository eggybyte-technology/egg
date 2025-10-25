// Package internal provides internal implementation details for storex.
package internal

import (
	"context"
	"fmt"
	"time"
)

// Store defines the interface for storage backends.
type Store interface {
	Ping(ctx context.Context) error
	Close() error
}

// Registry manages multiple storage connections and their health.
type Registry struct {
	stores map[string]Store
}

// NewRegistry creates a new storage registry.
func NewRegistry() *Registry {
	return &Registry{
		stores: make(map[string]Store),
	}
}

// Register registers a storage backend with the given name.
func (r *Registry) Register(name string, store Store) error {
	if name == "" {
		return fmt.Errorf("store name is required")
	}
	if store == nil {
		return fmt.Errorf("store cannot be nil")
	}
	if _, exists := r.stores[name]; exists {
		return fmt.Errorf("store %s already registered", name)
	}

	r.stores[name] = store
	return nil
}

// Unregister removes a storage backend from the registry.
func (r *Registry) Unregister(name string) error {
	if _, exists := r.stores[name]; !exists {
		return fmt.Errorf("store %s not found", name)
	}

	delete(r.stores, name)
	return nil
}

// Ping performs health checks on all registered storage backends.
func (r *Registry) Ping(ctx context.Context) error {
	if len(r.stores) == 0 {
		return nil
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var errors []error
	for name, store := range r.stores {
		if err := store.Ping(pingCtx); err != nil {
			errors = append(errors, fmt.Errorf("store %s ping failed: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("health check failed: %v", errors)
	}

	return nil
}

// Close closes all registered storage connections.
func (r *Registry) Close() error {
	var errors []error
	for name, store := range r.stores {
		if err := store.Close(); err != nil {
			errors = append(errors, fmt.Errorf("store %s close failed: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("close failed: %v", errors)
	}

	return nil
}

// List returns the names of all registered stores.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.stores))
	for name := range r.stores {
		names = append(names, name)
	}
	return names
}

// Get returns a registered store by name.
func (r *Registry) Get(name string) (Store, bool) {
	store, exists := r.stores[name]
	return store, exists
}


