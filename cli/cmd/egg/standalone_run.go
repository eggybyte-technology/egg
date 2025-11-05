// Package main provides the egg CLI standalone run command.
//
// Overview:
//   - Responsibility: Run standalone services locally with .env configuration
//   - Key Types: Command handler for local service execution
//   - Concurrency Model: Sequential execution
//   - Error Semantics: User-friendly error messages
//   - Performance Notes: Direct go run execution
//
// Usage:
//
//	egg standalone run
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"go.eggybyte.com/egg/cli/internal/envloader"
	"go.eggybyte.com/egg/cli/internal/ui"
)

var (
	standaloneRunEnvFile string
)

// standaloneRunCmd represents the standalone run command.
var standaloneRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run standalone service locally",
	Long: `Run standalone service locally with environment variables from .env file.

This command loads environment variables from .env file and runs the service
using 'go run'. The service runs in the foreground and can be stopped with Ctrl+C.

Examples:
  egg standalone run                    # Run with .env file
  egg standalone run --env-file .env.local  # Run with custom env file`,
	RunE: runStandaloneRun,
}

func init() {
	standaloneCmd.AddCommand(standaloneRunCmd)

	standaloneRunCmd.Flags().StringVar(&standaloneRunEnvFile, "env-file", ".env", "Environment file path")
}

// runStandaloneRun executes the standalone run command.
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
//   - Direct go run execution
func runStandaloneRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Verify we're in a standalone service directory
	if _, err := os.Stat("go.mod"); err != nil {
		return fmt.Errorf("go.mod not found - are you in a standalone service directory?")
	}
	if _, err := os.Stat("cmd/server/main.go"); err != nil {
		return fmt.Errorf("cmd/server/main.go not found - are you in a standalone service directory?")
	}

	// Load environment variables from .env file
	var envMap map[string]string

	if _, statErr := os.Stat(standaloneRunEnvFile); statErr == nil {
		ui.Info("Loading environment variables from %s...", standaloneRunEnvFile)
		var err error
		envMap, err = envloader.LoadEnvFile(standaloneRunEnvFile)
		if err != nil {
			return fmt.Errorf("failed to load .env file: %w", err)
		}
		ui.Success("Loaded %d environment variables", len(envMap))
	} else {
		ui.Warning("Environment file %s not found, using OS environment only", standaloneRunEnvFile)
		envMap = make(map[string]string)
	}

	// Merge with OS environment
	envSlice := envloader.MergeWithOS(envMap)

	// Run service with go run
	ui.Info("Starting service with 'go run cmd/server/main.go'...")
	ui.Info("Press Ctrl+C to stop")

	goCmd := exec.CommandContext(ctx, "go", "run", "cmd/server/main.go")
	goCmd.Env = envSlice
	goCmd.Stdout = os.Stdout
	goCmd.Stderr = os.Stderr
	goCmd.Stdin = os.Stdin

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start the command
	if err := goCmd.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Wait for signal or process completion
	go func() {
		<-sigChan
		ui.Info("Stopping service...")
		if goCmd.Process != nil {
			goCmd.Process.Signal(os.Interrupt)
		}
	}()

	// Wait for process to complete
	if err := goCmd.Wait(); err != nil {
		// Exit code 2 (Ctrl+C) is expected
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 2 || exitErr.ExitCode() == 130 {
				ui.Info("Service stopped")
				return nil
			}
		}
		return fmt.Errorf("service failed: %w", err)
	}

	return nil
}

