// Package egg provides the egg CLI command implementations.
//
// Overview:
//   - Responsibility: CLI command execution and orchestration
//   - Key Types: Command handlers, argument parsers, option processors
//   - Concurrency Model: Sequential command execution with context support
//   - Error Semantics: User-friendly error messages with suggestions
//   - Performance Notes: Fast command resolution, minimal initialization
//
// Usage:
//
//	egg create backend <name>
//	egg create frontend <name> --platforms web
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/eggybyte-technology/egg/cli/internal/configschema"
	"github.com/eggybyte-technology/egg/cli/internal/generators"
	"github.com/eggybyte-technology/egg/cli/internal/projectfs"
	"github.com/eggybyte-technology/egg/cli/internal/toolrunner"
	"github.com/eggybyte-technology/egg/cli/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// createCmd represents the create command.
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new service",
	Long: `Create a new backend or frontend service.

This command generates:
- Service directory structure
- Go module initialization (for backend)
- Flutter project initialization (for frontend)
- Basic service templates
- Workspace configuration updates

Examples:
  egg create backend user-service
  egg create frontend admin-portal --platforms web
  egg create frontend mobile-app --platforms android,ios`,
}

// createBackendCmd represents the create backend command.
var createBackendCmd = &cobra.Command{
	Use:   "backend <name>",
	Short: "Create a new backend service",
	Long: `Create a new backend service with Connect-first architecture.

This command creates:
- Go module with egg dependencies
- Connect-only service structure
- Health and metrics endpoints
- Configuration management
- Kubernetes deployment templates

Examples:
  egg create backend user-service
  egg create backend user-service --local-modules`,
	Args: cobra.ExactArgs(1),
	RunE: runCreateBackend,
}

// createFrontendCmd represents the create frontend command.
var createFrontendCmd = &cobra.Command{
	Use:   "frontend <name>",
	Short: "Create a new frontend service",
	Long: `Create a new frontend service with Flutter.

This command creates:
- Flutter project structure
- Web/mobile platform support
- Build configuration
- Deployment templates

Examples:
  egg create frontend admin-portal --platforms web
  egg create frontend mobile-app --platforms android,ios
  egg create frontend hybrid-app --platforms web,android,ios
  
Note: For specific Flutter versions, use FVM before running this command.`,
	Args: cobra.ExactArgs(1),
	RunE: runCreateFrontend,
}

var (
	frontendPlatforms []string
	useLocalModules   bool
	forceCreate       bool
)

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.AddCommand(createBackendCmd)
	createCmd.AddCommand(createFrontendCmd)

	createBackendCmd.Flags().BoolVar(&useLocalModules, "local-modules", false, "Use local egg modules for development")
	createBackendCmd.Flags().BoolVar(&forceCreate, "force", false, "Force create service even if it already exists")
	createFrontendCmd.Flags().StringSliceVar(&frontendPlatforms, "platforms", []string{"web"}, "Target platforms (comma-separated: web, android, ios)")
	createFrontendCmd.Flags().BoolVar(&forceCreate, "force", false, "Force create service even if it already exists")
}

// runCreateBackend executes the create backend command.
//
// Parameters:
//   - cmd: Cobra command
//   - args: Command arguments
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Service generation and module initialization
func runCreateBackend(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	serviceName := args[0]

	ui.Info("Creating backend service: %s", serviceName)

	// Load configuration
	config, diags, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if diags.HasErrors() {
		ui.Error("Configuration validation failed:")
		for _, diag := range diags.Items() {
			if diag.Severity == configschema.SeverityError {
				ui.Error("  %s: %s", diag.Path, diag.Message)
			}
		}
		return fmt.Errorf("configuration validation failed")
	}

	// Check if service is defined in configuration
	serviceExists := false
	if _, exists := config.Backend[serviceName]; exists {
		serviceExists = true
	}

	// If service doesn't exist, auto-register it
	if !serviceExists {
		ui.Info("Service '%s' not found in configuration, auto-registering...", serviceName)
		if err := autoRegisterBackendService(serviceName, config); err != nil {
			return fmt.Errorf("failed to auto-register service: %w", err)
		}
		ui.Success("Service '%s' registered in egg.yaml", serviceName)
	} else if !forceCreate {
		// Service exists but no force flag
		return fmt.Errorf("backend service '%s' already exists in configuration. Use --force to overwrite", serviceName)
	} else {
		ui.Info("Service '%s' already exists, overwriting due to --force flag", serviceName)
	}

	// Create project file system
	fs := projectfs.NewProjectFS(".")
	fs.SetVerbose(true)

	// Create tool runner
	runner := toolrunner.NewRunner(".")
	runner.SetVerbose(true)

	// Create backend generator
	backendGen := generators.NewBackendGenerator(fs, runner)

	// Generate service
	if err := backendGen.Create(ctx, serviceName, config, useLocalModules); err != nil {
		return fmt.Errorf("failed to create backend service: %w", err)
	}

	ui.Success("Backend service created: %s", serviceName)
	ui.Info("Next steps:")
	ui.Info("  1. Implement your service logic in internal/")
	ui.Info("  2. Add API definitions in api/")
	ui.Info("  3. Generate code: egg api generate")
	ui.Info("  4. Test locally: egg compose up")

	return nil
}

// runCreateFrontend executes the create frontend command.
//
// Parameters:
//   - cmd: Cobra command
//   - args: Command arguments
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Flutter project creation
func runCreateFrontend(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	serviceName := args[0]

	ui.Info("Creating frontend service: %s", serviceName)

	// Load configuration
	config, diags, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if diags.HasErrors() {
		ui.Error("Configuration validation failed:")
		for _, diag := range diags.Items() {
			if diag.Severity == configschema.SeverityError {
				ui.Error("  %s: %s", diag.Path, diag.Message)
			}
		}
		return fmt.Errorf("configuration validation failed")
	}

	// Check if service is defined in configuration
	serviceExists := false
	if _, exists := config.Frontend[serviceName]; exists {
		serviceExists = true
	}

	// If service doesn't exist, auto-register it
	if !serviceExists {
		ui.Info("Service '%s' not found in configuration, auto-registering...", serviceName)
		if err := autoRegisterFrontendService(serviceName, config); err != nil {
			return fmt.Errorf("failed to auto-register service: %w", err)
		}
		ui.Success("Service '%s' registered in egg.yaml", serviceName)
	} else if !forceCreate {
		// Service exists but no force flag
		return fmt.Errorf("frontend service '%s' already exists in configuration. Use --force to overwrite", serviceName)
	} else {
		ui.Info("Service '%s' already exists, overwriting due to --force flag", serviceName)
	}

	// Create project file system
	fs := projectfs.NewProjectFS(".")
	fs.SetVerbose(true)

	// Create tool runner
	runner := toolrunner.NewRunner(".")
	runner.SetVerbose(true)

	// Create frontend generator
	frontendGen := generators.NewFrontendGenerator(fs, runner)

	// Generate service
	if err := frontendGen.Create(ctx, serviceName, frontendPlatforms, config); err != nil {
		return fmt.Errorf("failed to create frontend service: %w", err)
	}

	ui.Success("Frontend service created: %s", serviceName)
	ui.Info("Next steps:")
	ui.Info("  1. Implement your Flutter app in lib/")
	ui.Info("  2. Add API client code")
	ui.Info("  3. Test locally: egg compose up")
	ui.Info("  4. Build for production: egg build")

	return nil
}

// loadConfig loads and validates the project configuration.
//
// Parameters:
//   - None
//
// Returns:
//   - *configschema.Config: Project configuration
//   - *configschema.Diagnostics: Validation diagnostics
//   - error: Loading error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Configuration parsing and validation
func loadConfig() (*configschema.Config, *configschema.Diagnostics, error) {
	config, diags := configschema.Load("egg.yaml")
	if config == nil {
		return nil, diags, fmt.Errorf("failed to load configuration")
	}

	return config, diags, nil
}

// autoRegisterBackendService automatically registers a backend service in egg.yaml.
//
// Parameters:
//   - serviceName: Name of the service to register
//   - config: Current configuration
//
// Returns:
//   - error: Registration error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - File I/O operation
func autoRegisterBackendService(serviceName string, config *configschema.Config) error {
	// Initialize backend map if nil
	if config.Backend == nil {
		config.Backend = make(map[string]configschema.BackendService)
	}

	// Add the service with default configuration
	config.Backend[serviceName] = configschema.BackendService{
		ImageName: serviceName,
	}

	// Save the updated configuration
	return saveConfig(config)
}

// autoRegisterFrontendService automatically registers a frontend service in egg.yaml.
//
// Parameters:
//   - serviceName: Name of the service to register
//   - config: Current configuration
//
// Returns:
//   - error: Registration error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - File I/O operation
func autoRegisterFrontendService(serviceName string, config *configschema.Config) error {
	// Initialize frontend map if nil
	if config.Frontend == nil {
		config.Frontend = make(map[string]configschema.FrontendService)
	}

	// Add the service with default configuration
	config.Frontend[serviceName] = configschema.FrontendService{
		Platforms: []string{"web"},
		ImageName: serviceName,
	}

	// Save the updated configuration
	return saveConfig(config)
}

// saveConfig saves the configuration to egg.yaml.
//
// Parameters:
//   - config: Configuration to save
//
// Returns:
//   - error: Save error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - File I/O operation
func saveConfig(config *configschema.Config) error {
	// Read current egg.yaml to preserve comments and formatting
	data, err := os.ReadFile("egg.yaml")
	if err != nil {
		return fmt.Errorf("failed to read egg.yaml: %w", err)
	}

	// Parse YAML to preserve structure
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return fmt.Errorf("failed to parse egg.yaml: %w", err)
	}

	// Update backend section
	if yamlData["backend"] == nil {
		yamlData["backend"] = make(map[string]interface{})
	}
	backendMap := yamlData["backend"].(map[string]interface{})
	for name, service := range config.Backend {
		backendMap[name] = map[string]interface{}{
			"image_name": service.ImageName,
		}
	}

	// Update frontend section
	if yamlData["frontend"] == nil {
		yamlData["frontend"] = make(map[string]interface{})
	}
	frontendMap := yamlData["frontend"].(map[string]interface{})
	for name, service := range config.Frontend {
		frontendMap[name] = map[string]interface{}{
			"platforms":  service.Platforms,
			"image_name": service.ImageName,
		}
	}

	// Write back to file
	output, err := yaml.Marshal(yamlData)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile("egg.yaml", output, 0644); err != nil {
		return fmt.Errorf("failed to write egg.yaml: %w", err)
	}

	return nil
}
