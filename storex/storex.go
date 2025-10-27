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
	"time"

	"go.eggybyte.com/egg/core/log"
	"go.eggybyte.com/egg/storex/internal"
	"gorm.io/gorm"
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

// GORMStore defines the interface for GORM-backed storage.
// This extends Store with GORM-specific functionality.
type GORMStore interface {
	Store
	// GetDB returns the underlying GORM database instance.
	// The returned *gorm.DB is safe for concurrent use.
	GetDB() *gorm.DB
}

// HealthChecker defines the interface for health check operations.
type HealthChecker interface {
	// Ping performs a health check on the storage backend.
	Ping(ctx context.Context) error
}

// Registry manages multiple storage connections and their health.
// This is a thin wrapper around internal.Registry.
type Registry struct {
	impl *internal.Registry
}

// NewRegistry creates a new storage registry.
func NewRegistry() *Registry {
	return &Registry{impl: internal.NewRegistry()}
}

// Register registers a storage backend with the given name.
func (r *Registry) Register(name string, store Store) error {
	return r.impl.Register(name, store)
}

// Unregister removes a storage backend from the registry.
func (r *Registry) Unregister(name string) error {
	return r.impl.Unregister(name)
}

// Ping performs health checks on all registered storage backends.
func (r *Registry) Ping(ctx context.Context) error {
	return r.impl.Ping(ctx)
}

// Close closes all registered storage connections.
func (r *Registry) Close() error {
	return r.impl.Close()
}

// List returns the names of all registered stores.
func (r *Registry) List() []string {
	return r.impl.List()
}

// Get returns a registered store by name.
func (r *Registry) Get(name string) (Store, bool) {
	return r.impl.Get(name)
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
// Returns a GORMStore that provides access to the underlying *gorm.DB.
func NewGORMStore(opts GORMOptions) (GORMStore, error) {
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
func NewMySQLStore(dsn string, logger log.Logger) (GORMStore, error) {
	return NewGORMStore(GORMOptions{
		DSN:    dsn,
		Driver: "mysql",
		Logger: logger,
	})
}

// NewPostgresStore creates a new PostgreSQL store with the given DSN.
func NewPostgresStore(dsn string, logger log.Logger) (GORMStore, error) {
	return NewGORMStore(GORMOptions{
		DSN:    dsn,
		Driver: "postgres",
		Logger: logger,
	})
}

// NewSQLiteStore creates a new SQLite store with the given DSN.
func NewSQLiteStore(dsn string, logger log.Logger) (GORMStore, error) {
	return NewGORMStore(GORMOptions{
		DSN:    dsn,
		Driver: "sqlite",
		Logger: logger,
	})
}
