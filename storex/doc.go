// Package storex defines storage interfaces and health check facilities
// with optional GORM-backed implementations.
//
// # Overview
//
// storex models storage backends behind small interfaces (Store) and offers
// a Registry to track multiple connections, perform health checks, and close
// them gracefully. GORM adapters are provided in internal packages.
//
// # Features
//
//   - Minimal storage interfaces with Ping and Close
//   - Registry for multi-store management and health checks
//   - GORM integration helpers for MySQL/Postgres/SQLite
//   - Time-bounded health checks and graceful shutdown
//
// # Usage
//
//	reg := storex.NewRegistry()
//	mysql, _ := storex.NewMySQLStore(dsn, logger)
//	_ = reg.Register("mysql", mysql)
//	_ = reg.Ping(ctx)
//
// # Layer
//
// storex is an auxiliary module and depends on core/log minimally.
//
// # Stability
//
// Experimental until v0.1.0.
package storex
