// Package compose provides Docker Compose rendering for egg projects.
//
// Overview:
//   - Responsibility: Render compose.yaml from egg configuration
//   - Key Types: Compose renderer, service templates, environment resolvers
//   - Concurrency Model: Immutable rendering with atomic file writes
//   - Error Semantics: Rendering errors with configuration validation
//   - Performance Notes: Template-based rendering, minimal I/O operations
//
// Usage:
//
//	renderer := NewRenderer(fs, refParser)
//	err := renderer.Render(config)
package compose

import (
	"fmt"
	"strconv"
	"strings"

	"go.eggybyte.com/egg/cli/internal/configschema"
	"go.eggybyte.com/egg/cli/internal/projectfs"
	"go.eggybyte.com/egg/cli/internal/ref"
	"go.eggybyte.com/egg/cli/internal/ui"
)

// Renderer provides Docker Compose rendering functionality.
//
// Parameters:
//   - fs: Project file system
//   - refParser: Reference expression parser
//
// Returns:
//   - None (data structure)
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Template-based rendering
type Renderer struct {
	fs        *projectfs.ProjectFS
	refParser *ref.Parser
}

// NewRenderer creates a new Compose renderer.
//
// Parameters:
//   - fs: Project file system
//   - refParser: Reference expression parser
//
// Returns:
//   - *Renderer: Compose renderer instance
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Minimal initialization overhead
func NewRenderer(fs *projectfs.ProjectFS, refParser *ref.Parser) *Renderer {
	return &Renderer{
		fs:        fs,
		refParser: refParser,
	}
}

// Render renders Docker Compose configuration.
//
// Parameters:
//   - config: Project configuration
//
// Returns:
//   - error: Rendering error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering and file I/O
func (r *Renderer) Render(config *configschema.Config) error {
	ui.Info("Rendering Docker Compose configuration...")

	// Create deploy/compose directory
	if err := r.fs.CreateDirectory("deploy/compose"); err != nil {
		return fmt.Errorf("failed to create deploy/compose directory: %w", err)
	}

	// Generate compose.yaml
	composeYAML, err := r.generateComposeYAML(config)
	if err != nil {
		return fmt.Errorf("failed to generate compose.yaml: %w", err)
	}

	// Write compose.yaml
	if err := r.fs.WriteFile("deploy/compose/compose.yaml", composeYAML, 0644); err != nil {
		return fmt.Errorf("failed to write compose.yaml: %w", err)
	}

	// Generate .env file if needed
	if err := r.generateEnvFile(config); err != nil {
		return fmt.Errorf("failed to generate .env file: %w", err)
	}

	ui.Success("Docker Compose configuration rendered")
	return nil
}

// generateComposeYAML generates the Docker Compose YAML content.
//
// Parameters:
//   - config: Project configuration
//
// Returns:
//   - string: Compose YAML content
//   - error: Generation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - String building and template rendering
func (r *Renderer) generateComposeYAML(config *configschema.Config) (string, error) {
	var builder strings.Builder

	// Write Compose header
	builder.WriteString("version: '3.8'\n\n")
	builder.WriteString("services:\n")

	// Render backend services
	for name, service := range config.Backend {
		serviceYAML, err := r.renderBackendService(name, service, config)
		if err != nil {
			return "", fmt.Errorf("failed to render backend service %s: %w", name, err)
		}
		builder.WriteString(serviceYAML)
		builder.WriteString("\n")
	}

	// Render frontend services
	for name, service := range config.Frontend {
		serviceYAML, err := r.renderFrontendService(name, service, config)
		if err != nil {
			return "", fmt.Errorf("failed to render frontend service %s: %w", name, err)
		}
		builder.WriteString(serviceYAML)
		builder.WriteString("\n")
	}

	// Add database service if enabled
	if config.Database.Enabled {
		databaseYAML := r.renderDatabaseService(config.Database, config.ProjectName)
		builder.WriteString(databaseYAML)
		builder.WriteString("\n")
	}

	// Add volumes section if database is enabled
	if config.Database.Enabled {
		builder.WriteString("volumes:\n")
		builder.WriteString("  mysql_data:\n\n")
	}

	// Add networks section
	builder.WriteString("networks:\n")
	builder.WriteString("  " + config.ProjectName + "-network:\n")
	builder.WriteString("    driver: bridge\n")

	return builder.String(), nil
}

// renderBackendService renders a backend service configuration.
//
// Parameters:
//   - name: Service name
//   - service: Service configuration
//   - config: Project configuration
//
// Returns:
//   - string: Service YAML content
//   - error: Rendering error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - String building and environment resolution
func (r *Renderer) renderBackendService(name string, service configschema.BackendService, config *configschema.Config) (string, error) {
	var builder strings.Builder

	// Service header
	builder.WriteString("  " + name + ":\n")
	// Use pre-built image instead of building
	imageName := fmt.Sprintf("%s/%s-%s:%s", config.DockerRegistry, config.ProjectName, name, config.Version)
	builder.WriteString("    image: " + imageName + "\n")

	// Ports - removed port mapping (services are accessed via docker network)
	// Ports are still configured via environment variables for servicex
	ports := service.Ports
	if ports == nil {
		ports = &config.BackendDefaults.Ports
	}
	// Port configuration is done via environment variables, not port mapping

	// Environment variables (servicex standard)
	builder.WriteString("    environment:\n")

	// Service Identity (servicex required)
	builder.WriteString("      # Service Identity (servicex standard)\n")
	builder.WriteString("      - SERVICE_NAME=" + name + "\n")
	builder.WriteString("      - SERVICE_VERSION=" + config.Version + "\n")

	// Environment (servicex standard: ENV, not APP_ENV)
	// Priority: service.env.common.ENV > env.backend.ENV > env.global.ENV > default "production"
	envValue := "production" // default for docker compose
	if envVal, exists := service.Env.Common["ENV"]; exists && envVal != "" {
		envValue = envVal
	} else if envVal, exists := config.Env.Backend["ENV"]; exists && envVal != "" {
		envValue = envVal
	} else if envVal, exists := config.Env.Global["ENV"]; exists && envVal != "" {
		envValue = envVal
	}
	builder.WriteString("      - ENV=" + envValue + "\n")

	// Logging Configuration (servicex standard)
	// Priority: service.env.common.LOG_LEVEL > env.backend.LOG_LEVEL > env.global.LOG_LEVEL > default "info"
	logLevel := "info" // default
	if logLevelVal, exists := service.Env.Common["LOG_LEVEL"]; exists && logLevelVal != "" {
		logLevel = logLevelVal
	} else if logLevelVal, exists := config.Env.Backend["LOG_LEVEL"]; exists && logLevelVal != "" {
		logLevel = logLevelVal
	} else if logLevelVal, exists := config.Env.Global["LOG_LEVEL"]; exists && logLevelVal != "" {
		logLevel = logLevelVal
	}
	builder.WriteString("      # Logging Configuration (servicex standard)\n")
	builder.WriteString("      # Supported levels: debug, info, warn, error (default: info)\n")
	builder.WriteString("      - LOG_LEVEL=" + logLevel + "\n")

	// Port Configuration (servicex BaseConfig)
	builder.WriteString("      # Port Configuration (servicex BaseConfig)\n")
	builder.WriteString("      - HTTP_PORT=" + strconv.Itoa(ports.HTTP) + "\n")
	builder.WriteString("      - HEALTH_PORT=" + strconv.Itoa(ports.Health) + "\n")
	builder.WriteString("      - METRICS_PORT=" + strconv.Itoa(ports.Metrics) + "\n")

	// Database Configuration (servicex BaseConfig.Database)
	if config.Database.Enabled {
		builder.WriteString("      # Database Configuration (servicex BaseConfig.Database)\n")
		builder.WriteString("      - DB_DRIVER=mysql\n")
		builder.WriteString("      - DB_DSN=" + config.Database.User + ":" + config.Database.Password + "@tcp(mysql:3306)/" + config.Database.Database + "?charset=utf8mb4&parseTime=True&loc=Local\n")

		// Database connection pool settings (only if overridden from defaults)
		// Defaults: DB_MAX_IDLE=10, DB_MAX_OPEN=100, DB_MAX_LIFETIME=1h, DB_PING_TIMEOUT=5s
		if dbMaxIdle, exists := config.Env.Backend["DB_MAX_IDLE"]; exists && dbMaxIdle != "" {
			builder.WriteString("      - DB_MAX_IDLE=" + dbMaxIdle + "\n")
		}
		if dbMaxOpen, exists := config.Env.Backend["DB_MAX_OPEN"]; exists && dbMaxOpen != "" {
			builder.WriteString("      - DB_MAX_OPEN=" + dbMaxOpen + "\n")
		}
		if dbMaxLifetime, exists := config.Env.Backend["DB_MAX_LIFETIME"]; exists && dbMaxLifetime != "" {
			builder.WriteString("      - DB_MAX_LIFETIME=" + dbMaxLifetime + "\n")
		}
		if dbPingTimeout, exists := config.Env.Backend["DB_PING_TIMEOUT"]; exists && dbPingTimeout != "" {
			builder.WriteString("      - DB_PING_TIMEOUT=" + dbPingTimeout + "\n")
		}
	}

	// Metrics Configuration (servicex standard)
	// Default: ENABLE_METRICS=true, only set if explicitly disabled
	enableMetrics := "true"
	if metricsVal, exists := service.Env.Common["ENABLE_METRICS"]; exists && metricsVal != "" {
		enableMetrics = metricsVal
	} else if metricsVal, exists := config.Env.Backend["ENABLE_METRICS"]; exists && metricsVal != "" {
		enableMetrics = metricsVal
	} else if metricsVal, exists := config.Env.Global["ENABLE_METRICS"]; exists && metricsVal != "" {
		enableMetrics = metricsVal
	}
	if enableMetrics != "true" {
		builder.WriteString("      - ENABLE_METRICS=" + enableMetrics + "\n")
	}

	// RPC Configuration (servicex standard)
	// Default: SLOW_REQUEST_MILLIS=1000, only set if overridden
	if slowRequestMillis, exists := service.Env.Common["SLOW_REQUEST_MILLIS"]; exists && slowRequestMillis != "" && slowRequestMillis != "1000" {
		builder.WriteString("      - SLOW_REQUEST_MILLIS=" + slowRequestMillis + "\n")
	} else if slowRequestMillis, exists := config.Env.Backend["SLOW_REQUEST_MILLIS"]; exists && slowRequestMillis != "" && slowRequestMillis != "1000" {
		builder.WriteString("      - SLOW_REQUEST_MILLIS=" + slowRequestMillis + "\n")
	} else if slowRequestMillis, exists := config.Env.Global["SLOW_REQUEST_MILLIS"]; exists && slowRequestMillis != "" && slowRequestMillis != "1000" {
		builder.WriteString("      - SLOW_REQUEST_MILLIS=" + slowRequestMillis + "\n")
	}

	// Shutdown Configuration (servicex standard)
	// Default: SHUTDOWN_TIMEOUT=15s, only set if overridden
	if shutdownTimeout, exists := service.Env.Common["SHUTDOWN_TIMEOUT"]; exists && shutdownTimeout != "" && shutdownTimeout != "15s" {
		builder.WriteString("      - SHUTDOWN_TIMEOUT=" + shutdownTimeout + "\n")
	} else if shutdownTimeout, exists := config.Env.Backend["SHUTDOWN_TIMEOUT"]; exists && shutdownTimeout != "" && shutdownTimeout != "15s" {
		builder.WriteString("      - SHUTDOWN_TIMEOUT=" + shutdownTimeout + "\n")
	} else if shutdownTimeout, exists := config.Env.Global["SHUTDOWN_TIMEOUT"]; exists && shutdownTimeout != "" && shutdownTimeout != "15s" {
		builder.WriteString("      - SHUTDOWN_TIMEOUT=" + shutdownTimeout + "\n")
	}

	// Custom environment variables (excluding servicex standard vars to avoid duplication)
	servicexVars := map[string]bool{
		"SERVICE_NAME":        true,
		"SERVICE_VERSION":     true,
		"ENV":                 true,
		"LOG_LEVEL":           true,
		"HTTP_PORT":           true,
		"HEALTH_PORT":         true,
		"METRICS_PORT":        true,
		"DB_DRIVER":           true,
		"DB_DSN":              true,
		"DB_MAX_IDLE":         true,
		"DB_MAX_OPEN":         true,
		"DB_MAX_LIFETIME":     true,
		"DB_PING_TIMEOUT":     true,
		"ENABLE_METRICS":      true,
		"SLOW_REQUEST_MILLIS": true,
		"SHUTDOWN_TIMEOUT":    true,
	}

	// Global environment (excluding servicex standard vars)
	for key, value := range config.Env.Global {
		if !servicexVars[key] {
			builder.WriteString("      - " + key + "=" + value + "\n")
		}
	}

	// Backend environment (excluding servicex standard vars)
	for key, value := range config.Env.Backend {
		if !servicexVars[key] {
			builder.WriteString("      - " + key + "=" + value + "\n")
		}
	}

	// Service-specific environment (excluding servicex standard vars)
	for key, value := range service.Env.Common {
		if !servicexVars[key] {
			builder.WriteString("      - " + key + "=" + value + "\n")
		}
	}

	// Docker-specific environment
	for key, value := range service.Env.Docker {
		if !servicexVars[key] {
			// Resolve expressions for Compose environment
			resolved, err := r.refParser.ReplaceAll(value, ref.EnvironmentCompose, config)
			if err != nil {
				return "", fmt.Errorf("failed to resolve expression %s: %w", value, err)
			}
			builder.WriteString("      - " + key + "=" + resolved + "\n")
		}
	}

	// Dependencies
	if config.Database.Enabled {
		builder.WriteString("    depends_on:\n")
		builder.WriteString("      mysql:\n")
		builder.WriteString("        condition: service_healthy\n")
	}

	// Networks
	builder.WriteString("    networks:\n")
	builder.WriteString("      - " + config.ProjectName + "-network\n")

	// Health check
	builder.WriteString("    healthcheck:\n")
	builder.WriteString("      test: [\"CMD\", \"wget\", \"--spider\", \"-q\", \"http://localhost:" + strconv.Itoa(ports.Health) + "/health\"]\n")
	builder.WriteString("      interval: 10s\n")
	builder.WriteString("      timeout: 5s\n")
	builder.WriteString("      retries: 5\n")
	builder.WriteString("      start_period: 10s\n")
	builder.WriteString("    restart: unless-stopped\n")

	return builder.String(), nil
}

// renderFrontendService renders a frontend service configuration.
//
// Parameters:
//   - name: Service name
//   - service: Service configuration
//   - config: Project configuration
//
// Returns:
//   - string: Service YAML content
//   - error: Rendering error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - String building and environment resolution
func (r *Renderer) renderFrontendService(name string, service configschema.FrontendService, config *configschema.Config) (string, error) {
	var builder strings.Builder

	// Service header
	builder.WriteString("  " + name + ":\n")
	// Use pre-built image instead of building
	// Frontend image names use hyphens instead of underscores
	dockerServiceName := strings.ReplaceAll(name, "_", "-")
	imageName := fmt.Sprintf("%s/%s-%s-frontend:%s", config.DockerRegistry, config.ProjectName, dockerServiceName, config.Version)
	builder.WriteString("    image: " + imageName + "\n")
	builder.WriteString("    container_name: " + config.ProjectName + "-" + dockerServiceName + "\n")
	builder.WriteString("    restart: unless-stopped\n")

	// Ports - removed port mapping (services are accessed via docker network)
	// Port 3000 is still configured via environment variables

	// Environment variables
	builder.WriteString("    environment:\n")

	// Global environment
	for key, value := range config.Env.Global {
		builder.WriteString("      - " + key + "=" + value + "\n")
	}

	// Frontend environment
	for key, value := range config.Env.Frontend {
		builder.WriteString("      - " + key + "=" + value + "\n")
	}

	// Networks
	builder.WriteString("    networks:\n")
	builder.WriteString("      - " + config.ProjectName + "-network\n")

	// Health check for frontend (nginx health check)
	builder.WriteString("    healthcheck:\n")
	builder.WriteString("      test: [\"CMD\", \"wget\", \"--spider\", \"-q\", \"http://localhost:3000\"]\n")
	builder.WriteString("      interval: 10s\n")
	builder.WriteString("      timeout: 5s\n")
	builder.WriteString("      retries: 5\n")
	builder.WriteString("      start_period: 10s\n")

	return builder.String(), nil
}

// renderDatabaseService renders the database service configuration.
//
// Parameters:
//   - db: Database configuration
//
// Returns:
//   - string: Database service YAML content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - String building
func (r *Renderer) renderDatabaseService(db configschema.DatabaseConfig, projectName string) string {
	var builder strings.Builder

	builder.WriteString("  mysql:\n")
	builder.WriteString("    image: " + db.Image + "\n")
	builder.WriteString("    container_name: " + projectName + "-mysql\n")
	builder.WriteString("    restart: unless-stopped\n")
	builder.WriteString("    environment:\n")
	builder.WriteString("      - MYSQL_ROOT_PASSWORD=" + db.RootPassword + "\n")
	builder.WriteString("      - MYSQL_DATABASE=" + db.Database + "\n")
	builder.WriteString("      - MYSQL_USER=" + db.User + "\n")
	builder.WriteString("      - MYSQL_PASSWORD=" + db.Password + "\n")
	// Ports - removed port mapping (database is accessed via docker network)
	builder.WriteString("    volumes:\n")
	builder.WriteString("      - mysql_data:/var/lib/mysql\n")
	builder.WriteString("    healthcheck:\n")
	builder.WriteString("      test: [\"CMD\", \"mysqladmin\", \"ping\", \"-h\", \"localhost\", \"-u\", \"root\", \"-p" + db.RootPassword + "\"]\n")
	builder.WriteString("      interval: 10s\n")
	builder.WriteString("      timeout: 5s\n")
	builder.WriteString("      retries: 5\n")
	builder.WriteString("    networks:\n")
	builder.WriteString("      - " + projectName + "-network\n")

	return builder.String()
}

// generateEnvFile generates a .env file for environment variables.
//
// Parameters:
//   - config: Project configuration
//
// Returns:
//   - error: Generation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - File I/O operation
func (r *Renderer) generateEnvFile(config *configschema.Config) error {
	var builder strings.Builder

	// Global environment
	for key, value := range config.Env.Global {
		builder.WriteString(key + "=" + value + "\n")
	}

	// Backend environment
	for key, value := range config.Env.Backend {
		builder.WriteString(key + "=" + value + "\n")
	}

	// Frontend environment
	for key, value := range config.Env.Frontend {
		builder.WriteString(key + "=" + value + "\n")
	}

	// Database environment
	if config.Database.Enabled {
		builder.WriteString("MYSQL_ROOT_PASSWORD=" + config.Database.RootPassword + "\n")
		builder.WriteString("MYSQL_DATABASE=" + config.Database.Database + "\n")
		builder.WriteString("MYSQL_USER=" + config.Database.User + "\n")
		builder.WriteString("MYSQL_PASSWORD=" + config.Database.Password + "\n")
	}

	// Write .env file
	if err := r.fs.WriteFile("deploy/compose/.env", builder.String(), 0644); err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	return nil
}

// AttachMySQL adds MySQL service to Compose configuration.
//
// Parameters:
//   - config: Project configuration
//
// Returns:
//   - error: Attachment error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - File modification
func (r *Renderer) AttachMySQL(config *configschema.Config) error {
	if !config.Database.Enabled {
		return nil
	}

	ui.Info("Attaching MySQL service to Compose configuration...")

	// This is already handled in the main render method
	// This function exists for future extensibility

	ui.Success("MySQL service attached")
	return nil
}
