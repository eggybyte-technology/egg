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
//	egg check
package main

import (
	"fmt"

	"github.com/eggybyte-technology/egg/cli/internal/lint"
	"github.com/eggybyte-technology/egg/cli/internal/projectfs"
	"github.com/eggybyte-technology/egg/cli/internal/ui"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command.
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check project configuration and structure",
	Long: `Check project configuration and structure for issues.

This command provides:
- Configuration validation
- Project structure verification
- Service configuration checks
- Expression validation
- Best practice recommendations

Example:
  egg check`,
	RunE: runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)
}

// runCheck executes the check command.
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
//   - Project validation and linting
func runCheck(cmd *cobra.Command, args []string) error {
	ui.Info("Checking project configuration and structure...")

	// Load configuration
	config, _, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create project file system
	fs := projectfs.NewProjectFS(".")
	fs.SetVerbose(true)

	// Create linter
	linter := lint.NewLinter()

	// Run linting
	results, err := linter.Check(config, fs)
	if err != nil {
		return fmt.Errorf("failed to run linting: %w", err)
	}

	// Display results
	displayLintResults(results)

	// Check if there are errors
	if results.ErrorCount > 0 {
		return fmt.Errorf("project check failed with %d errors", results.ErrorCount)
	}

	// Display summary
	if results.WarningCount > 0 || results.InfoCount > 0 {
		ui.Warning("Project check completed with %d warnings and %d info messages",
			results.WarningCount, results.InfoCount)
	} else {
		ui.Success("Project check passed! No issues found.")
	}

	return nil
}

// displayLintResults displays linting results to the user.
//
// Parameters:
//   - results: Linting results
//
// Returns:
//   - None
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Result display
func displayLintResults(results *lint.LintResults) {
	// Group results by level
	errors := make([]lint.LintResult, 0)
	warnings := make([]lint.LintResult, 0)
	infos := make([]lint.LintResult, 0)

	for _, result := range results.Results {
		switch result.Level {
		case "error":
			errors = append(errors, result)
		case "warning":
			warnings = append(warnings, result)
		case "info":
			infos = append(infos, result)
		}
	}

	// Display errors
	if len(errors) > 0 {
		ui.Error("Errors found:")
		for _, result := range errors {
			ui.Error("  %s: %s", result.Path, result.Message)
			if result.Suggestion != "" {
				ui.Error("    Suggestion: %s", result.Suggestion)
			}
		}
		ui.Error("")
	}

	// Display warnings
	if len(warnings) > 0 {
		ui.Warning("Warnings found:")
		for _, result := range warnings {
			ui.Warning("  %s: %s", result.Path, result.Message)
			if result.Suggestion != "" {
				ui.Warning("    Suggestion: %s", result.Suggestion)
			}
		}
		ui.Warning("")
	}

	// Display info
	if len(infos) > 0 {
		ui.Info("Info messages:")
		for _, result := range infos {
			ui.Info("  %s: %s", result.Path, result.Message)
			if result.Suggestion != "" {
				ui.Info("    Suggestion: %s", result.Suggestion)
			}
		}
		ui.Info("")
	}
}
