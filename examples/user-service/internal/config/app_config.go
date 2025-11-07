// Package config provides configuration management for the user service.
//
// Overview:
//   - Responsibility: Application configuration and environment management
//   - Key Types: AppConfig struct with validation and defaults
//   - Concurrency Model: Thread-safe configuration access (immutable after load)
//   - Error Semantics: Configuration errors are returned with clear messages
//   - Performance Notes: Configuration is loaded once at startup
//
// Usage:
//
//	cfg := &config.AppConfig{}
//	err := servicex.Run(ctx,
//	    servicex.WithAppConfig(cfg),
//	    // ... other options
//	)
//	// Access config: cfg.DefaultPageSize
package config

import (
	"fmt"

	"go.eggybyte.com/egg/configx"
)

// AppConfig extends BaseConfig with application-specific settings for the user service.
//
// This configuration struct demonstrates the egg framework's configuration pattern:
// extending BaseConfig for standard service settings (ports, service name, database)
// and adding domain-specific settings for business logic.
//
// Database configuration is inherited from BaseConfig.Database and includes:
//   - DSN: Database connection string
//   - MaxOpenConns: Maximum number of open connections
//   - MaxIdleConns: Maximum number of idle connections
//   - ConnMaxLifetime: Maximum connection lifetime
//
// Concurrency:
//
//	Configuration is loaded once at startup and should be treated as immutable
//	during runtime. Safe for concurrent read access.
type AppConfig struct {
	configx.BaseConfig

	// DefaultPageSize specifies the default number of items per page for list operations.
	// Default: 10
	// Environment variable: DEFAULT_PAGE_SIZE
	DefaultPageSize int `env:"DEFAULT_PAGE_SIZE" default:"10"`

	// MaxPageSize specifies the maximum allowed page size to prevent excessive memory usage.
	// Default: 100
	// Environment variable: MAX_PAGE_SIZE
	MaxPageSize int `env:"MAX_PAGE_SIZE" default:"100"`

	// GreetServiceURL specifies the base URL for the greet service.
	// This is used for service-to-service communication examples.
	// Default: "http://minimal-service:8080"
	// Environment variable: GREET_SERVICE_URL
	GreetServiceURL string `env:"GREET_SERVICE_URL" default:"http://minimal-service:8080"`
}

// GetBaseConfig returns the embedded BaseConfig for framework use.
//
// This method implements the servicex.BaseConfigProvider interface, allowing
// the framework to access standard configuration fields (log level, ports, etc.)
// in a type-safe way without reflection.
//
// Returns:
//   - *configx.BaseConfig: pointer to the embedded BaseConfig
//
// Concurrency:
//
//	Safe for concurrent read access.
func (c *AppConfig) GetBaseConfig() *configx.BaseConfig {
	return &c.BaseConfig
}

// Validate performs configuration validation to ensure all settings are valid.
//
// This method is called automatically by configx after loading configuration from
// environment variables. It implements the configx.Validator interface.
//
// Returns:
//   - error: nil if configuration is valid; error with descriptive message otherwise
//
// Validation rules:
//   - DefaultPageSize must be positive (>= 1)
//   - MaxPageSize must be greater than or equal to DefaultPageSize
//   - Database configuration is validated by BaseConfig
//
// Concurrency:
//
//	Called once during initialization, not safe for concurrent use.
func (c *AppConfig) Validate() error {
	if c.DefaultPageSize < 1 {
		return fmt.Errorf("default page size must be positive, got: %d", c.DefaultPageSize)
	}

	if c.MaxPageSize < c.DefaultPageSize {
		return fmt.Errorf("max page size (%d) must be >= default page size (%d)", c.MaxPageSize, c.DefaultPageSize)
	}

	return nil
}
