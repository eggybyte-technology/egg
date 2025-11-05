// Package main provides the egg CLI standalone init command.
//
// Overview:
//   - Responsibility: Initialize standalone service projects
//   - Key Types: Command handler for service initialization
//   - Concurrency Model: Sequential execution
//   - Error Semantics: User-friendly error messages
//   - Performance Notes: Fast scaffolding with templates
//
// Usage:
//
//	egg standalone init <name> --proto crud
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.eggybyte.com/egg/cli/internal/generators"
	"go.eggybyte.com/egg/cli/internal/projectfs"
	"go.eggybyte.com/egg/cli/internal/toolrunner"
	"go.eggybyte.com/egg/cli/internal/ui"
)

var (
	standaloneInitModulePath   string
	standaloneInitProto        string
	standaloneInitLocalModules bool
)

// standaloneInitCmd represents the standalone init command.
var standaloneInitCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Initialize a new standalone service",
	Long: `Initialize a new standalone Go backend service project.

This command creates a complete service structure following the egg framework
conventions, including directory layout, configuration, and templates.

The service uses the same structure as full egg project services but exists
independently without requiring an egg.yaml configuration file.

Version management:
- Default: Uses published egg framework versions (from CLI release)
- With --local-modules: Uses v0.0.0-dev for local development

Examples:
  egg standalone init my-service
  egg standalone init my-service --proto crud
  egg standalone init my-service --module-path github.com/myorg/my-service
  egg standalone init my-service --local-modules`,
	Args: cobra.ExactArgs(1),
	RunE: runStandaloneInit,
}

func init() {
	standaloneCmd.AddCommand(standaloneInitCmd)

	standaloneInitCmd.Flags().StringVar(&standaloneInitModulePath, "module-path", "", "Go module path (default: inferred from current directory)")
	standaloneInitCmd.Flags().StringVar(&standaloneInitProto, "proto", "echo", "Proto template: echo, crud, or none")
	standaloneInitCmd.Flags().BoolVar(&standaloneInitLocalModules, "local-modules", false, "Use local egg modules for development (v0.0.0-dev)")
}

// runStandaloneInit executes the standalone init command.
//
// Parameters:
//   - cmd: Cobra command
//   - args: Command arguments (service name)
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Service generation and module initialization
func runStandaloneInit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	serviceName := args[0]

	// Validate proto template
	if standaloneInitProto != "echo" && standaloneInitProto != "crud" && standaloneInitProto != "none" {
		ui.Error("Invalid proto template: %s (must be: echo, crud, or none)", standaloneInitProto)
		return fmt.Errorf("proto template validation failed")
	}

	ui.Info("Initializing standalone service: %s", serviceName)

	// Determine module path
	modulePath := standaloneInitModulePath
	if modulePath == "" {
		// Infer from service name
		modulePath = fmt.Sprintf("github.com/eggybyte-technology/%s", serviceName)
		ui.Info("Using inferred module path: %s", modulePath)
	}

	// Check if directory already exists
	if _, err := os.Stat(serviceName); err == nil {
		return fmt.Errorf("directory '%s' already exists. Choose a different name or remove the existing directory", serviceName)
	}

	// Create service directory
	if err := os.MkdirAll(serviceName, 0755); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	// Change to service directory for generation
	if err := os.Chdir(serviceName); err != nil {
		return fmt.Errorf("failed to change to service directory: %w", err)
	}

	// Get absolute path for project root
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create project file system
	fs := projectfs.NewProjectFS(rootDir)
	fs.SetVerbose(true)

	// Create tool runner
	runner := toolrunner.NewRunner(rootDir)
	runner.SetVerbose(true)

	// Create standalone generator
	gen := generators.NewStandaloneGenerator(fs, runner)

	// Generate service
	if err := gen.Create(ctx, serviceName, modulePath, standaloneInitProto, standaloneInitLocalModules); err != nil {
		return fmt.Errorf("failed to create standalone service: %w", err)
	}

	// Return to parent directory
	if err := os.Chdir(".."); err != nil {
		ui.Warning("Failed to return to parent directory: %v", err)
	}

	ui.Success("Standalone service initialized: %s", serviceName)
	ui.Info("")
	ui.Info("Service location: %s", filepath.Join(rootDir))
	ui.Info("")
	ui.Info("Next steps:")
	ui.Info("  1. cd %s", serviceName)
	ui.Info("  2. Configure your service in .env (copy from .env.example)")
	ui.Info("  3. Generate code: buf generate")
	ui.Info("  4. Build: egg standalone build")
	ui.Info("  5. Run: egg standalone run")

	return nil
}

