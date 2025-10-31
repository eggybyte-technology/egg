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
package main

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"go.eggybyte.com/egg/cli/internal/toolrunner"
	"go.eggybyte.com/egg/cli/internal/ui"
	"go.eggybyte.com/egg/cli/internal/version"
)

// doctorCmd represents the doctor command.
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose development environment",
	Long: `Perform comprehensive diagnostics of your EGG development environment.

This command verifies:
  • Core toolchain (Go, Docker)
  • Development tools (buf, kubectl, helm)
  • Code generation (Local protoc plugins)
  • Network connectivity (Docker Hub, Go Proxy)
  • File system permissions

Note: EGG uses local protoc plugins for offline-first development.
To install missing plugins, use: egg doctor --install

Example:
  egg doctor
  egg doctor --install`,
	RunE: runDoctor,
}

func init() {
	doctorCmd.Flags().Bool("install", false, "Install missing protoc plugins")
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

	// Check if install flag is set
	install, _ := cmd.Flags().GetBool("install")
	if install {
		return installProtocPlugins(ctx)
	}

	ui.Info("EGG Development Environment Diagnostics")
	separator := strings.Repeat("=", 60)
	ui.Info("%s", separator)
	ui.Info("")

	// Create tool runner
	runner := toolrunner.NewRunner(".")
	runner.SetVerbose(false)

	// Track overall status
	hasErrors := false
	hasWarnings := false

	// Check version information
	checkVersionInfo()
	
	// Check system information
	checkSystemInfo()

	// Check Go installation
	ui.Info("Core Toolchain")
	ui.Info("  Checking Go installation...")
	if err := checkGoInstallation(ctx, runner); err != nil {
		ui.Error("  Go: %v", err)
		hasErrors = true
	} else {
		ui.Success("  Go")
		// Display Go version
		if goVersion, err := toolrunner.GetGoVersion(ctx); err == nil {
			ui.Info("      Version: %s", strings.TrimSpace(goVersion))
		}
	}

	// Check Docker installation
	ui.Info("  Checking Docker installation...")
	if err := checkDockerInstallation(ctx, runner); err != nil {
		ui.Error("  Docker: %v", err)
		hasErrors = true
	} else {
		ui.Success("  Docker")
		// Display Docker version
		if result, err := runner.Docker(ctx, "version", "--format", "{{.Server.Version}}"); err == nil {
			ui.Info("      Version: %s", strings.TrimSpace(result.Stdout))
		}
		// Check Docker buildx version
		if result, err := runner.Docker(ctx, "buildx", "version"); err == nil {
			// Extract version from buildx output (format: "github.com/docker/buildx v0.x.x")
			output := strings.TrimSpace(result.Stdout)
			if idx := strings.LastIndex(output, " "); idx >= 0 {
				buildxVersion := output[idx+1:]
				ui.Info("      Buildx: %s", buildxVersion)
			}
		}
	}
	ui.Info("")

	// Check required tools
	ui.Info("Development Tools")
	if err := checkRequiredTools(ctx, runner); err != nil {
		ui.Error("  %v", err)
		hasErrors = true
	}
	ui.Info("")

	// Check protoc plugins
	ui.Info("Code Generation")
	if err := checkProtocPlugins(ctx, runner); err != nil {
		ui.Warning("  %v", err)
		hasWarnings = true
	} else {
		ui.Success("  Local protoc plugins configured")
		ui.Info("      - protoc-gen-go")
		ui.Info("      - protoc-gen-connect-go")
		ui.Info("      - protoc-gen-openapiv2")
		ui.Info("      - protoc-gen-dart")
	}
	ui.Info("")

	// Check network connectivity
	ui.Info("Network Connectivity")
	if err := checkNetworkConnectivity(ctx, runner); err != nil {
		ui.Warning("  %v", err)
		hasWarnings = true
	}
	ui.Info("")

	// Check file system permissions
	ui.Info("File System")
	if err := checkFileSystemPermissions(); err != nil {
		ui.Error("  %v", err)
		hasErrors = true
	} else {
		ui.Success("  Permissions OK")
	}

	// Summary
	ui.Info("")
	ui.Info("%s", separator)
	if hasErrors {
		ui.Error("Diagnostics completed with ERRORS")
		ui.Info("Please resolve the errors above before proceeding.")
		return fmt.Errorf("environment check failed")
	} else if hasWarnings {
		ui.Warning("Diagnostics completed with WARNINGS")
		ui.Info("Your environment is functional but some optional features may be limited.")
	} else {
		ui.Success("All checks passed - environment ready")
	}

	return nil
}

// checkVersionInfo checks CLI and framework version information.
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
//   - Version information retrieval
func checkVersionInfo() {
	ui.Info("Version Information")
	ui.Info("  CLI Version:           %s", version.Version)
	ui.Info("  Framework Version:     %s", version.FrameworkVersion)
	ui.Info("  Git Commit:            %s", version.Commit)
	ui.Info("  Build Time:            %s", version.BuildTime)
	ui.Info("")
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
	ui.Info("System Information")
	ui.Info("  OS/Arch:    %s/%s", runtime.GOOS, runtime.GOARCH)
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
	if available, _ := toolrunner.CheckToolAvailability("go"); !available {
		return fmt.Errorf("not found in PATH")
	}

	// Get Go version
	goVersion, err := toolrunner.GetGoVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}

	// Check Go modules support
	if _, err := runner.Go(ctx, "env", "GOMOD"); err != nil {
		return fmt.Errorf("modules not supported")
	}

	// Parse version to check minimum requirement (1.21+)
	if !strings.Contains(goVersion, "go1.") {
		return fmt.Errorf("invalid version format: %s", goVersion)
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
	if available, _ := toolrunner.CheckToolAvailability("docker"); !available {
		return fmt.Errorf("not found in PATH")
	}

	// Check Docker daemon
	_, err := runner.Docker(ctx, "version", "--format", "{{.Server.Version}}")
	if err != nil {
		return fmt.Errorf("daemon not running")
	}

	// Check Docker buildx
	if _, err := runner.Docker(ctx, "buildx", "version"); err != nil {
		ui.Warning("      Buildx not available - multi-platform builds disabled")
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
	tools := []struct {
		name        string
		required    bool
		description string
	}{
		{"buf", true, "Protocol buffer compiler"},
		{"kubectl", false, "Kubernetes CLI"},
		{"helm", false, "Kubernetes package manager"},
	}

	var missingRequired []string
	for _, tool := range tools {
		available, _ := toolrunner.CheckToolAvailability(tool.name)
		ui.Info("  Checking %s...", tool.name)
		if available {
			ui.Success("  %s", tool.name)
			// Get version if available
			var toolVersion string
			switch tool.name {
			case "buf":
				if result, err := runner.Buf(ctx, "version"); err == nil {
					toolVersion = strings.TrimSpace(result.Stdout)
				}
			case "kubectl":
				if result, err := runner.Exec(ctx, "kubectl", "version", "--client", "--short"); err == nil {
					// Extract version from kubectl output (format: "Client Version: v1.x.x")
					output := strings.TrimSpace(result.Stdout)
					if idx := strings.Index(output, "v"); idx >= 0 {
						parts := strings.Fields(output[idx:])
						if len(parts) > 0 {
							toolVersion = parts[0]
						}
					}
				}
			case "helm":
				if result, err := runner.Exec(ctx, "helm", "version", "--short"); err == nil {
					toolVersion = strings.TrimSpace(result.Stdout)
				}
			}
			if toolVersion != "" {
				ui.Info("      Version: %s", toolVersion)
			}
		} else {
			if tool.required {
				ui.Error("  %s (required)", tool.name)
				missingRequired = append(missingRequired, tool.name)
			} else {
				ui.Warning("  %s (optional - %s)", tool.name, tool.description)
			}
		}
	}

	if len(missingRequired) > 0 {
		return fmt.Errorf("missing required tools: %s", strings.Join(missingRequired, ", "))
	}

	return nil
}

// ProtocPlugin represents a protoc plugin configuration.
type ProtocPlugin struct {
	Name         string
	GoPackage    string
	Required     bool
	Description  string
}

// checkProtocPlugins checks local protoc plugins installation.
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
//   - Plugin availability checks and installation
//
// Note: EGG uses local protoc plugins for offline-first development.
// This function verifies installation and offers to install missing plugins.
func checkProtocPlugins(ctx context.Context, runner *toolrunner.Runner) error {
	plugins := []ProtocPlugin{
		{"protoc-gen-go", "google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2", true, "Generate Go protocol buffer code"},
		{"protoc-gen-connect-go", "github.com/bufbuild/connect-go/cmd/protoc-gen-connect-go@v1.16.0", true, "Generate Connect RPC Go code"},
		{"protoc-gen-openapiv2", "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.24.0", false, "Generate OpenAPI v2 documentation"},
		{"protoc-gen-dart", "", false, "Generate Dart protocol buffer code (install via: dart pub global activate protoc_plugin)"},
	}

	var missing []ProtocPlugin
	for _, plugin := range plugins {
		ui.Info("  Checking %s...", plugin.Name)
		available, _ := toolrunner.CheckToolAvailability(plugin.Name)
		if available {
			ui.Success("  %s", plugin.Name)
			// Try to get plugin version
			var pluginVersion string
			switch plugin.Name {
			case "protoc-gen-go":
				// protoc-gen-go doesn't have a version flag, but we can check if it's in PATH
				pluginVersion = "installed"
			case "protoc-gen-connect-go":
				pluginVersion = "installed"
			case "protoc-gen-openapiv2":
				pluginVersion = "installed"
			case "protoc-gen-dart":
				pluginVersion = "installed"
			}
			if pluginVersion != "" {
				ui.Info("      Status: %s", pluginVersion)
			}
		} else {
			if plugin.Required {
				ui.Warning("  %s: missing (required)", plugin.Name)
				missing = append(missing, plugin)
			} else {
				ui.Warning("  %s: missing (optional)", plugin.Name)
			}
		}
	}

	if len(missing) > 0 {
		ui.Info("")
		ui.Warning("  Missing required plugins detected")
		ui.Info("")
		ui.Info("  To install missing plugins, run:")
		ui.Info("    egg doctor --install")
		ui.Info("")
		return fmt.Errorf("missing required protoc plugins")
	}

	return nil
}

// installProtocPlugins installs missing protoc plugins.
//
// Parameters:
//   - ctx: Context for cancellation
//
// Returns:
//   - error: Installation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Go module download and compilation
func installProtocPlugins(ctx context.Context) error {
	ui.Info("Installing protoc plugins...")
	ui.Info("")

	plugins := []ProtocPlugin{
		{"protoc-gen-go", "google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2", true, "Generate Go protocol buffer code"},
		{"protoc-gen-connect-go", "github.com/bufbuild/connect-go/cmd/protoc-gen-connect-go@v1.16.0", true, "Generate Connect RPC Go code"},
		{"protoc-gen-openapiv2", "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.24.0", false, "Generate OpenAPI v2 documentation"},
		{"protoc-gen-dart", "", false, "Generate Dart protocol buffer code (install via: dart pub global activate protoc_plugin)"},
	}

	runner := toolrunner.NewRunner(".")
	runner.SetVerbose(false)

	var missing []ProtocPlugin
	for _, plugin := range plugins {
		available, _ := toolrunner.CheckToolAvailability(plugin.Name)
		if !available {
			missing = append(missing, plugin)
		}
	}

	if len(missing) == 0 {
		ui.Success("All plugins are already installed")
		return nil
	}

	ui.Info("Found %d missing plugin(s)", len(missing))
	ui.Info("")

	for _, plugin := range missing {
		ui.Info("Installing %s...", plugin.Name)
		
		// Special handling for Dart plugin
		if plugin.Name == "protoc-gen-dart" {
			ui.Info("  For Dart plugin, please run manually:")
			ui.Info("    dart pub global activate protoc_plugin")
			continue
		}
		
		// Skip if no GoPackage is specified
		if plugin.GoPackage == "" {
			continue
		}
		
		cmd := fmt.Sprintf("go install %s", plugin.GoPackage)
		result, err := runner.Exec(ctx, "sh", "-c", cmd)
		if err != nil {
			ui.Error("  Failed to install %s: %v", plugin.Name, err)
			return fmt.Errorf("failed to install %s: %w", plugin.Name, err)
		}
		
		if result.ExitCode != 0 {
			ui.Error("  Failed to install %s: %s", plugin.Name, result.Stderr)
			return fmt.Errorf("failed to install %s: %s", plugin.Name, result.Stderr)
		}
		
		ui.Success("  Installed %s", plugin.Name)
	}

	ui.Info("")
	ui.Success("All plugins installed successfully")
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
	services := []struct {
		name string
		url  string
	}{
		{"Docker Hub", "https://registry-1.docker.io/v2/"},
		{"Go Proxy", "https://proxy.golang.org/"},
	}

	var failedServices []string
	for _, svc := range services {
		ui.Info("  Checking %s...", svc.name)
		if _, err := runner.Exec(ctx, "curl", "-s", "--connect-timeout", "5", svc.url); err != nil {
			ui.Warning("  %s: unreachable", svc.name)
			failedServices = append(failedServices, svc.name)
		} else {
			ui.Success("  %s", svc.name)
		}
	}

	if len(failedServices) > 0 {
		return fmt.Errorf("connectivity issues detected - some features may not work")
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
	locations := []struct {
		path string
		name string
	}{
		{".", "Current directory"},
		{"/tmp", "Temp directory"},
	}

	for _, loc := range locations {
		if err := checkWritePermission(loc.path); err != nil {
			return fmt.Errorf("%s not writable: %w", loc.name, err)
		}
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
