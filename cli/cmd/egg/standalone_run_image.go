// Package main provides the egg CLI standalone run-image command.
//
// Overview:
//   - Responsibility: Run standalone service Docker containers with .env configuration
//   - Key Types: Command handler for container execution
//   - Concurrency Model: Sequential execution
//   - Error Semantics: User-friendly error messages
//   - Performance Notes: Docker container startup
//
// Usage:
//
//	egg standalone run-image <image-name>
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
	standaloneRunImageEnvFile string
	standaloneRunImagePort    int
	standaloneRunImageName    string
)

// standaloneRunImageCmd represents the standalone run-image command.
var standaloneRunImageCmd = &cobra.Command{
	Use:   "run-image <image-name>",
	Short: "Run standalone service Docker container",
	Long: `Run standalone service Docker container with environment variables from .env file.

This command loads environment variables from .env file and runs the Docker
container. The container runs in the foreground and can be stopped with Ctrl+C.

Port mapping:
- HTTP port (from .env HTTP_PORT or default 8080)
- Health port (from .env HEALTH_PORT or default 8081)
- Metrics port (from .env METRICS_PORT or default 9091)

Examples:
  egg standalone run-image my-service:latest
  egg standalone run-image my-service:v1.0.0 --env-file .env.prod
  egg standalone run-image my-service:latest --port 8080 --name my-service-1`,
	Args: cobra.ExactArgs(1),
	RunE: runStandaloneRunImage,
}

func init() {
	standaloneCmd.AddCommand(standaloneRunImageCmd)

	standaloneRunImageCmd.Flags().StringVar(&standaloneRunImageEnvFile, "env-file", ".env", "Environment file path")
	standaloneRunImageCmd.Flags().IntVar(&standaloneRunImagePort, "port", 8080, "HTTP port mapping (host:container)")
	standaloneRunImageCmd.Flags().StringVar(&standaloneRunImageName, "name", "", "Container name (default: auto-generated)")
}

// runStandaloneRunImage executes the standalone run-image command.
//
// Parameters:
//   - cmd: Cobra command
//   - args: Command arguments (image name)
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Docker container startup time
func runStandaloneRunImage(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	imageName := args[0]

	// Load environment variables from .env file
	var envMap map[string]string

	if _, statErr := os.Stat(standaloneRunImageEnvFile); statErr == nil {
		ui.Info("Loading environment variables from %s...", standaloneRunImageEnvFile)
		var err error
		envMap, err = envloader.LoadEnvFile(standaloneRunImageEnvFile)
		if err != nil {
			return fmt.Errorf("failed to load .env file: %w", err)
		}
		ui.Success("Loaded %d environment variables", len(envMap))
	} else {
		ui.Warning("Environment file %s not found, using defaults", standaloneRunImageEnvFile)
		envMap = make(map[string]string)
	}

	// Extract ports from environment or use defaults
	httpPort := getEnvOrDefault(envMap, "HTTP_PORT", "8080")
	healthPort := getEnvOrDefault(envMap, "HEALTH_PORT", "8081")
	metricsPort := getEnvOrDefault(envMap, "METRICS_PORT", "9091")

	// Build docker run command
	dockerArgs := []string{"run", "--rm", "-it"}

	// Add container name if specified
	if standaloneRunImageName != "" {
		dockerArgs = append(dockerArgs, "--name", standaloneRunImageName)
	}

	// Add port mappings
	dockerArgs = append(dockerArgs,
		"-p", fmt.Sprintf("%s:%s", httpPort, httpPort),
		"-p", fmt.Sprintf("%s:%s", healthPort, healthPort),
		"-p", fmt.Sprintf("%s:%s", metricsPort, metricsPort),
	)

	// Add environment variables
	for key, value := range envMap {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Add image name
	dockerArgs = append(dockerArgs, imageName)

	ui.Info("Starting container: %s", imageName)
	ui.Info("Port mappings: HTTP=%s, Health=%s, Metrics=%s", httpPort, healthPort, metricsPort)
	ui.Info("Press Ctrl+C to stop")

	dockerCmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr
	dockerCmd.Stdin = os.Stdin

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start the command
	if err := dockerCmd.Start(); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Wait for signal or process completion
	go func() {
		<-sigChan
		ui.Info("Stopping container...")
		if dockerCmd.Process != nil {
			dockerCmd.Process.Signal(os.Interrupt)
		}
	}()

	// Wait for process to complete
	if err := dockerCmd.Wait(); err != nil {
		// Exit code 2 or 130 (Ctrl+C) is expected
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 2 || exitErr.ExitCode() == 130 {
				ui.Info("Container stopped")
				return nil
			}
		}
		return fmt.Errorf("container failed: %w", err)
	}

	ui.Info("Container exited")
	return nil
}

// getEnvOrDefault returns environment variable value or default.
//
// Parameters:
//   - envMap: Environment variables map
//   - key: Environment variable key
//   - defaultValue: Default value if key not found
//
// Returns:
//   - string: Environment variable value or default
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - O(1) map lookup
func getEnvOrDefault(envMap map[string]string, key, defaultValue string) string {
	if value, ok := envMap[key]; ok && value != "" {
		return value
	}
	return defaultValue
}

