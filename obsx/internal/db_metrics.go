// Package internal provides internal implementation for obsx.
package internal

import (
	"context"
	"database/sql"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// RegisterDBMetrics registers metrics for a database connection pool.
// Metrics are collected from sql.DBStats periodically.
//
// Metrics collected:
//   - db_pool_open_connections: Number of established connections
//   - db_pool_in_use: Number of connections currently in use
//   - db_pool_idle: Number of idle connections
//   - db_pool_wait_count_total: Total number of connections waited for
//   - db_pool_wait_seconds_total: Total time blocked waiting for connections
//   - db_pool_max_open: Maximum number of open connections
//
// Parameters:
//   - name: database instance name for labeling (e.g., "main", "cache")
//   - db: sql.DB instance to monitor
//   - meterProvider: OpenTelemetry meter provider
//
// Returns:
//   - error: registration error if any
//
// Concurrency:
//   - Safe to call multiple times with different names
//
// Performance:
//   - Stats collected on scrape by OpenTelemetry SDK
func RegisterDBMetrics(name string, db *sql.DB, meterProvider *sdkmetric.MeterProvider) error {
	meter := meterProvider.Meter("go.eggybyte.com/egg/obsx/database")

	// Attribute for database name
	dbAttr := attribute.String("db_name", name)

	// Open connections gauge
	openConns, err := meter.Int64ObservableGauge(
		"db_pool_open_connections",
		metric.WithDescription("Number of established connections both in use and idle"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return err
	}

	// In use connections gauge
	inUse, err := meter.Int64ObservableGauge(
		"db_pool_in_use",
		metric.WithDescription("Number of connections currently in use"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return err
	}

	// Idle connections gauge
	idle, err := meter.Int64ObservableGauge(
		"db_pool_idle",
		metric.WithDescription("Number of idle connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return err
	}

	// Wait count counter
	waitCount, err := meter.Int64ObservableCounter(
		"db_pool_wait_count_total",
		metric.WithDescription("Total number of connections waited for"),
		metric.WithUnit("{wait}"),
	)
	if err != nil {
		return err
	}

	// Wait duration counter
	waitDuration, err := meter.Float64ObservableCounter(
		"db_pool_wait_seconds_total",
		metric.WithDescription("Total time blocked waiting for new connections"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	// Max open connections gauge
	maxOpen, err := meter.Int64ObservableGauge(
		"db_pool_max_open",
		metric.WithDescription("Maximum number of open connections to the database"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return err
	}

	// Register callback
	_, err = meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			stats := db.Stats()

			observer.ObserveInt64(openConns, int64(stats.OpenConnections), metric.WithAttributes(dbAttr))
			observer.ObserveInt64(inUse, int64(stats.InUse), metric.WithAttributes(dbAttr))
			observer.ObserveInt64(idle, int64(stats.Idle), metric.WithAttributes(dbAttr))
			observer.ObserveInt64(waitCount, stats.WaitCount, metric.WithAttributes(dbAttr))
			observer.ObserveFloat64(waitDuration, stats.WaitDuration.Seconds(), metric.WithAttributes(dbAttr))
			observer.ObserveInt64(maxOpen, int64(stats.MaxOpenConnections), metric.WithAttributes(dbAttr))

			return nil
		},
		openConns,
		inUse,
		idle,
		waitCount,
		waitDuration,
		maxOpen,
	)

	return err
}

// RegisterGORMMetrics registers metrics for a GORM database connection pool.
// This is a convenience wrapper around RegisterDBMetrics.
//
// Parameters:
//   - name: database instance name for labeling
//   - gormDB: gorm.DB interface with DB() method
//   - meterProvider: OpenTelemetry meter provider
//
// Returns:
//   - error: registration error if any
func RegisterGORMMetrics(name string, gormDB interface{ DB() (*sql.DB, error) }, meterProvider *sdkmetric.MeterProvider) error {
	sqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from gorm.DB: %w", err)
	}

	return RegisterDBMetrics(name, sqlDB, meterProvider)
}
