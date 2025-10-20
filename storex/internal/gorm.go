// Package internal contains GORM database adapter implementation.
package internal

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/eggybyte-technology/egg/core/log"
)

// GORMStore implements the Store interface using GORM.
type GORMStore struct {
	db     *gorm.DB
	logger log.Logger
}

// NewGORMStore creates a new GORM store.
func NewGORMStore(db *gorm.DB, logger log.Logger) *GORMStore {
	return &GORMStore{
		db:     db,
		logger: logger,
	}
}

// Ping checks if the database connection is healthy.
func (s *GORMStore) Ping(ctx context.Context) error {
	if s.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (s *GORMStore) Close() error {
	if s.db == nil {
		return nil // Already closed or never opened
	}

	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	return nil
}

// GetDB returns the underlying GORM database instance.
func (s *GORMStore) GetDB() *gorm.DB {
	return s.db
}

// GORMOptions holds configuration for GORM database connections.
type GORMOptions struct {
	DSN             string          // Database connection string
	Driver          string          // Database driver (mysql, postgres, sqlite)
	MaxIdleConns    int             // Maximum number of idle connections
	MaxOpenConns    int             // Maximum number of open connections
	ConnMaxLifetime time.Duration   // Maximum connection lifetime
	Logger          log.Logger      // Logger for database operations
	LogLevel        logger.LogLevel // GORM log level
}

// DefaultGORMOptions returns default GORM options.
func DefaultGORMOptions() GORMOptions {
	return GORMOptions{
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
		LogLevel:        logger.Silent,
	}
}

// NewGORMStoreFromOptions creates a new GORM store from options.
func NewGORMStoreFromOptions(opts GORMOptions) (*GORMStore, error) {
	if opts.DSN == "" {
		return nil, fmt.Errorf("DSN is required")
	}

	if opts.Driver == "" {
		return nil, fmt.Errorf("driver is required")
	}

	// Create GORM logger
	var gormLogger logger.Interface
	if opts.Logger != nil {
		gormLogger = &gormLogAdapter{logger: opts.Logger}
	} else {
		gormLogger = logger.Default.LogMode(opts.LogLevel)
	}

	// Get GORM driver
	driver, err := getGORMDriver(opts.Driver, opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver: %w", err)
	}

	// Open database connection
	db, err := gorm.Open(driver, &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Check if db is nil (some drivers might return nil on failure)
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
	sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(opts.ConnMaxLifetime)

	return NewGORMStore(db, opts.Logger), nil
}

// getGORMDriver returns the GORM driver for the given driver name.
func getGORMDriver(driver, dsn string) (gorm.Dialector, error) {
	switch driver {
	case "mysql":
		return mysql.Open(dsn), nil
	case "postgres":
		return postgres.Open(dsn), nil
	case "sqlite":
		return sqlite.Open(dsn), nil
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}
}

// gormLogAdapter adapts our logger to GORM's logger interface.
type gormLogAdapter struct {
	logger log.Logger
}

func (l *gormLogAdapter) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

func (l *gormLogAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Info(msg, data...)
}

func (l *gormLogAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Warn(msg, data...)
}

func (l *gormLogAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Error(nil, msg, data...)
}

func (l *gormLogAdapter) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if err != nil {
		l.logger.Error(err, "database query failed")
	} else {
		sql, rows := fc()
		l.logger.Debug("database query", log.Str("sql", sql), log.Int("rows", int(rows)))
	}
}
