// Package config provides configuration management for the user service.
//
// Overview:
//   - Responsibility: Application configuration and environment management
//   - Key Types: AppConfig struct with validation and defaults
//   - Concurrency Model: Thread-safe configuration access
//   - Error Semantics: Configuration errors are wrapped and returned
//   - Performance Notes: Optimized for hot reload and validation
//
// Usage:
//
//	var cfg AppConfig
//	mgr.Bind(&cfg)
//	dbURL := cfg.Database.DSN
package config

import (
	"fmt"

	"github.com/eggybyte-technology/egg/configx"
)

// AppConfig extends BaseConfig with application-specific settings.
// Database configuration is inherited from BaseConfig.
type AppConfig struct {
	configx.BaseConfig

	// Business configuration
	DefaultPageSize int `env:"DEFAULT_PAGE_SIZE" default:"10"`
	MaxPageSize     int `env:"MAX_PAGE_SIZE" default:"100"`
}

// Validate performs configuration validation.
// Returns an error if validation fails.
func (c *AppConfig) Validate() error {
	if c.DefaultPageSize < 1 {
		return fmt.Errorf("default page size must be positive")
	}

	if c.MaxPageSize < c.DefaultPageSize {
		return fmt.Errorf("max page size must be >= default page size")
	}

	return nil
}
