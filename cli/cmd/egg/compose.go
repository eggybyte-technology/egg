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
//	egg compose up [--detached]
//	egg compose down
//	egg compose logs [--service <name>]
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/eggybyte-technology/egg/cli/internal/configschema"
	"github.com/eggybyte-technology/egg/cli/internal/generators"
	"github.com/eggybyte-technology/egg/cli/internal/projectfs"
	"github.com/eggybyte-technology/egg/cli/internal/ref"
	"github.com/eggybyte-technology/egg/cli/internal/render/compose"
	"github.com/eggybyte-technology/egg/cli/internal/toolrunner"
	"github.com/eggybyte-technology/egg/cli/internal/ui"
	"github.com/spf13/cobra"
)

// composeCmd represents the compose command.
var composeCmd = &cobra.Command{
	Use:   "compose",
	Short: "Manage Docker Compose services",
	Long: `Manage Docker Compose services for local development.

This command provides:
- Service orchestration with Docker Compose
- Database integration (MySQL)
- Environment variable management
- Log aggregation and monitoring

Examples:
  egg compose up
  egg compose up --detached
  egg compose down
  egg compose logs --service user-service`,
}

// composeUpCmd represents the compose up command.
var composeUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Start services",
	Long: `Start all services with Docker Compose.

This command:
- Renders compose.yaml from egg configuration
- Starts all backend and frontend services
- Attaches MySQL database if enabled
- Sets up service dependencies

Example:
  egg compose up
  egg compose up --detached`,
	RunE: runComposeUp,
}

// composeDownCmd represents the compose down command.
var composeDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop services",
	Long: `Stop all services and clean up.

This command:
- Stops all running services
- Removes containers and networks
- Cleans up volumes (optional)

Example:
  egg compose down`,
	RunE: runComposeDown,
}

// composeLogsCmd represents the compose logs command.
var composeLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show service logs",
	Long: `Show logs from services.

This command:
- Displays logs from all services
- Filters logs by service name
- Follows log output in real-time

Example:
  egg compose logs
  egg compose logs --service user-service`,
	RunE: runComposeLogs,
}

// composeGenerateCmd represents the compose generate command.
var composeGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate Docker Compose configuration",
	Long: `Generate Docker Compose configuration from egg.yaml.

This command:
- Creates docker-compose.yaml with all services
- Configures database connections
- Sets up service dependencies
- Uses local eggybyte-go-alpine base image

Example:
  egg compose generate`,
	RunE: runComposeGenerate,
}

var (
	detached      bool
	serviceFilter string
	followLogs    bool
)

func init() {
	rootCmd.AddCommand(composeCmd)
	composeCmd.AddCommand(composeUpCmd)
	composeCmd.AddCommand(composeDownCmd)
	composeCmd.AddCommand(composeLogsCmd)
	composeCmd.AddCommand(composeGenerateCmd)

	composeUpCmd.Flags().BoolVar(&detached, "detached", false, "Run in detached mode")
	composeLogsCmd.Flags().StringVar(&serviceFilter, "service", "", "Filter logs by service name")
	composeLogsCmd.Flags().BoolVarP(&followLogs, "follow", "f", false, "Follow log output")
}

// runComposeUp executes the compose up command.
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
//   - Docker Compose rendering and service startup
func runComposeUp(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	ui.Info("Starting services with Docker Compose...")

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

	// Create project file system
	fs := projectfs.NewProjectFS(".")
	fs.SetVerbose(true)

	// Create reference parser
	refParser := ref.NewParser()

	// Create Compose renderer
	composeRenderer := compose.NewRenderer(fs, refParser)

	// Render Compose configuration
	if err := composeRenderer.Render(config); err != nil {
		return fmt.Errorf("failed to render Compose configuration: %w", err)
	}

	// Create tool runner
	runner := toolrunner.NewRunner(".")
	runner.SetVerbose(true)

	// Start services
	if err := startComposeServices(ctx, runner, detached); err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}

	ui.Success("Services started successfully!")

	if !detached {
		ui.Info("Press Ctrl+C to stop services")
		// Wait for interrupt
		select {
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// runComposeDown executes the compose down command.
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
//   - Docker Compose service shutdown
func runComposeDown(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	ui.Info("Stopping services...")

	// Create tool runner
	runner := toolrunner.NewRunner(".")
	runner.SetVerbose(true)

	// Stop services
	if err := stopComposeServices(ctx, runner); err != nil {
		return fmt.Errorf("failed to stop services: %w", err)
	}

	ui.Success("Services stopped successfully!")
	return nil
}

// runComposeLogs executes the compose logs command.
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
//   - Docker Compose log retrieval
func runComposeLogs(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	ui.Info("Showing service logs...")

	// Create tool runner
	runner := toolrunner.NewRunner(".")
	runner.SetVerbose(true)

	// Show logs
	if err := showComposeLogs(ctx, runner, serviceFilter, followLogs); err != nil {
		return fmt.Errorf("failed to show logs: %w", err)
	}

	return nil
}

// startComposeServices starts Docker Compose services.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//   - detached: Whether to run in detached mode
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Docker Compose service startup
func startComposeServices(ctx context.Context, runner *toolrunner.Runner, detached bool) error {
	composeFile := "deploy/compose.yaml"

	// Check if compose file exists
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("compose file not found: %s", composeFile)
	}

	// Build command arguments
	args := []string{"-f", composeFile, "up"}
	if detached {
		args = append(args, "-d")
	}

	// Execute docker-compose command
	result, err := runner.Docker(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}

	if runner.GetVerbose() {
		ui.Debug("Docker Compose output: %s", result.Stdout)
	}

	return nil
}

// stopComposeServices stops Docker Compose services.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Docker Compose service shutdown
func stopComposeServices(ctx context.Context, runner *toolrunner.Runner) error {
	composeFile := "deploy/compose.yaml"

	// Check if compose file exists
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("compose file not found: %s", composeFile)
	}

	// Execute docker-compose command
	args := []string{"-f", composeFile, "down"}
	result, err := runner.Docker(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to stop services: %w", err)
	}

	if runner.GetVerbose() {
		ui.Debug("Docker Compose output: %s", result.Stdout)
	}

	return nil
}

// showComposeLogs shows Docker Compose service logs.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//   - serviceFilter: Service name filter
//   - follow: Whether to follow logs
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Docker Compose log retrieval
func showComposeLogs(ctx context.Context, runner *toolrunner.Runner, serviceFilter string, follow bool) error {
	composeFile := "deploy/compose.yaml"

	// Check if compose file exists
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("compose file not found: %s", composeFile)
	}

	// Build command arguments
	args := []string{"-f", composeFile, "logs"}
	if follow {
		args = append(args, "-f")
	}
	if serviceFilter != "" {
		args = append(args, serviceFilter)
	}

	// Execute docker-compose command
	result, err := runner.Docker(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to show logs: %w", err)
	}

	// Print logs to stdout
	fmt.Print(result.Stdout)

	return nil
}

// runComposeGenerate executes the compose generate command.
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
//   - Docker Compose configuration generation
func runComposeGenerate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	ui.Info("Generating Docker Compose configuration...")

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

	// Create project file system
	fs := projectfs.NewProjectFS(".")
	fs.SetVerbose(true)

	// Create backend generator
	runner := toolrunner.NewRunner(".")
	runner.SetVerbose(true)
	backendGen := generators.NewBackendGenerator(fs, runner)

	// Generate docker-compose.yaml
	if err := backendGen.GenerateCompose(ctx, config); err != nil {
		return fmt.Errorf("failed to generate docker-compose.yaml: %w", err)
	}

	ui.Success("Docker Compose configuration generated successfully!")
	ui.Info("Next steps:")
	ui.Info("  1. Build base image: docker build -t localhost:5000/eggybyte-go-alpine:latest -f build/Dockerfile.eggybyte-go-alpine .")
	ui.Info("  2. Build backend services: go build -o server ./cmd/server (in each backend service directory)")
	ui.Info("  3. Start services: docker-compose up")

	return nil
}
