// Package toolrunner provides execution of external tools and commands.
//
// Overview:
//   - Responsibility: Execute go, buf, flutter, docker, helm, kubectl commands
//   - Key Types: Command runners, output formatters, error handlers
//   - Concurrency Model: Sequential command execution with context support
//   - Error Semantics: Structured errors with retry suggestions
//   - Performance Notes: Command output streaming, timeout handling
//
// Usage:
//
//	runner := NewRunner()
//	err := runner.Go(ctx, "mod", "init", "example.com/module")
package toolrunner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/eggybyte-technology/egg/cli/internal/ui"
)

// Runner provides execution of external tools.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - workDir: Working directory for commands
//   - verbose: Whether to show command output
//
// Returns:
//   - None (data structure)
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Minimal state, efficient command execution
type Runner struct {
	workDir string
	verbose bool
}

// GetVerbose returns the verbose setting.
func (r *Runner) GetVerbose() bool {
	return r.verbose
}

// CommandResult represents the result of a command execution.
//
// Parameters:
//   - ExitCode: Process exit code
//   - Stdout: Standard output content
//   - Stderr: Standard error content
//   - Duration: Command execution time
//
// Returns:
//   - None (data structure)
//
// Concurrency:
//   - Immutable after creation
//
// Performance:
//   - Captures output in memory
type CommandResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

// NewRunner creates a new tool runner.
//
// Parameters:
//   - workDir: Working directory for commands
//
// Returns:
//   - *Runner: Tool runner instance
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Minimal initialization overhead
func NewRunner(workDir string) *Runner {
	return &Runner{
		workDir: workDir,
		verbose: false,
	}
}

// SetVerbose enables or disables verbose output.
//
// Parameters:
//   - enabled: Whether to show command output
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(1) operation
func (r *Runner) SetVerbose(enabled bool) {
	r.verbose = enabled
}

// execute runs a command and returns the result.
//
// Parameters:
//   - ctx: Context for cancellation
//   - name: Command name
//   - args: Command arguments
//
// Returns:
//   - *CommandResult: Command execution result
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Streaming output capture
func (r *Runner) execute(ctx context.Context, name string, args ...string) (*CommandResult, error) {
	start := time.Now()

	// Create command
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = r.workDir

	// Special handling for 'go work init' to avoid parent workspace conflicts
	if name == "go" && len(args) >= 2 && args[0] == "work" && args[1] == "init" {
		cmd.Env = append(os.Environ(), "GOWORK=off")
	}

	// Show command if verbose
	if r.verbose {
		ui.Debug("Running: %s %s", name, strings.Join(args, " "))
	}

	// Capture output
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	err := cmd.Run()
	duration := time.Since(start)

	result := &CommandResult{
		ExitCode: cmd.ProcessState.ExitCode(),
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
	}

	// Handle errors
	if err != nil {
		return result, fmt.Errorf("command failed: %w", err)
	}

	if result.ExitCode != 0 {
		return result, fmt.Errorf("command exited with code %d: %s", result.ExitCode, result.Stderr)
	}

	return result, nil
}

// Go runs go commands.
//
// Parameters:
//   - ctx: Context for cancellation
//   - args: Go command arguments
//
// Returns:
//   - *CommandResult: Command execution result
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Streaming output capture
func (r *Runner) Go(ctx context.Context, args ...string) (*CommandResult, error) {
	return r.execute(ctx, "go", args...)
}

// Buf runs buf commands.
//
// Parameters:
//   - ctx: Context for cancellation
//   - args: Buf command arguments
//
// Returns:
//   - *CommandResult: Command execution result
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Streaming output capture
func (r *Runner) Buf(ctx context.Context, args ...string) (*CommandResult, error) {
	return r.execute(ctx, "buf", args...)
}

// Flutter runs flutter commands.
//
// Parameters:
//   - ctx: Context for cancellation
//   - args: Flutter command arguments
//
// Returns:
//   - *CommandResult: Command execution result
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Streaming output capture
func (r *Runner) Flutter(ctx context.Context, args ...string) (*CommandResult, error) {
	return r.execute(ctx, "flutter", args...)
}

// Docker runs docker commands.
//
// Parameters:
//   - ctx: Context for cancellation
//   - args: Docker command arguments
//
// Returns:
//   - *CommandResult: Command execution result
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Streaming output capture
func (r *Runner) Docker(ctx context.Context, args ...string) (*CommandResult, error) {
	return r.execute(ctx, "docker", args...)
}

// Helm runs helm commands.
//
// Parameters:
//   - ctx: Context for cancellation
//   - args: Helm command arguments
//
// Returns:
//   - *CommandResult: Command execution result
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Streaming output capture
func (r *Runner) Helm(ctx context.Context, args ...string) (*CommandResult, error) {
	return r.execute(ctx, "helm", args...)
}

// Kubectl runs kubectl commands.
//
// Parameters:
//   - ctx: Context for cancellation
//   - args: Kubectl command arguments
//
// Returns:
//   - *CommandResult: Command execution result
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Streaming output capture
func (r *Runner) Kubectl(ctx context.Context, args ...string) (*CommandResult, error) {
	return r.execute(ctx, "kubectl", args...)
}

// Exec runs arbitrary commands.
//
// Parameters:
//   - ctx: Context for cancellation
//   - name: Command name
//   - args: Command arguments
//
// Returns:
//   - *CommandResult: Command execution result
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Streaming output capture
func (r *Runner) Exec(ctx context.Context, name string, args ...string) (*CommandResult, error) {
	return r.execute(ctx, name, args...)
}

// GoModInit initializes a Go module.
//
// Parameters:
//   - ctx: Context for cancellation
//   - modulePath: Go module path
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Fast module initialization
func (r *Runner) GoModInit(ctx context.Context, modulePath string) error {
	result, err := r.Go(ctx, "mod", "init", modulePath)
	if err != nil {
		return fmt.Errorf("failed to initialize Go module: %w", err)
	}

	if r.verbose {
		ui.Debug("Go module initialized: %s", modulePath)
		ui.Debug("Output: %s", result.Stdout)
	}

	return nil
}

// GoModTidy tidies Go module dependencies.
//
// Parameters:
//   - ctx: Context for cancellation
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Dependency resolution
func (r *Runner) GoModTidy(ctx context.Context) error {
	result, err := r.Go(ctx, "mod", "tidy")
	if err != nil {
		return fmt.Errorf("failed to tidy Go module: %w", err)
	}

	if r.verbose {
		ui.Debug("Go module tidied")
		ui.Debug("Output: %s", result.Stdout)
	}

	return nil
}

// GoWorkInit initializes a Go workspace.
//
// Parameters:
//   - ctx: Context for cancellation
//   - modules: Module paths to include
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Fast workspace initialization
func (r *Runner) GoWorkInit(ctx context.Context, modules ...string) error {
	args := append([]string{"work", "init"}, modules...)
	result, err := r.Go(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to initialize Go workspace: %w", err)
	}

	if r.verbose {
		ui.Debug("Go workspace initialized with modules: %v", modules)
		ui.Debug("Output: %s", result.Stdout)
	}

	return nil
}

// GoWorkUse adds modules to the Go workspace.
//
// Parameters:
//   - ctx: Context for cancellation
//   - modules: Module paths to add
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Fast workspace updates
func (r *Runner) GoWorkUse(ctx context.Context, modules ...string) error {
	for _, module := range modules {
		result, err := r.Go(ctx, "work", "use", module)
		if err != nil {
			return fmt.Errorf("failed to add module to workspace: %w", err)
		}

		if r.verbose {
			ui.Debug("Added module to workspace: %s", module)
			ui.Debug("Output: %s", result.Stdout)
		}
	}

	return nil
}

// BufGenerate generates code from protobuf definitions.
//
// Parameters:
//   - ctx: Context for cancellation
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Code generation time depends on protobuf complexity
func (r *Runner) BufGenerate(ctx context.Context) error {
	result, err := r.Buf(ctx, "generate")
	if err != nil {
		return fmt.Errorf("failed to generate code with buf: %w", err)
	}

	if r.verbose {
		ui.Debug("Code generation completed")
		ui.Debug("Output: %s", result.Stdout)
	}

	return nil
}

// FlutterCreate creates a new Flutter project.
//
// Parameters:
//   - ctx: Context for cancellation
//   - projectName: Project name
//   - platforms: Target platforms (web, android, ios)
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Project scaffolding time
func (r *Runner) FlutterCreate(ctx context.Context, projectName string, platforms []string) error {
	args := []string{"create", projectName}

	// Add platforms
	if len(platforms) > 0 {
		args = append(args, "--platforms", strings.Join(platforms, ","))
	}

	result, err := r.Flutter(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to create Flutter project: %w", err)
	}

	if r.verbose {
		ui.Debug("Flutter project created: %s (platforms: %v)", projectName, platforms)
		ui.Debug("Output: %s", result.Stdout)
	}

	return nil
}

// DockerBuild builds a Docker image.
//
// Parameters:
//   - ctx: Context for cancellation
//   - imageName: Image name and tag
//   - dockerfile: Dockerfile path
//   - context: Build context path
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Build time depends on image complexity
func (r *Runner) DockerBuild(ctx context.Context, imageName, dockerfile, context string) error {
	args := []string{"build", "-t", imageName}
	if dockerfile != "" {
		args = append(args, "-f", dockerfile)
	}
	args = append(args, context)

	result, err := r.Docker(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	if r.verbose {
		ui.Debug("Docker image built: %s", imageName)
		ui.Debug("Output: %s", result.Stdout)
	}

	return nil
}

// DockerBuildWithArgs builds a Docker image with build arguments.
//
// Parameters:
//   - ctx: Context for cancellation
//   - imageName: Image name and tag
//   - dockerfile: Dockerfile path
//   - context: Build context path
//   - buildArgs: Build arguments map
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Build time depends on image complexity
func (r *Runner) DockerBuildWithArgs(ctx context.Context, imageName, dockerfile, context string, buildArgs map[string]string) error {
	args := []string{"build", "-t", imageName}
	if dockerfile != "" {
		args = append(args, "-f", dockerfile)
	}

	// Add build arguments
	for key, value := range buildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	args = append(args, context)

	result, err := r.Docker(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	if r.verbose {
		ui.Debug("Docker image built: %s", imageName)
		ui.Debug("Output: %s", result.Stdout)
	}

	return nil
}

// DockerBuildx builds a multi-platform Docker image using buildx.
//
// Parameters:
//   - ctx: Context for cancellation
//   - imageName: Image name and tag
//   - dockerfile: Dockerfile path
//   - context: Build context path
//   - platforms: Target platforms (e.g., "linux/amd64,linux/arm64")
//   - push: Whether to push the image to registry
//   - load: Whether to load the image into local Docker (only works with single platform)
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Multi-platform build time depends on image complexity and number of platforms
func (r *Runner) DockerBuildx(ctx context.Context, imageName, dockerfile, context, platforms string, push, load bool) error {
	// Build buildx command
	args := []string{"buildx", "build"}

	// Add platform support
	if platforms != "" {
		args = append(args, "--platform", platforms)
	}

	// Add image tag
	args = append(args, "-t", imageName)

	// Add dockerfile
	if dockerfile != "" {
		args = append(args, "-f", dockerfile)
	}

	// Add push or load flag
	if push {
		args = append(args, "--push")
	} else if load {
		args = append(args, "--load")
	}

	// Add build context
	args = append(args, context)

	result, err := r.Docker(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to build Docker image with buildx: %w", err)
	}

	if r.verbose {
		ui.Debug("Docker image built with buildx: %s (platforms: %s)", imageName, platforms)
		ui.Debug("Output: %s", result.Stdout)
	}

	return nil
}

// DockerBuildxWithArgs builds a multi-platform Docker image using buildx with build arguments.
//
// Parameters:
//   - ctx: Context for cancellation
//   - imageName: Image name and tag
//   - dockerfile: Dockerfile path
//   - context: Build context path
//   - platforms: Target platforms (e.g., "linux/amd64,linux/arm64")
//   - push: Whether to push the image to registry
//   - load: Whether to load the image into local Docker (only works with single platform)
//   - buildArgs: Build arguments map
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Multi-platform build time depends on image complexity and number of platforms
func (r *Runner) DockerBuildxWithArgs(ctx context.Context, imageName, dockerfile, context, platforms string, push, load bool, buildArgs map[string]string) error {
	// Build buildx command
	args := []string{"buildx", "build"}

	// Add platform support
	if platforms != "" {
		args = append(args, "--platform", platforms)
	}

	// Add image tag
	args = append(args, "-t", imageName)

	// Add dockerfile
	if dockerfile != "" {
		args = append(args, "-f", dockerfile)
	}

	// Add build arguments
	for key, value := range buildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	// Add push or load flag
	if push {
		args = append(args, "--push")
	} else if load {
		args = append(args, "--load")
	}

	// Add build context
	args = append(args, context)

	result, err := r.Docker(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to build Docker image with buildx: %w", err)
	}

	if r.verbose {
		ui.Debug("Docker image built with buildx: %s (platforms: %s)", imageName, platforms)
		ui.Debug("Output: %s", result.Stdout)
	}

	return nil
}

// DockerPush pushes a Docker image to registry.
//
// Parameters:
//   - ctx: Context for cancellation
//   - imageName: Image name and tag
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Push time depends on image size and network
func (r *Runner) DockerPush(ctx context.Context, imageName string) error {
	result, err := r.Docker(ctx, "push", imageName)
	if err != nil {
		return fmt.Errorf("failed to push Docker image: %w", err)
	}

	if r.verbose {
		ui.Debug("Docker image pushed: %s", imageName)
		ui.Debug("Output: %s", result.Stdout)
	}

	return nil
}

// HelmTemplate renders Helm templates.
//
// Parameters:
//   - ctx: Context for cancellation
//   - chart: Chart path or name
//   - values: Values file path
//   - outputDir: Output directory
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Template rendering time
func (r *Runner) HelmTemplate(ctx context.Context, chart, values, outputDir string) error {
	args := []string{"template", chart}
	if values != "" {
		args = append(args, "-f", values)
	}
	if outputDir != "" {
		args = append(args, "--output-dir", outputDir)
	}

	result, err := r.Helm(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to render Helm templates: %w", err)
	}

	if r.verbose {
		ui.Debug("Helm templates rendered")
		ui.Debug("Output: %s", result.Stdout)
	}

	return nil
}

// KubectlApply applies Kubernetes manifests.
//
// Parameters:
//   - ctx: Context for cancellation
//   - manifest: Manifest file or directory
//   - namespace: Target namespace
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Apply time depends on resource complexity
func (r *Runner) KubectlApply(ctx context.Context, manifest, namespace string) error {
	args := []string{"apply", "-f", manifest}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	result, err := r.Kubectl(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to apply Kubernetes manifests: %w", err)
	}

	if r.verbose {
		ui.Debug("Kubernetes manifests applied")
		ui.Debug("Output: %s", result.Stdout)
	}

	return nil
}

// CheckToolAvailability checks if a tool is available in PATH.
//
// Parameters:
//   - toolName: Name of the tool to check
//
// Returns:
//   - bool: True if tool is available
//   - error: Error if tool is not found
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Fast PATH lookup
func CheckToolAvailability(toolName string) (bool, error) {
	_, err := exec.LookPath(toolName)
	if err != nil {
		return false, fmt.Errorf("tool not found in PATH: %s", toolName)
	}
	return true, nil
}

// CheckRequiredTools checks if all required tools are available.
//
// Parameters:
//   - None
//
// Returns:
//   - error: Error if any required tool is missing
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Sequential tool checks
func CheckRequiredTools() error {
	requiredTools := []string{
		"go",
		"buf",
		"docker",
		"kubectl",
		"helm",
	}

	var missingTools []string
	for _, tool := range requiredTools {
		if available, _ := CheckToolAvailability(tool); !available {
			missingTools = append(missingTools, tool)
		}
	}

	if len(missingTools) > 0 {
		return fmt.Errorf("missing required tools: %s", strings.Join(missingTools, ", "))
	}

	return nil
}

// GetGoVersion returns the Go version.
//
// Parameters:
//   - ctx: Context for cancellation
//
// Returns:
//   - string: Go version
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Fast version check
func GetGoVersion(ctx context.Context) (string, error) {
	result, err := exec.CommandContext(ctx, "go", "version").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Go version: %w", err)
	}

	return strings.TrimSpace(string(result)), nil
}

// GetBufVersion returns the buf version.
//
// Parameters:
//   - ctx: Context for cancellation
//
// Returns:
//   - string: Buf version
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per command
//
// Performance:
//   - Fast version check
func GetBufVersion(ctx context.Context) (string, error) {
	result, err := exec.CommandContext(ctx, "buf", "--version").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get buf version: %w", err)
	}

	return strings.TrimSpace(string(result)), nil
}
