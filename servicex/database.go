// Package servicex provides database integration for services.
//
// Overview:
//   - Responsibility: Simplify database initialization and management
//   - Key Types: DatabaseConfig for configuration, DB accessor in App
//   - Concurrency Model: Database connections are safe for concurrent use
//   - Error Semantics: Returns wrapped errors for all failure cases
//   - Performance Notes: Connection pooling is automatically configured
//
// Usage:
//
//	servicex.Run(ctx,
//	    servicex.WithDatabase(servicex.DatabaseConfig{
//	        Driver: "mysql",
//	        DSN:    "user:pass@tcp(localhost:3306)/dbname",
//	    }),
//	    servicex.WithAutoMigrate(&User{}, &Post{}),
//	)
package servicex

import (
	"context"
	"fmt"
	"time"

	"github.com/eggybyte-technology/egg/core/log"
	"github.com/eggybyte-technology/egg/storex"
	"gorm.io/gorm"
)

// WithDatabase enables database support for the service.
// The database connection will be initialized during service startup
// and closed automatically during graceful shutdown.
//
// Parameters:
//   - cfg: Database configuration.
//
// Returns:
//   - Option: Configuration option for servicex.Run.
//
// Example:
//
//	servicex.Run(ctx,
//	    servicex.WithDatabase(servicex.DatabaseConfig{
//	        Driver: "mysql",
//	        DSN:    "user:pass@tcp(localhost:3306)/db?parseTime=true",
//	    }),
//	)
func WithDatabase(cfg DatabaseConfig) Option {
	return func(c *serviceConfig) {
		c.dbConfig = &cfg
	}
}

// WithAutoMigrate specifies database models to auto-migrate during startup.
// Migration happens after database connection is established.
//
// Parameters:
//   - models: Variadic list of model pointers (e.g., &User{}, &Post{}).
//
// Returns:
//   - Option: Configuration option for servicex.Run.
//
// Example:
//
//	servicex.Run(ctx,
//	    servicex.WithDatabase(...),
//	    servicex.WithAutoMigrate(&User{}, &Post{}, &Comment{}),
//	)
//
// Concurrency:
//   - Auto-migration runs synchronously during startup before handlers are registered.
func WithAutoMigrate(models ...any) Option {
	return func(c *serviceConfig) {
		c.autoMigrateModels = models
	}
}

// initDatabase initializes the database connection based on the provided configuration.
//
// Parameters:
//   - ctx: Context for timeout control during connection establishment.
//   - cfg: Database configuration.
//   - logger: Logger for database operations.
//
// Returns:
//   - *gorm.DB: GORM database instance.
//   - error: Error if connection fails or configuration is invalid.
//
// Error Cases:
//   - Missing or invalid driver/DSN
//   - Connection timeout
//   - Database ping failure
//
// Performance:
//   - Connection pooling is automatically configured based on provided parameters.
func initDatabase(ctx context.Context, cfg *DatabaseConfig, logger log.Logger) (*gorm.DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database config is nil")
	}

	if cfg.Driver == "" {
		return nil, fmt.Errorf("database driver is required")
	}

	if cfg.DSN == "" {
		return nil, fmt.Errorf("database DSN is required")
	}

	// Set defaults
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = 10
	}
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = 100
	}
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = time.Hour
	}
	if cfg.PingTimeout == 0 {
		cfg.PingTimeout = 5 * time.Second
	}

	logger.Info("initializing database connection",
		log.Str("driver", cfg.Driver),
		log.Int("max_idle", cfg.MaxIdleConns),
		log.Int("max_open", cfg.MaxOpenConns),
	)

	// Create store using storex
	store, err := storex.NewGORMStore(storex.GORMOptions{
		DSN:             cfg.DSN,
		Driver:          cfg.Driver,
		MaxIdleConns:    cfg.MaxIdleConns,
		MaxOpenConns:    cfg.MaxOpenConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		Logger:          logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create database store: %w", err)
	}

	// Test connection with timeout
	pingCtx, cancel := context.WithTimeout(ctx, cfg.PingTimeout)
	defer cancel()

	if err := store.Ping(pingCtx); err != nil {
		store.Close() // Clean up on failure
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	logger.Info("database connection established successfully")

	// Extract GORM DB instance
	return store.GetDB(), nil
}

// autoMigrate runs GORM auto-migration for the provided models.
//
// Parameters:
//   - db: GORM database instance.
//   - logger: Logger for migration operations.
//   - models: Slice of model pointers to migrate.
//
// Returns:
//   - error: Error if migration fails for any model.
//
// Concurrency:
//   - Must be called before service starts accepting requests.
//   - Not safe for concurrent execution.
func autoMigrate(db *gorm.DB, logger log.Logger, models []any) error {
	if len(models) == 0 {
		return nil
	}

	logger.Info("starting database auto-migration", log.Int("models", len(models)))

	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}

	logger.Info("database auto-migration completed successfully")
	return nil
}
