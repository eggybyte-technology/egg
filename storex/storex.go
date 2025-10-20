// Package storex provides storage interfaces and health check registry.
//
// Overview:
//   - Responsibility: Define storage interfaces and manage connection health
//   - Key Types: Store interface, HealthChecker interface, Registry for management
//   - Concurrency Model: All interfaces are safe for concurrent use
//   - Error Semantics: Functions return errors for failure cases
//   - Performance Notes: Designed for high-throughput storage operations
//
// Usage:
//
//	registry := storex.NewRegistry()
//	registry.Register("mysql", mysqlStore)
//	err := registry.Ping(ctx)
package storex

import (
	"context"
	"fmt"
	"time"

	"github.com/eggybyte-technology/egg/core/log"
	"github.com/eggybyte-technology/egg/storex/internal"
)

// Store defines the interface for storage backends.
// Implementations must be safe for concurrent use.
type Store interface {
	// Ping checks if the storage backend is healthy.
	// Returns an error if the backend is unavailable.
	Ping(ctx context.Context) error

	// Close closes the storage connection.
	// Returns an error if the connection cannot be closed gracefully.
	Close() error
}

// HealthChecker defines the interface for health check operations.
type HealthChecker interface {
	// Ping performs a health check on the storage backend.
	Ping(ctx context.Context) error
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
// Returns an error if any backend is unhealthy.
func (r *Registry) Ping(ctx context.Context) error {
	if len(r.stores) == 0 {
		return nil // No stores to check
	}

	// Create a timeout context for health checks
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

// GORMOptions holds configuration for GORM database connections.
type GORMOptions struct {
	DSN             string        // Database connection string
	Driver          string        // Database driver (mysql, postgres, sqlite)
	MaxIdleConns    int           // Maximum number of idle connections
	MaxOpenConns    int           // Maximum number of open connections
	ConnMaxLifetime time.Duration // Maximum connection lifetime
	Logger          log.Logger    // Logger for database operations
}

// NewGORMStore creates a new GORM store with the given options.
func NewGORMStore(opts GORMOptions) (Store, error) {
	return internal.NewGORMStoreFromOptions(internal.GORMOptions{
		DSN:             opts.DSN,
		Driver:          opts.Driver,
		MaxIdleConns:    opts.MaxIdleConns,
		MaxOpenConns:    opts.MaxOpenConns,
		ConnMaxLifetime: opts.ConnMaxLifetime,
		Logger:          opts.Logger,
	})
}

// NewMySQLStore creates a new MySQL store with the given DSN.
func NewMySQLStore(dsn string, logger log.Logger) (Store, error) {
	return NewGORMStore(GORMOptions{
		DSN:    dsn,
		Driver: "mysql",
		Logger: logger,
	})
}

// NewPostgresStore creates a new PostgreSQL store with the given DSN.
func NewPostgresStore(dsn string, logger log.Logger) (Store, error) {
	return NewGORMStore(GORMOptions{
		DSN:    dsn,
		Driver: "postgres",
		Logger: logger,
	})
}

// NewSQLiteStore creates a new SQLite store with the given DSN.
func NewSQLiteStore(dsn string, logger log.Logger) (Store, error) {
	return NewGORMStore(GORMOptions{
		DSN:    dsn,
		Driver: "sqlite",
		Logger: logger,
	})
}
