package config

import "github.com/eggybyte-technology/egg/configx"

// AppConfig represents the application configuration.
type AppConfig struct {
	configx.BaseConfig
	// Add your custom configuration fields here
	// Example: RateLimitQPS int ` + "`" + `env:"RATE_LIMIT_QPS" default:"200"` + "`" + `
}
