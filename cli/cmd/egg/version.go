// Package main provides the egg CLI version command.
//
// Overview:
//   - Responsibility: Display CLI version information
//   - Key Types: Version command handler
//   - Concurrency Model: Single-threaded CLI execution
//   - Error Semantics: No errors (version display only)
//   - Performance Notes: Fast version lookup
//
// Usage:
//
//	egg --version
//	egg -v
package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.eggybyte.com/egg/cli/internal/version"
)

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show egg CLI version information",
	Long: `Display version information for the egg CLI tool and egg framework.

This command shows:
  • CLI version, git commit hash, and build timestamp
  • Egg framework version
  • Go runtime version`,
	Run: runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
	
	// Add --version and -v flags to root command
	rootCmd.Version = version.GetVersionString()
	rootCmd.SetVersionTemplate(`{{.Version}}
`)
}

// runVersion executes the version command.
//
// Parameters:
//   - cmd: Cobra command
//   - args: Command arguments
//
// Returns:
//   - None (writes to stdout)
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Fast version string formatting
func runVersion(cmd *cobra.Command, args []string) {
	fmt.Println(version.GetFullVersionInfo())
}

