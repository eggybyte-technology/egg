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
//	egg build [--push] [--version vX.Y.Z] [--subset svc1,svc2]
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/eggybyte-technology/egg/cli/internal/configschema"
	"github.com/eggybyte-technology/egg/cli/internal/toolrunner"
	"github.com/eggybyte-technology/egg/cli/internal/ui"
	"github.com/spf13/cobra"
)

// buildCmd represents the build command.
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build and push Docker images",
	Long: `Build and push Docker images for all services.

This command provides:
- Multi-platform Docker image building
- Image tagging and versioning
- Registry push support
- Selective service building

Examples:
  egg build
  egg build --push
  egg build --version v1.2.3
  egg build --subset user-service,admin-portal`,
	RunE: runBuild,
}

var (
	pushImages     bool
	buildVersion   string
	buildSubset    string
	buildPlatforms string
	useBuildx      bool
)

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().BoolVar(&pushImages, "push", false, "Push images to registry after building")
	buildCmd.Flags().StringVar(&buildVersion, "version", "", "Image version tag")
	buildCmd.Flags().StringVar(&buildSubset, "subset", "", "Comma-separated list of services to build")
	buildCmd.Flags().StringVar(&buildPlatforms, "platforms", "linux/amd64,linux/arm64", "Target platforms for multi-arch builds (comma-separated)")
	buildCmd.Flags().BoolVar(&useBuildx, "buildx", true, "Use Docker buildx for multi-platform builds")
}

// runBuild executes the build command.
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
//   - Docker image building and pushing
func runBuild(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	ui.Info("Building Docker images...")

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

	// Create tool runner
	runner := toolrunner.NewRunner(".")
	runner.SetVerbose(true)

	// Determine services to build
	servicesToBuild, err := determineServicesToBuild(config, buildSubset)
	if err != nil {
		return fmt.Errorf("failed to determine services to build: %w", err)
	}

	// Set default version if not provided
	if buildVersion == "" {
		buildVersion = config.Version
	}

	// Build runtime image first
	if err := buildRuntimeImage(ctx, runner, useBuildx, buildPlatforms, pushImages); err != nil {
		return fmt.Errorf("failed to build runtime image: %w", err)
	}

	// Build images
	if err := buildImages(ctx, runner, config, servicesToBuild, buildVersion, useBuildx, buildPlatforms, pushImages); err != nil {
		return fmt.Errorf("failed to build images: %w", err)
	}

	// Push images if requested
	if pushImages {
		if err := pushImagesToRegistry(ctx, runner, config, servicesToBuild, buildVersion); err != nil {
			return fmt.Errorf("failed to push images: %w", err)
		}
	}

	ui.Success("Docker images built successfully!")

	if pushImages {
		ui.Info("Images pushed to registry: %s", config.DockerRegistry)
	} else {
		ui.Info("Images built locally")
		ui.Info("To push images: egg build --push")
	}

	return nil
}

// determineServicesToBuild determines which services to build based on subset.
//
// Parameters:
//   - config: Project configuration
//   - subset: Comma-separated service names
//
// Returns:
//   - []string: List of service names to build
//   - error: Determination error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Service list processing
func determineServicesToBuild(config *configschema.Config, subset string) ([]string, error) {
	var services []string

	if subset != "" {
		// Parse subset
		serviceNames := strings.Split(subset, ",")
		for _, name := range serviceNames {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}

			// Check if service exists
			if _, exists := config.Backend[name]; exists {
				services = append(services, name)
			} else if _, exists := config.Frontend[name]; exists {
				services = append(services, name)
			} else {
				return nil, fmt.Errorf("service not found: %s", name)
			}
		}
	} else {
		// Build all services
		for name := range config.Backend {
			services = append(services, name)
		}
		for name := range config.Frontend {
			services = append(services, name)
		}
	}

	return services, nil
}

// buildImages builds Docker images for specified services.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//   - config: Project configuration
//   - services: List of service names
//   - version: Image version tag
//   - useBuildx: Whether to use buildx for multi-platform builds
//   - platforms: Target platforms (comma-separated)
//   - push: Whether to push images to registry
//
// Returns:
//   - error: Build error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Docker image building
func buildImages(ctx context.Context, runner *toolrunner.Runner, config *configschema.Config, services []string, version string, useBuildx bool, platforms string, push bool) error {
	for _, serviceName := range services {
		// Determine service type and configuration
		var serviceType string
		var imageName string

		if backendService, exists := config.Backend[serviceName]; exists {
			serviceType = "backend"
			imageName = backendService.ImageName
		} else if frontendService, exists := config.Frontend[serviceName]; exists {
			serviceType = "frontend"
			imageName = frontendService.ImageName
		} else {
			return fmt.Errorf("service not found: %s", serviceName)
		}

		// Build image
		if err := buildServiceImage(ctx, runner, config, serviceName, serviceType, imageName, version, useBuildx, platforms, push); err != nil {
			return fmt.Errorf("failed to build image for %s: %w", serviceName, err)
		}
	}

	return nil
}

// buildServiceImage builds a Docker image for a specific service.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//   - config: Project configuration
//   - serviceName: Service name
//   - serviceType: Service type (backend/frontend)
//   - imageName: Image name
//   - version: Image version tag
//   - useBuildx: Whether to use buildx for multi-platform builds
//   - platforms: Target platforms (comma-separated)
//   - push: Whether to push images to registry
//
// Returns:
//   - error: Build error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Single Docker image build
func buildServiceImage(ctx context.Context, runner *toolrunner.Runner, config *configschema.Config, serviceName, serviceType, imageName, version string, useBuildx bool, platforms string, push bool) error {
	ui.Info("Building %s service: %s", serviceType, serviceName)

	// Build locally first
	if err := buildServiceLocally(ctx, runner, serviceName, serviceType); err != nil {
		return fmt.Errorf("failed to build service locally: %w", err)
	}

	// Determine build context and dockerfile
	var buildContext string
	var dockerfile string

	if serviceType == "backend" {
		// Use root directory as build context to access bin directory
		buildContext = "."
		dockerfile = "build/Dockerfile.backend"
	} else {
		buildContext = fmt.Sprintf("./frontend/%s", serviceName)
		dockerfile = "build/Dockerfile.frontend"
	}

	// Build image name with registry and version
	fullImageName := fmt.Sprintf("%s/%s:%s", config.DockerRegistry, imageName, version)

	// Build Docker image
	if useBuildx {
		// Use buildx for multi-platform builds
		// Note: --load only works with single platform, so we use --push if specified
		// or build without --load/--push for multi-platform
		load := !push && !strings.Contains(platforms, ",")

		ui.Info("Building with buildx for platforms: %s", platforms)
		buildArgs := map[string]string{
			"BINARY_NAME": serviceName,
		}
		if err := runner.DockerBuildxWithArgs(ctx, fullImageName, dockerfile, buildContext, platforms, push, load, buildArgs); err != nil {
			return fmt.Errorf("failed to build Docker image with buildx: %w", err)
		}
	} else {
		// Use regular docker build
		buildArgs := map[string]string{
			"BINARY_NAME": serviceName,
		}
		if err := runner.DockerBuildWithArgs(ctx, fullImageName, dockerfile, buildContext, buildArgs); err != nil {
			return fmt.Errorf("failed to build Docker image: %w", err)
		}
	}

	ui.Success("Built image: %s", fullImageName)
	return nil
}

// buildServiceLocally builds the service locally before Docker packaging.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//   - serviceName: Service name
//   - serviceType: Service type (backend/frontend)
//
// Returns:
//   - error: Build error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Local build operations
func buildServiceLocally(ctx context.Context, runner *toolrunner.Runner, serviceName, serviceType string) error {
	if serviceType == "backend" {
		// Build Go binary to bin directory
		serviceDir := fmt.Sprintf("./backend/%s", serviceName)
		serviceRunner := toolrunner.NewRunner(serviceDir)
		serviceRunner.SetVerbose(true)

		// Ensure bin directory exists
		if _, err := runner.Exec(ctx, "mkdir", "-p", "bin"); err != nil {
			return fmt.Errorf("failed to create bin directory: %w", err)
		}

		// Build the binary to bin directory
		binaryPath := fmt.Sprintf("../../bin/%s", serviceName)
		if _, err := serviceRunner.Go(ctx, "build", "-o", binaryPath, "./cmd/server"); err != nil {
			return fmt.Errorf("failed to build Go binary: %w", err)
		}

		ui.Debug("Go binary built: bin/%s", serviceName)
	} else {
		// Build Flutter web assets
		serviceDir := fmt.Sprintf("./frontend/%s", serviceName)
		serviceRunner := toolrunner.NewRunner(serviceDir)
		serviceRunner.SetVerbose(true)

		// Build Flutter web
		if _, err := serviceRunner.Flutter(ctx, "build", "web"); err != nil {
			return fmt.Errorf("failed to build Flutter web: %w", err)
		}

		ui.Debug("Flutter web built: %s/build/web", serviceDir)
	}

	return nil
}

// pushImagesToRegistry pushes Docker images to registry.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//   - config: Project configuration
//   - services: List of service names
//   - version: Image version tag
//
// Returns:
//   - error: Push error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Docker image pushing
func pushImagesToRegistry(ctx context.Context, runner *toolrunner.Runner, config *configschema.Config, services []string, version string) error {
	ui.Info("Pushing images to registry...")

	for _, serviceName := range services {
		// Determine image name
		var imageName string

		if backendService, exists := config.Backend[serviceName]; exists {
			imageName = backendService.ImageName
		} else if frontendService, exists := config.Frontend[serviceName]; exists {
			imageName = frontendService.ImageName
		} else {
			return fmt.Errorf("service not found: %s", serviceName)
		}

		// Build image name with registry and version
		fullImageName := fmt.Sprintf("%s/%s:%s", config.DockerRegistry, imageName, version)

		// Push Docker image
		if err := runner.DockerPush(ctx, fullImageName); err != nil {
			return fmt.Errorf("failed to push Docker image: %w", err)
		}

		ui.Success("Pushed image: %s", fullImageName)
	}

	return nil
}

// buildRuntimeImage checks if the eggybyte-go-alpine runtime image exists.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//   - useBuildx: Whether to use buildx for multi-platform builds
//   - platforms: Target platforms (comma-separated)
//   - push: Whether to push images to registry
//
// Returns:
//   - error: Check error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Docker image check
func buildRuntimeImage(ctx context.Context, runner *toolrunner.Runner, useBuildx bool, platforms string, push bool) error {
	ui.Info("Checking runtime image: eggybyte-go-alpine")

	// Check if runtime image already exists
	result, err := runner.Docker(ctx, "images", "-q", "eggybyte-go-alpine")
	if err == nil && result.ExitCode == 0 && strings.TrimSpace(result.Stdout) != "" {
		ui.Debug("Runtime image already exists")
		ui.Success("Runtime image available: eggybyte-go-alpine")
		return nil
	}

	// If image doesn't exist, inform user to pull it
	ui.Info("Runtime image not found locally")
	ui.Info("Please pull the runtime image manually:")
	ui.Info("  docker pull ghcr.io/eggybyte-technology/eggybyte-go-alpine:latest")
	ui.Info("Or use the pre-built image in your deployments")

	return nil
}
