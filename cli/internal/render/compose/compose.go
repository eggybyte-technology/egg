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
		databaseYAML := r.renderDatabaseService(config.Database)
		builder.WriteString(databaseYAML)
		builder.WriteString("\n")
	}

	// Add networks section
	builder.WriteString("networks:\n")
	builder.WriteString("  default:\n")
	builder.WriteString("    name: " + config.ProjectName + "-network\n")

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
	builder.WriteString("    build:\n")
	builder.WriteString("      context: ../backend/" + name + "\n")
	builder.WriteString("      dockerfile: ../build/Dockerfile.backend\n")

	// Ports
	ports := service.Ports
	if ports == nil {
		ports = &config.BackendDefaults.Ports
	}

	builder.WriteString("    ports:\n")
	builder.WriteString("      - \"" + strconv.Itoa(ports.HTTP) + ":" + strconv.Itoa(ports.HTTP) + "\"\n")
	builder.WriteString("      - \"" + strconv.Itoa(ports.Health) + ":" + strconv.Itoa(ports.Health) + "\"\n")
	builder.WriteString("      - \"" + strconv.Itoa(ports.Metrics) + ":" + strconv.Itoa(ports.Metrics) + "\"\n")

	// Environment variables
	builder.WriteString("    environment:\n")

	// Global environment
	for key, value := range config.Env.Global {
		builder.WriteString("      - " + key + "=" + value + "\n")
	}

	// Backend environment
	for key, value := range config.Env.Backend {
		builder.WriteString("      - " + key + "=" + value + "\n")
	}

	// Service-specific environment
	for key, value := range service.Env.Common {
		builder.WriteString("      - " + key + "=" + value + "\n")
	}

	// Docker-specific environment
	for key, value := range service.Env.Docker {
		// Resolve expressions for Compose environment
		resolved, err := r.refParser.ReplaceAll(value, ref.EnvironmentCompose, config)
		if err != nil {
			return "", fmt.Errorf("failed to resolve expression %s: %w", value, err)
		}
		builder.WriteString("      - " + key + "=" + resolved + "\n")
	}

	// Port environment variables
	builder.WriteString("      - HTTP_PORT=" + strconv.Itoa(ports.HTTP) + "\n")
	builder.WriteString("      - HEALTH_PORT=" + strconv.Itoa(ports.Health) + "\n")
	builder.WriteString("      - METRICS_PORT=" + strconv.Itoa(ports.Metrics) + "\n")

	// Dependencies
	if config.Database.Enabled {
		builder.WriteString("    depends_on:\n")
		builder.WriteString("      - mysql\n")
	}

	// Health check
	builder.WriteString("    healthcheck:\n")
	builder.WriteString("      test: [\"CMD\", \"curl\", \"-f\", \"http://localhost:" + strconv.Itoa(ports.Health) + "/health\"]\n")
	builder.WriteString("      interval: 30s\n")
	builder.WriteString("      timeout: 10s\n")
	builder.WriteString("      retries: 3\n")

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
	builder.WriteString("    build:\n")
	builder.WriteString("      context: ../frontend/" + name + "\n")
	builder.WriteString("      dockerfile: ../build/Dockerfile.frontend\n")

	// Ports (default for frontend)
	builder.WriteString("    ports:\n")
	builder.WriteString("      - \"3000:3000\"\n")

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
func (r *Renderer) renderDatabaseService(db configschema.DatabaseConfig) string {
	var builder strings.Builder

	builder.WriteString("  mysql:\n")
	builder.WriteString("    image: " + db.Image + "\n")
	builder.WriteString("    ports:\n")
	builder.WriteString("      - \"" + strconv.Itoa(db.Port) + ":" + strconv.Itoa(db.Port) + "\"\n")
	builder.WriteString("    environment:\n")
	builder.WriteString("      - MYSQL_ROOT_PASSWORD=" + db.RootPassword + "\n")
	builder.WriteString("      - MYSQL_DATABASE=" + db.Database + "\n")
	builder.WriteString("      - MYSQL_USER=" + db.User + "\n")
	builder.WriteString("      - MYSQL_PASSWORD=" + db.Password + "\n")
	builder.WriteString("    healthcheck:\n")
	builder.WriteString("      test: [\"CMD\", \"mysqladmin\", \"ping\", \"-h\", \"localhost\"]\n")
	builder.WriteString("      interval: 30s\n")
	builder.WriteString("      timeout: 10s\n")
	builder.WriteString("      retries: 3\n")

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
