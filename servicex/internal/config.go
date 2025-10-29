// Package internal provides internal implementation details for servicex.
package internal

import (
	"fmt"
	"log/slog"
	"time"

	"go.eggybyte.com/egg/configx"
	"go.eggybyte.com/egg/core/log"
	"gorm.io/gorm"
)

// ServiceConfig holds the service configuration.
type ServiceConfig struct {
	ServiceName    string
	ServiceVersion string
	Config         any
	Logger         log.Logger
	EnableMetrics  bool
	MetricsConfig  *MetricsConfig // Fine-grained metrics configuration
	EnableDebug    bool
	RegisterFn     func(interface{}) error // Takes *App interface

	// Server ports
	HTTPPort    int
	HealthPort  int
	MetricsPort int

	// Connect options
	DefaultTimeoutMs  int64
	SlowRequestMillis int64

	// Database
	DBConfig          *DatabaseConfig
	AutoMigrateModels []any

	// Shutdown
	ShutdownTimeout time.Duration
	ShutdownHooks   []func(interface{}) error
}

// MetricsConfig holds fine-grained metrics configuration.
type MetricsConfig struct {
	EnableRuntime bool // Enable Go runtime metrics (goroutines, GC, memory)
	EnableProcess bool // Enable process metrics (CPU, RSS, uptime)
	EnableDB      bool // Enable database connection pool metrics
	EnableClient  bool // Enable client-side RPC metrics
}

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
	Driver          string
	DSN             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	PingTimeout     time.Duration
}

// NewServiceConfig creates a new service configuration with defaults.
func NewServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		ServiceName:       "app",
		ServiceVersion:    "0.0.0",
		EnableMetrics:     true,
		MetricsConfig:     nil, // Will be set by WithMetricsConfig if needed
		HTTPPort:          8080,
		HealthPort:        8081,
		MetricsPort:       9091,
		DefaultTimeoutMs:  30000,
		SlowRequestMillis: 1000,
		ShutdownTimeout:   15 * time.Second,
	}
}

// ParseLogLevel converts a log level string to slog.Level.
func ParseLogLevel(levelStr string) (slog.Level, error) {
	switch levelStr {
	case "debug", "DEBUG":
		return slog.LevelDebug, nil
	case "info", "INFO":
		return slog.LevelInfo, nil
	case "warn", "WARN", "warning", "WARNING":
		return slog.LevelWarn, nil
	case "error", "ERROR":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level: %s", levelStr)
	}
}

// BaseConfigProvider is an interface for config structs that provide BaseConfig.
//
// Types implementing this interface can return their BaseConfig for framework use.
// This is the modern, idiomatic Go way to handle configuration polymorphism.
//
// Example:
//
//	type AppConfig struct {
//	    BaseConfig configx.BaseConfig
//	    // ... app-specific fields
//	}
//
//	func (c *AppConfig) GetBaseConfig() *configx.BaseConfig {
//	    return &c.BaseConfig
//	}
type BaseConfigProvider interface {
	GetBaseConfig() *configx.BaseConfig
}

// ExtractBaseConfig extracts BaseConfig from a config struct in a type-safe way.
//
// Supports two patterns:
//  1. Direct *BaseConfig: cfg is *configx.BaseConfig itself
//  2. BaseConfigProvider: cfg implements GetBaseConfig() method
//
// This modern approach avoids reflection and is compile-time safe.
func ExtractBaseConfig(cfg any) (*configx.BaseConfig, bool) {
	if cfg == nil {
		return nil, false
	}

	// Pattern 1: Direct BaseConfig pointer (e.g., &configx.BaseConfig{})
	if bc, ok := cfg.(*configx.BaseConfig); ok {
		return bc, true
	}

	// Pattern 2: Implements BaseConfigProvider interface (recommended for custom configs)
	if provider, ok := cfg.(BaseConfigProvider); ok {
		return provider.GetBaseConfig(), true
	}

	return nil, false
}

// AutoMigrate runs GORM auto-migration for the provided models.
func AutoMigrate(db *gorm.DB, logger log.Logger, models []any) error {
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
