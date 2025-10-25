// Package internal provides internal implementation details for servicex.
package internal

import (
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/eggybyte-technology/egg/configx"
	"github.com/eggybyte-technology/egg/core/log"
	"gorm.io/gorm"
)

// ServiceConfig holds the service configuration.
type ServiceConfig struct {
	ServiceName    string
	ServiceVersion string
	Config         any
	Logger         log.Logger
	EnableTracing  bool
	EnableMetrics  bool
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
		EnableTracing:     true,
		EnableMetrics:     true,
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

// ExtractBaseConfig tries to extract BaseConfig from a config struct.
func ExtractBaseConfig(cfg any) (*configx.BaseConfig, bool) {
	val := reflect.ValueOf(cfg)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, false
	}
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := val.Type().Field(i)
		if fieldType.Type == reflect.TypeOf(configx.BaseConfig{}) {
			if bc, ok := field.Interface().(configx.BaseConfig); ok {
				return &bc, true
			}
		}
		if fieldType.Type == reflect.TypeOf(&configx.BaseConfig{}) {
			if field.IsNil() {
				continue
			}
			if bc, ok := field.Interface().(*configx.BaseConfig); ok {
				return bc, true
			}
		}
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
