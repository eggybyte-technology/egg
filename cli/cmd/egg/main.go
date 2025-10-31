// Package main provides the egg CLI tool entry point.
//
// Overview:
//   - Responsibility: CLI command parsing and execution
//   - Key Types: Cobra command structure
//   - Concurrency Model: Single-threaded CLI execution
//   - Error Semantics: Exit codes and user-friendly error messages
//   - Performance Notes: Fast startup, minimal memory footprint
//
// Usage:
//
//	egg [command] [flags]
package main

import (
	"os"

	"github.com/spf13/cobra"
	"go.eggybyte.com/egg/cli/internal/ui"
)

var (
	verbose        bool
	nonInteractive bool
	jsonOutput     bool
)

// rootCmd represents the base command when called without any subcommands.
//
// Parameters:
//   - None (global flags are parsed automatically)
//
// Returns:
//   - None (executes subcommands)
//
// Concurrency:
//   - Single-threaded CLI execution
//
// Performance:
//   - Fast startup, minimal initialization
var rootCmd = &cobra.Command{
	Use:   "egg",
	Short: "EggyByte platform CLI tool",
	Long: `EggyByte platform CLI tool for managing Connect-first, Kubernetes-native applications.

This tool provides commands for:
- Project initialization and scaffolding
- Service creation (backend/frontend)
- API generation and management
- Local development with Docker Compose
- Kubernetes deployment with Helm
- Build and deployment automation

All commands are designed to work with egg.yaml configuration files.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ui.SetVerbose(verbose)
		ui.SetNonInteractive(nonInteractive)
		ui.SetJSONOutput(jsonOutput)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
//
// Parameters:
//   - None (uses global variables)
//
// Returns:
//   - None (exits with appropriate code)
//
// Concurrency:
//   - Single-threaded execution
//
// Performance:
//   - Fast command resolution and execution
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ui.Error("Command failed: %v", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&nonInteractive, "non-interactive", false, "Disable interactive prompts")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	// Add version flags (--version and -v)
	rootCmd.Flags().BoolP("version", "v", false, "Show version information")
}

// main is the entry point for the egg CLI tool.
func main() {
	Execute()
}
