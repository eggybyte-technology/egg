// Package main provides the egg CLI standalone build command.
//
// Overview:
//   - Responsibility: Build Docker images for standalone services
//   - Key Types: Command handler for image building
//   - Concurrency Model: Sequential execution
//   - Error Semantics: User-friendly error messages
//   - Performance Notes: Multi-platform build support with buildx
//
// Usage:
//
//	egg standalone build --push
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"go.eggybyte.com/egg/cli/internal/ui"
)

var (
	standaloneBuildPush     bool
	standaloneBuildTag      string
	standaloneBuildPlatform string
)

// standaloneBuildCmd represents the standalone build command.
var standaloneBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build Docker image for standalone service",
	Long: `Build Docker image for standalone service with multi-platform support.

This command uses docker buildx to build images for multiple architectures
(arm64, amd64) and uses a fixed builder named 'egg-builder' for all builds.

The image is built using remote egg framework versions (not local replace directives)
to ensure the image works independently of the local development environment.

Examples:
  egg standalone build                              # Build for local platform
  egg standalone build --push                       # Build and push multi-platform
  egg standalone build --tag my-service:v1.0.0     # Build with custom tag
  egg standalone build --platform linux/amd64       # Build for specific platform`,
	RunE: runStandaloneBuild,
}

func init() {
	standaloneCmd.AddCommand(standaloneBuildCmd)

	standaloneBuildCmd.Flags().BoolVar(&standaloneBuildPush, "push", false, "Push image to registry (required for multi-platform)")
	standaloneBuildCmd.Flags().StringVar(&standaloneBuildTag, "tag", "", "Image tag (default: <service-name>:latest)")
	standaloneBuildCmd.Flags().StringVar(&standaloneBuildPlatform, "platform", "linux/amd64,linux/arm64", "Target platforms")
}

// runStandaloneBuild executes the standalone build command.
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
//   - Multi-platform build time depends on image complexity
func runStandaloneBuild(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Verify we're in a standalone service directory
	if _, err := os.Stat("go.mod"); err != nil {
		return fmt.Errorf("go.mod not found - are you in a standalone service directory?")
	}
	if _, err := os.Stat("Dockerfile"); err != nil {
		return fmt.Errorf("Dockerfile not found - are you in a standalone service directory?")
	}

	// Read service name from go.mod
	serviceName, modulePath, err := readGoModInfo()
	if err != nil {
		return fmt.Errorf("failed to read go.mod: %w", err)
	}

	ui.Info("Building Docker image for standalone service: %s", serviceName)

	// Determine image tag
	imageTag := standaloneBuildTag
	if imageTag == "" {
		imageTag = fmt.Sprintf("%s:latest", serviceName)
	}

	// Ensure buildx builder exists
	if err := ensureBuildxBuilder(ctx); err != nil {
		return fmt.Errorf("failed to ensure buildx builder: %w", err)
	}

	// Determine if multi-platform build
	isMultiPlatform := strings.Contains(standaloneBuildPlatform, ",")

	// Multi-platform builds MUST use --push
	if isMultiPlatform && !standaloneBuildPush {
		ui.Warning("Multi-platform builds require --push flag (buildx limitation)")
		ui.Info("Switching to single platform: linux/amd64")
		standaloneBuildPlatform = "linux/amd64"
		isMultiPlatform = false
	}

	// Build arguments
	buildArgs := []string{
		fmt.Sprintf("SERVICE_NAME=%s", serviceName),
		fmt.Sprintf("MODULE_PATH=%s", modulePath),
		fmt.Sprintf("GO_VERSION=%s", "1.25.1"),
	}

	if isMultiPlatform {
		// Multi-platform build with push
		ui.Info("Building multi-platform Docker image for %s...", standaloneBuildPlatform)
		if err := buildMultiPlatformImageStandalone(ctx, imageTag, standaloneBuildPlatform, buildArgs); err != nil {
			return fmt.Errorf("failed to build and push multi-platform image: %w", err)
		}
		ui.Success("Multi-platform image built and pushed: %s (platforms: %s)", imageTag, standaloneBuildPlatform)
	} else {
		// Single platform build
		ui.Info("Building Docker image for %s...", standaloneBuildPlatform)
		if err := buildDockerImageStandalone(ctx, imageTag, standaloneBuildPlatform, buildArgs); err != nil {
			return fmt.Errorf("failed to build Docker image: %w", err)
		}
		ui.Success("Image built: %s (platform: %s)", imageTag, standaloneBuildPlatform)

		// Push if requested (single platform)
		if standaloneBuildPush {
			ui.Info("Pushing image to registry...")
			if err := pushDockerImage(ctx, imageTag); err != nil {
				return fmt.Errorf("failed to push image: %w", err)
			}
			ui.Success("Image pushed successfully")
		}
	}

	return nil
}

// ensureBuildxBuilder ensures the egg-builder buildx instance exists.
//
// Parameters:
//   - ctx: Context for cancellation
//
// Returns:
//   - error: Builder creation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Fast check, minimal overhead if builder exists
func ensureBuildxBuilder(ctx context.Context) error {
	// Check if builder exists
	cmd := exec.CommandContext(ctx, "docker", "buildx", "inspect", "egg-builder")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err == nil {
		ui.Debug("Using existing buildx builder: egg-builder")
		return nil // Builder exists
	}

	// Create builder
	ui.Info("Creating buildx builder 'egg-builder'...")
	cmd = exec.CommandContext(ctx, "docker", "buildx", "create", "--name", "egg-builder", "--use")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create buildx builder: %w", err)
	}

	ui.Success("Buildx builder created: egg-builder")
	return nil
}

// buildDockerImageStandalone builds a single-platform Docker image.
//
// Parameters:
//   - ctx: Context for cancellation
//   - tag: Image tag
//   - platform: Target platform
//   - buildArgs: Build arguments
//
// Returns:
//   - error: Build error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Build time depends on image complexity
func buildDockerImageStandalone(ctx context.Context, tag, platform string, buildArgs []string) error {
	args := []string{
		"buildx", "build",
		"--builder", "egg-builder",
		"-f", "Dockerfile",
		"-t", tag,
	}

	// Always specify platform to ensure correct base image architecture is pulled
	if platform != "" {
		args = append(args, "--platform", platform)
		args = append(args, "--pull")
	}

	// Load image into local docker daemon for single platform builds
	if platform != "" && !strings.Contains(platform, ",") {
		args = append(args, "--load")
	}

	// Add build arguments
	for _, arg := range buildArgs {
		args = append(args, "--build-arg", arg)
	}

	args = append(args, ".")

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	ui.Debug("Running: docker %s", strings.Join(args, " "))

	return cmd.Run()
}

// buildMultiPlatformImageStandalone builds a multi-platform Docker image and pushes it.
//
// Parameters:
//   - ctx: Context for cancellation
//   - tag: Image tag
//   - platforms: Comma-separated list of platforms
//   - buildArgs: Build arguments
//
// Returns:
//   - error: Build error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Build time depends on image complexity and number of platforms
func buildMultiPlatformImageStandalone(ctx context.Context, tag, platforms string, buildArgs []string) error {
	// Verify buildx is available
	if err := exec.CommandContext(ctx, "docker", "buildx", "version").Run(); err != nil {
		return fmt.Errorf("docker buildx not available - please ensure Docker Buildx is installed")
	}

	args := []string{
		"buildx", "build",
		"--builder", "egg-builder",
		"-f", "Dockerfile",
		"-t", tag,
		"--platform", platforms,
		"--pull",
		"--push",
	}

	for _, arg := range buildArgs {
		args = append(args, "--build-arg", arg)
	}

	args = append(args, ".")

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	ui.Debug("Running: docker %s", strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("buildx multi-platform build failed: %w", err)
	}

	return nil
}

// readGoModInfo reads service name and module path from go.mod.
//
// Parameters:
//   - None
//
// Returns:
//   - serviceName: Service name (last component of module path)
//   - modulePath: Full module path
//   - error: Read error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Fast file read and parse
func readGoModInfo() (string, string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", "", fmt.Errorf("failed to read go.mod: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module"))
			// Extract service name from module path
			parts := strings.Split(modulePath, "/")
			serviceName := parts[len(parts)-1]
			return serviceName, modulePath, nil
		}
	}

	return "", "", fmt.Errorf("module declaration not found in go.mod")
}

// detectLocalPlatform detects the local platform architecture.
//
// Parameters:
//   - None
//
// Returns:
//   - string: Platform string (e.g., "linux/arm64")
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Constant time
func detectLocalPlatformStandalone() string {
	if runtime.GOARCH == "arm64" {
		return "linux/arm64"
	}
	return "linux/amd64"
}

