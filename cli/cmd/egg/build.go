// Package main provides the egg CLI build command.
//
// Overview:
//   - Responsibility: Build foundation images, backend services, and frontend applications
//   - Key Types: Build commands for foundation, backend, frontend, and all
//   - Concurrency Model: Sequential builds with parallel image builds (future)
//   - Error Semantics: Build errors with detailed context
//   - Performance Notes: Docker build caching, multi-stage builds
//
// Usage:
//
//	egg build foundation          # Build base images
//	egg build backend <service>   # Build backend service
//	egg build frontend <service>  # Build frontend service
//	egg build all                 # Build all services
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"go.eggybyte.com/egg/cli/internal/ui"
	"gopkg.in/yaml.v3"
)

// Image version constants (pinned for stability)
const (
	GoVersion     = "1.25.1"
	AlpineVersion = "3.22"
	NginxVersion  = "1.27.2"
	BuilderImage  = "ghcr.io/eggybyte-technology/eggybyte-go-builder:go" + GoVersion + "-alpine" + AlpineVersion
	RuntimeImage  = "ghcr.io/eggybyte-technology/eggybyte-go-alpine:go" + GoVersion + "-alpine" + AlpineVersion
	NginxImage    = "nginx:" + NginxVersion + "-alpine"
)

var (
	buildPush     bool // For individual backend/frontend commands
	buildLocal    bool // For build all command
	buildPlatform string
	buildTag      string
)

// buildCmd represents the build command.
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build Docker images for services",
	Long: `Build Docker images for EggyByte services.

This command provides several subcommands:
- backend:  Build a specific backend service
- frontend: Build a specific frontend service
- all:      Build all services in the project

Build Process:
1. Backend: Compile binary using builder container, then package into runtime image
2. Frontend: Build Flutter web assets locally, then package into nginx image

Note: Foundation images (builder + runtime) are built from the egg repository
using 'make docker-build-foundation' (not from generated projects).

Examples:
  egg build backend user                     # Build user service
  egg build frontend admin_portal            # Build admin portal
  egg build all                              # Build all services`,
}

// Note: buildFoundationCmd has been removed.
// Foundation images should be built from the egg repository using Makefile:
//   make docker-build-foundation
//   make docker-build-foundation PUSH=true PLATFORM=linux/amd64,linux/arm64

// buildBackendCmd represents the build backend command.
var buildBackendCmd = &cobra.Command{
	Use:   "backend [service]",
	Short: "Build backend service image(s) (multi-arch)",
	Long: `Build backend service Docker image(s) with multi-architecture support.

Build Process:
1. Compile Go binary in Docker builder container
2. Package binary into eggybyte-go-alpine runtime image
3. Build for multiple architectures (default: linux/amd64, linux/arm64)

All compilation is done inside Docker containers for consistency.

Flags:
  --push: Push image to registry after building
  --platform: Target platforms (default: linux/amd64,linux/arm64)
  --tag: Custom tag (overrides egg.yaml version)

Example:
  egg build backend user              # Build specific service
  egg build backend                   # Build all backend services
  egg build backend order --push --tag v1.0.0
  egg build backend user --platform linux/amd64`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBuildBackend,
}

// buildFrontendCmd represents the build frontend command.
var buildFrontendCmd = &cobra.Command{
	Use:   "frontend [service]",
	Short: "Build frontend service image(s)",
	Long: `Build frontend service Docker image(s).

Build Process:
1. Build Flutter web assets using local Flutter SDK
2. Package assets into nginx image

The web assets are built to bin/frontend/<service>/ before packaging.

Flags:
  --push: Push image to registry after building
  --tag: Custom tag (overrides egg.yaml version)

Example:
  egg build frontend admin_portal     # Build specific service
  egg build frontend                  # Build all frontend services
  egg build frontend user_app --push --tag v1.0.0`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBuildFrontend,
}

// buildAllCmd represents the build all command.
var buildAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Build all services in the project",
	Long: `Build all backend and frontend services in the project.

This command discovers all services in backend/ and frontend/ directories
and builds each one sequentially.

Default behavior:
  - Multi-platform build (linux/amd64,linux/arm64) with push enabled
  - Use --local to disable push and build for local platform only

Flags:
  --local: Build for local platform only (no push)
  --platform: Target platform (default: linux/amd64,linux/arm64)

Example:
  egg build all                    # Multi-platform build and push (default)
  egg build all --local            # Build for local platform only
  egg build all --platform linux/amd64`,
	RunE: runBuildAll,
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.AddCommand(buildBackendCmd)
	buildCmd.AddCommand(buildFrontendCmd)
	buildCmd.AddCommand(buildAllCmd)

	// Backend flags
	buildBackendCmd.Flags().BoolVar(&buildPush, "push", false, "Push image to registry")
	buildBackendCmd.Flags().BoolVar(&buildLocal, "local", false, "Build for local platform only (no push)")
	buildBackendCmd.Flags().StringVar(&buildPlatform, "platform", "linux/amd64,linux/arm64", "Target platforms (comma-separated)")
	buildBackendCmd.Flags().StringVar(&buildTag, "tag", "", "Custom tag (overrides egg.yaml version)")

	// Frontend flags
	buildFrontendCmd.Flags().BoolVar(&buildPush, "push", false, "Push image to registry")
	buildFrontendCmd.Flags().BoolVar(&buildLocal, "local", false, "Build for local platform only (no push)")
	buildFrontendCmd.Flags().StringVar(&buildTag, "tag", "", "Custom tag (overrides egg.yaml version)")

	// All flags
	buildAllCmd.Flags().BoolVar(&buildLocal, "local", false, "Build for local platform only (no push)")
	buildAllCmd.Flags().StringVar(&buildPlatform, "platform", "linux/amd64,linux/arm64", "Target platforms for backend (comma-separated)")
}

// Note: runBuildFoundation has been removed.
// Foundation images are built from the egg repository Makefile.

// runBuildBackend builds backend service image(s).
func runBuildBackend(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load project configuration
	config, err := loadProjectConfig()
	if err != nil {
		return fmt.Errorf("failed to load project config: %w", err)
	}

	// Determine which services to build
	var servicesToBuild []string
	if len(args) > 0 && args[0] != "" {
		// Build specific service
		serviceName := args[0]
		serviceDir := filepath.Join("backend", serviceName)
		if _, err := os.Stat(serviceDir); os.IsNotExist(err) {
			return fmt.Errorf("service not found: %s", serviceDir)
		}
		servicesToBuild = []string{serviceName}
	} else {
		// Build all backend services
		services, err := discoverServices("backend")
		if err != nil {
			return fmt.Errorf("failed to discover backend services: %w", err)
		}
		if len(services) == 0 {
			return fmt.Errorf("no backend services found")
		}
		servicesToBuild = services
		ui.Info("Building all backend services: %v", servicesToBuild)
	}

	// Build each service
	for _, serviceName := range servicesToBuild {
		if err := buildBackendService(ctx, serviceName, config); err != nil {
			return fmt.Errorf("failed to build backend service %s: %w", serviceName, err)
		}
	}

	ui.Success("Backend service(s) built successfully")
	return nil
}

// buildBackendService builds a single backend service image.
func buildBackendService(ctx context.Context, serviceName string, config *ProjectConfig) error {
	ui.Info("Building backend service: %s", serviceName)

	// Prepare image metadata
	imageTag := buildTag
	if imageTag == "" {
		imageTag = config.Version
	}
	imageName := fmt.Sprintf("%s/%s-%s:%s", config.DockerRegistry, config.ProjectName, serviceName, imageTag)

	// Determine if multi-platform build
	isMultiPlatform := strings.Contains(buildPlatform, ",")

	// Handle --local flag
	if buildLocal {
		localPlatform := detectLocalPlatform()
		buildPlatform = localPlatform
		isMultiPlatform = false
		buildPush = false
		ui.Info("--local specified: building for local platform (%s) only", localPlatform)
	}

	// Multi-platform builds MUST use --push (buildx limitation)
	if isMultiPlatform && !buildPush {
		ui.Warning("Multi-platform builds require --push flag (buildx limitation)")
		ui.Info("Switching to single platform: linux/amd64")
		buildPlatform = "linux/amd64"
		isMultiPlatform = false
	}

	// Prepare build arguments
	buildArgs := []string{
		fmt.Sprintf("SERVICE_NAME=%s", serviceName),
		fmt.Sprintf("PROJECT_NAME=%s", config.ProjectName),
		fmt.Sprintf("MODULE_PREFIX=%s", config.ModulePrefix),
		fmt.Sprintf("VERSION=%s", config.Version),
		fmt.Sprintf("GO_VERSION=%s", GoVersion),
		fmt.Sprintf("OUT_DIR=%s", "/out"),
	}

	// Get service ports from config if available
	// Note: This would require loading full config schema, for now use defaults
	// buildArgs = append(buildArgs, fmt.Sprintf("HTTP_PORT=%d", httpPort))

	if isMultiPlatform {
		// Multi-platform build with push
		ui.Info("Building multi-platform Docker image for %s...", buildPlatform)
		if err := buildMultiPlatformImage(ctx, "docker/Dockerfile.backend", imageName, buildPlatform, buildArgs); err != nil {
			return fmt.Errorf("failed to build and push multi-platform image: %w", err)
		}
		ui.Success("Multi-platform image built and pushed: %s (platforms: %s)", imageName, buildPlatform)
	} else {
		// Single platform build
		ui.Info("Building Docker image for %s...", buildPlatform)
		if err := buildDockerImageWithArgs(ctx, "docker/Dockerfile.backend", imageName, ".", buildPlatform, buildArgs); err != nil {
			return fmt.Errorf("failed to build Docker image: %w", err)
		}
		ui.Success("Image built: %s (platform: %s)", imageName, buildPlatform)

		// Push if requested (single platform)
		if buildPush {
			ui.Info("Pushing image to registry...")
			if err := pushDockerImage(ctx, imageName); err != nil {
				return fmt.Errorf("failed to push image: %w", err)
			}
			ui.Success("Image pushed successfully")
		}
	}

	return nil
}

// runBuildFrontend builds frontend service image(s).
func runBuildFrontend(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load project configuration
	config, err := loadProjectConfig()
	if err != nil {
		return fmt.Errorf("failed to load project config: %w", err)
	}

	// Determine which services to build
	var servicesToBuild []string
	if len(args) > 0 && args[0] != "" {
		// Build specific service
		serviceName := args[0]
		serviceDir := filepath.Join("frontend", serviceName)
		if _, err := os.Stat(serviceDir); os.IsNotExist(err) {
			return fmt.Errorf("service not found: %s", serviceDir)
		}
		servicesToBuild = []string{serviceName}
	} else {
		// Build all frontend services
		services, err := discoverServices("frontend")
		if err != nil {
			return fmt.Errorf("failed to discover frontend services: %w", err)
		}
		if len(services) == 0 {
			return fmt.Errorf("no frontend services found")
		}
		servicesToBuild = services
		ui.Info("Building all frontend services: %v", servicesToBuild)
	}

	// Build each service
	for _, serviceName := range servicesToBuild {
		if err := buildFrontendService(ctx, serviceName, config); err != nil {
			return fmt.Errorf("failed to build frontend service %s: %w", serviceName, err)
		}
	}

	ui.Success("Frontend service(s) built successfully")
	return nil
}

// buildFrontendService builds a single frontend service image.
func buildFrontendService(ctx context.Context, serviceName string, config *ProjectConfig) error {
	ui.Info("Building frontend service: %s", serviceName)

	// Step 1: Build Flutter web assets
	ui.Info("Building Flutter web assets...")
	serviceDir := filepath.Join("frontend", serviceName)
	flutterBuildDir := filepath.Join(serviceDir, "build", "web")

	if err := buildFlutterWeb(ctx, serviceName, serviceDir); err != nil {
		return fmt.Errorf("failed to build Flutter web: %w", err)
	}
	ui.Success("Flutter web built: %s", flutterBuildDir)

	// Step 2: Build Docker image with multi-platform support
	imageTag := buildTag
	if imageTag == "" {
		imageTag = config.Version
	}
	// Convert underscores to hyphens for Docker image names (Docker naming convention)
	dockerServiceName := strings.ReplaceAll(serviceName, "_", "-")
	imageName := fmt.Sprintf("%s/%s-%s-frontend:%s", config.DockerRegistry, config.ProjectName, dockerServiceName, imageTag)

	buildArgs := []string{
		fmt.Sprintf("SERVICE_NAME=%s", serviceName),
		fmt.Sprintf("WEB_DIR=%s", flutterBuildDir),
		fmt.Sprintf("PROJECT_NAME=%s", config.ProjectName),
		fmt.Sprintf("MODULE_PREFIX=%s", config.ModulePrefix),
		fmt.Sprintf("VERSION=%s", config.Version),
	}

	// Frontend images are platform-agnostic (static files), but we support multi-platform for consistency
	isMultiPlatform := strings.Contains(buildPlatform, ",")

	// Handle --local flag
	if buildLocal {
		localPlatform := detectLocalPlatform()
		buildPlatform = localPlatform
		isMultiPlatform = false
		buildPush = false
		ui.Info("--local specified: building for local platform (%s) only", localPlatform)
	}

	// Multi-platform builds MUST use --push (buildx limitation)
	if isMultiPlatform && !buildPush {
		ui.Warning("Multi-platform builds require --push flag (buildx limitation)")
		ui.Info("Building for single platform instead: linux/amd64")
		buildPlatform = "linux/amd64"
		isMultiPlatform = false
	}

	if isMultiPlatform {
		// Multi-platform build with push
		ui.Info("Building multi-platform Docker image for %s...", buildPlatform)
		if err := buildMultiPlatformImage(ctx, "docker/Dockerfile.frontend", imageName, buildPlatform, buildArgs); err != nil {
			return fmt.Errorf("failed to build and push multi-platform image: %w", err)
		}
		ui.Success("Multi-platform image built and pushed: %s (platforms: %s)", imageName, buildPlatform)
	} else {
		// Single platform build
		ui.Info("Building Docker image for %s...", buildPlatform)
		if err := buildDockerImageWithArgs(ctx, "docker/Dockerfile.frontend", imageName, ".", buildPlatform, buildArgs); err != nil {
			return fmt.Errorf("failed to build Docker image: %w", err)
		}
		ui.Success("Image built: %s", imageName)

		// Push if requested (single platform)
		if buildPush {
			ui.Info("Pushing image to registry...")
			if err := pushDockerImage(ctx, imageName); err != nil {
				return fmt.Errorf("failed to push image: %w", err)
			}
			ui.Success("Image pushed successfully")
		}
	}

	return nil
}

// runBuildAll builds all services in the project.
func runBuildAll(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	ui.Info("Building all services...")

	// Load project configuration
	config, err := loadProjectConfig()
	if err != nil {
		return fmt.Errorf("failed to load project config: %w", err)
	}

	// Handle --local flag: default is push for multi-platform, --local disables push
	isMultiPlatform := strings.Contains(buildPlatform, ",")
	var shouldPush bool

	if buildLocal {
		// Detect local platform and use single platform
		localPlatform := detectLocalPlatform()
		buildPlatform = localPlatform
		isMultiPlatform = false
		shouldPush = false
		ui.Info("--local specified: building for local platform (%s) only", localPlatform)
	} else if isMultiPlatform {
		ui.Info("Multi-platform build: images will be pushed to registry")
		shouldPush = true
	} else {
		// Single platform, no push by default
		shouldPush = false
	}

	// Discover backend services
	backendServices, err := discoverServices("backend")
	if err != nil {
		return fmt.Errorf("failed to discover backend services: %w", err)
	}

	// Discover frontend services
	frontendServices, err := discoverServices("frontend")
	if err != nil {
		return fmt.Errorf("failed to discover frontend services: %w", err)
	}

	ui.Info("Found %d backend services, %d frontend services", len(backendServices), len(frontendServices))

	// Store original push setting for individual commands
	originalPush := buildPush
	originalPlatform := buildPlatform

	// Build backend services
	for _, service := range backendServices {
		ui.Info("Building backend service: %s", service)
		// Set push and platform for this build
		buildPush = shouldPush
		buildPlatform = originalPlatform
		if err := buildBackendService(ctx, service, config); err != nil {
			return fmt.Errorf("failed to build backend service %s: %w", service, err)
		}
	}

	// Build frontend services
	for _, service := range frontendServices {
		ui.Info("Building frontend service: %s", service)
		buildPush = shouldPush
		buildPlatform = originalPlatform
		if err := buildFrontendService(ctx, service, config); err != nil {
			return fmt.Errorf("failed to build frontend service %s: %w", service, err)
		}
	}

	// Restore original settings
	buildPush = originalPush
	buildPlatform = originalPlatform

	ui.Success("All services built successfully")
	return nil
}

// Helper functions

// ProjectConfig represents the minimal egg.yaml configuration needed for builds.
type ProjectConfig struct {
	ProjectName    string `yaml:"project_name"`
	Version        string `yaml:"version"`
	ModulePrefix   string `yaml:"module_prefix"`
	DockerRegistry string `yaml:"docker_registry"`
}

// loadProjectConfig loads the project configuration from egg.yaml.
func loadProjectConfig() (*ProjectConfig, error) {
	data, err := os.ReadFile("egg.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read egg.yaml: %w (are you in the project root?)", err)
	}

	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse egg.yaml: %w", err)
	}

	// Validate required fields
	if config.ProjectName == "" {
		return nil, fmt.Errorf("project_name is required in egg.yaml")
	}
	if config.DockerRegistry == "" {
		return nil, fmt.Errorf("docker_registry is required in egg.yaml")
	}
	if config.Version == "" {
		config.Version = "latest"
	}

	return &config, nil
}

// compileBinaryInContainer compiles a Go binary using the builder container.
//
// Deprecated: This function is no longer used. All compilation is now done
// inside Docker containers via the Dockerfile builder stage. This function
// is kept for potential rollback purposes.
//
// Parameters:
//   - ctx: Context for cancellation
//   - serviceName: Name of the service to compile
//   - binaryPath: Output path for the compiled binary
//
// Returns:
//   - error: Compilation error if any
func compileBinaryInContainer(ctx context.Context, serviceName, binaryPath string) error {
	// Create output directory
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get absolute path for volume mount
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Build command in container
	// Note: Using -ldflags="-s -w" to strip debug info for smaller binaries
	buildCmd := fmt.Sprintf("cd /src/backend/%s && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /src/%s ./cmd/server", serviceName, binaryPath)

	// Run builder container
	dockerArgs := []string{
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/src", cwd),
		"-w", "/src",
		BuilderImage,
		"-c", buildCmd,
	}

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	ui.Debug("Running: docker %s", strings.Join(dockerArgs, " "))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	// Verify binary was created
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary was not created at %s", binaryPath)
	}

	return nil
}

// buildFlutterWeb builds Flutter web assets in the service directory.
//
// Parameters:
//   - ctx: Context for cancellation
//   - serviceName: Name of the frontend service
//   - serviceDir: Service directory path (frontend/<serviceName>)
//
// Returns:
//   - error: Build error if any
//
// Note:
//   - Flutter build outputs to <serviceDir>/build/web/
//   - This function does NOT copy files elsewhere - Dockerfile copies directly from build/web
func buildFlutterWeb(ctx context.Context, serviceName, serviceDir string) error {
	// Check if flutter is available
	if _, err := exec.LookPath("flutter"); err != nil {
		return fmt.Errorf("flutter not found in PATH - please install Flutter SDK")
	}

	// Verify service directory exists
	if _, err := os.Stat(serviceDir); os.IsNotExist(err) {
		return fmt.Errorf("frontend service directory not found: %s", serviceDir)
	}

	// Run flutter build web (outputs to build/web by default)
	ui.Debug("Building Flutter web in %s", serviceDir)

	cmd := exec.CommandContext(ctx, "flutter", "build", "web", "--release")
	cmd.Dir = serviceDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("flutter build failed: %w", err)
	}

	// Flutter outputs to build/web directory (relative to service dir)
	flutterBuildDir := filepath.Join(serviceDir, "build", "web")

	// Verify build output exists
	indexPath := filepath.Join(flutterBuildDir, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return fmt.Errorf("flutter build did not create index.html at %s", indexPath)
	}

	return nil
}

// buildDockerImage builds a Docker image with buildx for multi-arch support.
func buildDockerImage(ctx context.Context, dockerfile, tag, contextPath, platform string) error {
	// Use docker buildx for multi-arch builds
	args := []string{"buildx", "build", "-f", dockerfile, "-t", tag}

	if platform != "" {
		args = append(args, "--platform", platform)
	}

	// Load image into local docker daemon for single platform builds
	// For multi-platform, use --push instead
	if !strings.Contains(platform, ",") {
		args = append(args, "--load")
	}

	args = append(args, contextPath)

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	ui.Debug("Running: docker %s", strings.Join(args, " "))

	return cmd.Run()
}

// buildDockerImageWithArgs builds a Docker image with build args using buildx.
//
// Parameters:
//   - ctx: Context for cancellation
//   - dockerfile: Path to Dockerfile
//   - tag: Image tag
//   - contextPath: Build context path
//   - platform: Target platform (e.g., "linux/amd64" or "linux/arm64")
//   - buildArgs: Build arguments as key=value strings
//
// Returns:
//   - error: Build error if any
//
// Note:
//   - For single platform builds, uses --load to load image into local Docker daemon
//   - Platform parameter ensures correct base image architecture is pulled
func buildDockerImageWithArgs(ctx context.Context, dockerfile, tag, contextPath, platform string, buildArgs []string) error {
	// Use docker buildx for consistent multi-arch support
	args := []string{"buildx", "build", "-f", dockerfile, "-t", tag}

	// Always specify platform to ensure correct base image architecture is pulled
	if platform != "" {
		args = append(args, "--platform", platform)
		// Use --pull to force Docker to pull the correct platform variant of base images
		// This prevents InvalidBaseImagePlatform warnings when building for different architectures
		args = append(args, "--pull")
	}

	// Load image into local docker daemon for single platform builds
	// For multi-platform, use --push instead (handled by buildMultiPlatformImage)
	if platform != "" && !strings.Contains(platform, ",") {
		args = append(args, "--load")
	}

	// Add build arguments
	for _, arg := range buildArgs {
		args = append(args, "--build-arg", arg)
	}

	args = append(args, contextPath)

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	ui.Debug("Running: docker %s", strings.Join(args, " "))

	return cmd.Run()
}

// buildMultiPlatformImage builds a multi-platform Docker image using buildx and pushes it.
// Multi-platform builds MUST push to registry (cannot load to local docker daemon).
//
// Parameters:
//   - ctx: Context for cancellation
//   - dockerfile: Path to Dockerfile
//   - tag: Image tag
//   - platforms: Comma-separated list of platforms (e.g., "linux/amd64,linux/arm64")
//   - buildArgs: Build arguments
//
// Returns:
//   - error: Build error if any
func buildMultiPlatformImage(ctx context.Context, dockerfile, tag, platforms string, buildArgs []string) error {
	// Verify buildx is available
	if err := exec.CommandContext(ctx, "docker", "buildx", "version").Run(); err != nil {
		return fmt.Errorf("docker buildx not available - please ensure Docker Buildx is installed")
	}

	// Build multi-platform image with push
	// Note: --load cannot be used with multi-platform, must use --push
	args := []string{
		"buildx", "build",
		"-f", dockerfile,
		"-t", tag,
		"--platform", platforms,
		"--pull", // Force pull correct platform variants of base images
		"--push", // Required for multi-platform builds
	}

	for _, arg := range buildArgs {
		args = append(args, "--build-arg", arg)
	}

	args = append(args, ".") // context

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	ui.Debug("Running: docker %s", strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("buildx multi-platform build failed: %w", err)
	}

	return nil
}

// pushDockerImage pushes a Docker image to registry.
func pushDockerImage(ctx context.Context, tag string) error {
	cmd := exec.CommandContext(ctx, "docker", "push", tag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	ui.Debug("Running: docker push %s", tag)

	return cmd.Run()
}

// discoverServices discovers services in a directory.
func discoverServices(dir string) ([]string, error) {
	var services []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return services, nil // Directory doesn't exist, return empty list
		}
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			services = append(services, entry.Name())
		}
	}

	return services, nil
}

// detectLocalPlatform detects the local platform architecture.
func detectLocalPlatform() string {
	// Try to detect from runtime
	if runtime.GOARCH == "arm64" {
		return "linux/arm64"
	}
	// Default to amd64
	return "linux/amd64"
}
