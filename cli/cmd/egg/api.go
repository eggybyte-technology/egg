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
//	egg api init
//	egg api generate
package egg

import (
	"context"
	"fmt"

	"github.com/eggybyte-technology/egg/cli/internal/configschema"
	"github.com/eggybyte-technology/egg/cli/internal/generators"
	"github.com/eggybyte-technology/egg/cli/internal/projectfs"
	"github.com/eggybyte-technology/egg/cli/internal/toolrunner"
	"github.com/eggybyte-technology/egg/cli/internal/ui"
	"github.com/spf13/cobra"
)

// apiCmd represents the api command.
var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Manage API definitions",
	Long: `Manage API definitions and code generation.

This command provides:
- API configuration initialization
- Code generation from protobuf definitions
- Multi-language support (Go, Dart, TypeScript, OpenAPI)

Examples:
  egg api init
  egg api generate`,
}

// apiInitCmd represents the api init command.
var apiInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize API definitions",
	Long: `Initialize API definitions and configuration.

This command creates:
- buf.yaml configuration
- buf.gen.yaml generation rules
- Basic protobuf structure
- Multi-language code generation setup

Example:
  egg api init`,
	RunE: runAPIInit,
}

// apiGenerateCmd represents the api generate command.
var apiGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate code from API definitions",
	Long: `Generate code from protobuf definitions.

This command generates:
- Go code with Connect support
- Dart code for Flutter
- TypeScript types (pb only)
- OpenAPI specifications

Example:
  egg api generate`,
	RunE: runAPIGenerate,
}

func init() {
	rootCmd.AddCommand(apiCmd)
	apiCmd.AddCommand(apiInitCmd)
	apiCmd.AddCommand(apiGenerateCmd)
}

// runAPIInit executes the api init command.
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
//   - API configuration generation
func runAPIInit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	ui.Info("Initializing API definitions...")

	// Create project file system
	fs := projectfs.NewProjectFS(".")
	fs.SetVerbose(true)

	// Create tool runner
	runner := toolrunner.NewRunner(".")
	runner.SetVerbose(true)

	// Create API generator
	apiGen := generators.NewAPIGenerator(fs, runner)

	// Initialize API
	if err := apiGen.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize API: %w", err)
	}

	ui.Success("API definitions initialized")
	ui.Info("Next steps:")
	ui.Info("  1. Add your .proto files to api/")
	ui.Info("  2. Generate code: egg api generate")
	ui.Info("  3. Use generated code in your services")

	return nil
}

// runAPIGenerate executes the api generate command.
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
//   - Code generation from protobuf definitions
func runAPIGenerate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	ui.Info("Generating code from API definitions...")

	// Load configuration
	_, diags, err := loadConfig()
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

	// Create tool runner
	runner := toolrunner.NewRunner(".")
	runner.SetVerbose(true)

	// Create API generator
	apiGen := generators.NewAPIGenerator(fs, runner)

	// Install buf plugins if needed
	if err := installBufPlugins(ctx, runner); err != nil {
		return fmt.Errorf("failed to install buf plugins: %w", err)
	}

	// Generate code
	if err := apiGen.Generate(ctx); err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	ui.Success("Code generation completed")
	ui.Info("Generated files:")
	ui.Info("  - Go code: gen/go/")
	ui.Info("  - Dart code: gen/dart/")
	ui.Info("  - OpenAPI specs: gen/openapi/")

	return nil
}

// installBufPlugins installs required buf plugins.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//
// Returns:
//   - error: Installation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Plugin installation via go install
func installBufPlugins(ctx context.Context, runner *toolrunner.Runner) error {
	ui.Info("Installing buf plugins...")

	// Required plugins
	plugins := []string{
		"google.golang.org/protobuf/cmd/protoc-gen-go@latest",
		"connectrpc.com/connect/cmd/protoc-gen-connect-go@latest",
		"github.com/bufbuild/buf/cmd/protoc-gen-buf-lint@latest",
		"github.com/bufbuild/buf/cmd/protoc-gen-buf-breaking@latest",
		"github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest",
	}

	for _, plugin := range plugins {
		ui.Debug("Installing plugin: %s", plugin)
		if _, err := runner.Go(ctx, "install", plugin); err != nil {
			// Continue with other plugins if one fails
			ui.Warning("Failed to install plugin %s: %v", plugin, err)
		} else {
			ui.Debug("Successfully installed plugin: %s", plugin)
		}
	}

	ui.Success("Buf plugins installation completed")
	return nil
}
