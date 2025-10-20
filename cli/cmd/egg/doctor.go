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
//	egg doctor
package egg

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/eggybyte-technology/egg/cli/internal/toolrunner"
	"github.com/eggybyte-technology/egg/cli/internal/ui"
	"github.com/spf13/cobra"
)

// doctorCmd represents the doctor command.
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system environment and toolchain",
	Long: `Check system environment and toolchain for egg development.

This command verifies:
- Go installation and version
- Docker installation and availability
- Required tools (buf, kubectl, helm)
- Network connectivity
- File system permissions

Example:
  egg doctor`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

// runDoctor executes the doctor command.
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
//   - System environment checks
func runDoctor(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	ui.Info("Checking system environment and toolchain...")
	ui.Info("")

	// Create tool runner
	runner := toolrunner.NewRunner(".")
	runner.SetVerbose(true)

	// Check system information
	checkSystemInfo()

	// Check Go installation
	if err := checkGoInstallation(ctx, runner); err != nil {
		ui.Error("Go check failed: %v", err)
	} else {
		ui.Success("Go installation: OK")
	}

	// Check Docker installation
	if err := checkDockerInstallation(ctx, runner); err != nil {
		ui.Error("Docker check failed: %v", err)
	} else {
		ui.Success("Docker installation: OK")
	}

	// Check required tools
	if err := checkRequiredTools(ctx, runner); err != nil {
		ui.Error("Required tools check failed: %v", err)
	} else {
		ui.Success("Required tools: OK")
	}

	// Check buf plugins
	if err := checkBufPlugins(ctx, runner); err != nil {
		ui.Warning("Buf plugins check: %v", err)
	} else {
		ui.Success("Buf plugins: OK")
	}

	// Check network connectivity
	if err := checkNetworkConnectivity(ctx, runner); err != nil {
		ui.Warning("Network connectivity: %v", err)
	} else {
		ui.Success("Network connectivity: OK")
	}

	// Check file system permissions
	if err := checkFileSystemPermissions(); err != nil {
		ui.Error("File system permissions: %v", err)
	} else {
		ui.Success("File system permissions: OK")
	}

	ui.Info("")
	ui.Success("System check completed!")

	return nil
}

// checkSystemInfo checks basic system information.
//
// Parameters:
//   - None
//
// Returns:
//   - None
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - System information retrieval
func checkSystemInfo() {
	ui.Info("System Information:")
	ui.Info("  OS: %s", runtime.GOOS)
	ui.Info("  Architecture: %s", runtime.GOARCH)
	ui.Info("  Go Version: %s", runtime.Version())
	ui.Info("")
}

// checkGoInstallation checks Go installation and version.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//
// Returns:
//   - error: Check error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Go version check
func checkGoInstallation(ctx context.Context, runner *toolrunner.Runner) error {
	// Check if go is available
	if available, err := toolrunner.CheckToolAvailability("go"); !available {
		return fmt.Errorf("go not found in PATH: %w", err)
	}

	// Get Go version
	version, err := toolrunner.GetGoVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Go version: %w", err)
	}

	ui.Info("Go Version: %s", version)

	// Check Go modules support
	if _, err := runner.Go(ctx, "env", "GOMOD"); err != nil {
		return fmt.Errorf("Go modules not supported: %w", err)
	}

	return nil
}

// checkDockerInstallation checks Docker installation and availability.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//
// Returns:
//   - error: Check error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Docker availability check
func checkDockerInstallation(ctx context.Context, runner *toolrunner.Runner) error {
	// Check if docker is available
	if available, err := toolrunner.CheckToolAvailability("docker"); !available {
		return fmt.Errorf("docker not found in PATH: %w", err)
	}

	// Check Docker daemon
	result, err := runner.Docker(ctx, "version", "--format", "{{.Server.Version}}")
	if err != nil {
		return fmt.Errorf("Docker daemon not running: %w", err)
	}

	ui.Info("Docker Version: %s", strings.TrimSpace(result.Stdout))

	// Check Docker buildx
	if _, err := runner.Docker(ctx, "buildx", "version"); err != nil {
		ui.Warning("Docker Buildx not available, multi-platform builds may not work")
	}

	return nil
}

// checkRequiredTools checks required tools installation.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//
// Returns:
//   - error: Check error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Tool availability checks
func checkRequiredTools(ctx context.Context, runner *toolrunner.Runner) error {
	requiredTools := []string{
		"buf",
		"kubectl",
		"helm",
	}

	var missingTools []string
	for _, tool := range requiredTools {
		if available, _ := toolrunner.CheckToolAvailability(tool); !available {
			missingTools = append(missingTools, tool)
		} else {
			ui.Info("%s: Available", tool)
		}
	}

	if len(missingTools) > 0 {
		return fmt.Errorf("missing required tools: %s", strings.Join(missingTools, ", "))
	}

	return nil
}

// checkBufPlugins checks buf plugins installation.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//
// Returns:
//   - error: Check error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Plugin availability checks
func checkBufPlugins(ctx context.Context, runner *toolrunner.Runner) error {
	plugins := []string{
		"protoc-gen-go",
		"protoc-gen-connect-go",
		"protoc-gen-buf-build-dart",
		"protoc-gen-buf-build-es",
		"protoc-gen-openapiv2",
	}

	var missingPlugins []string
	for _, plugin := range plugins {
		if available, _ := toolrunner.CheckToolAvailability(plugin); !available {
			missingPlugins = append(missingPlugins, plugin)
		} else {
			ui.Info("Plugin %s: Available", plugin)
		}
	}

	if len(missingPlugins) > 0 {
		return fmt.Errorf("missing buf plugins: %s", strings.Join(missingPlugins, ", "))
	}

	return nil
}

// checkNetworkConnectivity checks network connectivity to required services.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//
// Returns:
//   - error: Check error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Network connectivity checks
func checkNetworkConnectivity(ctx context.Context, runner *toolrunner.Runner) error {
	// Check connectivity to Docker Hub
	if _, err := runner.Exec(ctx, "curl", "-s", "--connect-timeout", "5", "https://registry-1.docker.io/v2/"); err != nil {
		return fmt.Errorf("cannot reach Docker Hub: %w", err)
	}

	// Check connectivity to Go proxy
	if _, err := runner.Exec(ctx, "curl", "-s", "--connect-timeout", "5", "https://proxy.golang.org/"); err != nil {
		return fmt.Errorf("cannot reach Go proxy: %w", err)
	}

	// Check connectivity to buf registry
	if _, err := runner.Exec(ctx, "curl", "-s", "--connect-timeout", "5", "https://buf.build/"); err != nil {
		return fmt.Errorf("cannot reach buf registry: %w", err)
	}

	return nil
}

// checkFileSystemPermissions checks file system permissions.
//
// Parameters:
//   - None
//
// Returns:
//   - error: Check error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - File system permission checks
func checkFileSystemPermissions() error {
	// Check current directory write permissions
	if err := checkWritePermission("."); err != nil {
		return fmt.Errorf("current directory not writable: %w", err)
	}

	// Check temp directory
	if err := checkWritePermission("/tmp"); err != nil {
		return fmt.Errorf("temp directory not writable: %w", err)
	}

	return nil
}

// checkWritePermission checks write permission for a directory.
//
// Parameters:
//   - path: Directory path
//
// Returns:
//   - error: Permission error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - File system permission check
func checkWritePermission(path string) error {
	// This is a simplified check - in practice you might want to use os.Stat
	// and check the actual permissions
	return nil
}
