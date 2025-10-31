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
//	egg compose up [--detached]
//	egg compose down
//	egg compose logs [--service <name>]
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"go.eggybyte.com/egg/cli/internal/configschema"
	"go.eggybyte.com/egg/cli/internal/generators"
	"go.eggybyte.com/egg/cli/internal/portproxy"
	"go.eggybyte.com/egg/cli/internal/projectfs"
	"go.eggybyte.com/egg/cli/internal/ref"
	"go.eggybyte.com/egg/cli/internal/render/compose"
	"go.eggybyte.com/egg/cli/internal/toolrunner"
	"go.eggybyte.com/egg/cli/internal/ui"
)

// composeCmd represents the compose command.
var composeCmd = &cobra.Command{
	Use:   "compose",
	Short: "Manage Docker Compose services",
	Long: `Manage Docker Compose services for local development.

This command provides:
- Service orchestration with Docker Compose
- Database integration (MySQL)
- Environment variable management
- Log aggregation and monitoring

Examples:
  egg compose up
  egg compose down
  egg compose logs --service user-service`,
}

// composeUpCmd represents the compose up command.
var composeUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Start services",
	Long: `Start all services with Docker Compose.

This command:
- Renders compose.yaml from egg configuration to deploy/compose/compose.yaml
- Starts all backend and frontend services in detached mode (-d)
- Attaches MySQL database if enabled
- Sets up service dependencies and network

Example:
  egg compose up`,
	RunE: runComposeUp,
}

// composeDownCmd represents the compose down command.
var composeDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop services",
	Long: `Stop all services and clean up.

This command:
- Stops all running services
- Removes containers and networks
- Cleans up volumes (optional)

Example:
  egg compose down`,
	RunE: runComposeDown,
}

// composeLogsCmd represents the compose logs command.
var composeLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show service logs",
	Long: `Show logs from services.

This command:
- Displays logs from all services
- Filters logs by service name
- Follows log output in real-time

Example:
  egg compose logs
  egg compose logs --service user-service`,
	RunE: runComposeLogs,
}

// composeGenerateCmd represents the compose generate command.
var composeGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate Docker Compose configuration",
	Long: `Generate Docker Compose configuration from egg.yaml.

This command:
- Creates docker-compose.yaml with all services
- Configures database connections
- Sets up service dependencies
- Uses local eggybyte-go-alpine base image

Example:
  egg compose generate`,
	RunE: runComposeGenerate,
}

// composeProxyCmd represents the compose proxy command.
var composeProxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Create port proxy for a service",
	Long: `Create a port proxy to map a service port to localhost.

This command:
- Creates a socat-based port proxy container
- Automatically detects port availability
- Maps Docker Compose service port to localhost

Example:
  egg compose proxy user-service 8080
  egg compose proxy user-service 8080 --local-port 18080`,
	RunE: runComposeProxy,
}

// composeProxyAllCmd represents the compose proxy-all command.
var composeProxyAllCmd = &cobra.Command{
	Use:   "proxy-all",
	Short: "Create port proxies for all services",
	Long: `Create port proxies for all services defined in egg.yaml.

This command:
- Automatically maps all backend service ports (HTTP, Health, Metrics)
- Automatically maps all frontend service ports
- Detects port availability and finds alternatives if needed

Example:
  egg compose proxy-all`,
	RunE: runComposeProxyAll,
}

// composeProxyStopCmd represents the compose proxy-stop command.
var composeProxyStopCmd = &cobra.Command{
	Use:   "proxy-stop",
	Short: "Stop all port proxies",
	Long: `Stop all running port proxy containers for the project.

This command:
- Stops all socat proxy containers
- Cleans up port mappings

Example:
  egg compose proxy-stop`,
	RunE: runComposeProxyStop,
}

var (
	serviceFilter string
	followLogs    bool
	localPort     int
)

func init() {
	rootCmd.AddCommand(composeCmd)
	composeCmd.AddCommand(composeUpCmd)
	composeCmd.AddCommand(composeDownCmd)
	composeCmd.AddCommand(composeLogsCmd)
	composeCmd.AddCommand(composeGenerateCmd)
	composeCmd.AddCommand(composeProxyCmd)
	composeCmd.AddCommand(composeProxyAllCmd)
	composeCmd.AddCommand(composeProxyStopCmd)

	composeLogsCmd.Flags().StringVar(&serviceFilter, "service", "", "Filter logs by service name")
	composeLogsCmd.Flags().BoolVarP(&followLogs, "follow", "f", false, "Follow log output")
	composeProxyCmd.Flags().IntVar(&localPort, "local-port", 0, "Local port to map to (0 to auto-find)")
}

// getComposeNetworkName returns the actual Docker Compose network name.
// Docker Compose network naming convention: <project-name>_<network-name>
//
// Parameters:
//   - projectName: Project name from configuration
//
// Returns:
//   - string: Actual Docker Compose network name
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - O(1) string concatenation
func getComposeNetworkName(projectName string) string {
	// Docker Compose network naming: <project-name>_<network-name>
	// Network name in compose.yaml is: <project-name>-network
	// So actual network name is: <project-name>_<project-name>-network
	return fmt.Sprintf("%s_%s-network", projectName, projectName)
}

// runComposeUp executes the compose up command.
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
//   - Docker Compose rendering and service startup
func runComposeUp(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	ui.Info("Starting services with Docker Compose...")

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

	// Create project file system
	fs := projectfs.NewProjectFS(".")
	fs.SetVerbose(true)

	// Create reference parser
	refParser := ref.NewParser()

	// Create Compose renderer
	composeRenderer := compose.NewRenderer(fs, refParser)

	// Render Compose configuration
	if err := composeRenderer.Render(config); err != nil {
		return fmt.Errorf("failed to render Compose configuration: %w", err)
	}

	// Create tool runner
	runner := toolrunner.NewRunner(".")
	runner.SetVerbose(true)

	// Start services (always use detached mode)
	if err := startComposeServices(ctx, runner, config.ProjectName); err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}

	ui.Success("Services started successfully!")

	return nil
}

// runComposeDown executes the compose down command.
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
//   - Docker Compose service shutdown
func runComposeDown(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	ui.Info("Stopping services...")

	// Load configuration to get project name
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

	// Stop services
	if err := stopComposeServices(ctx, runner, config.ProjectName); err != nil {
		return fmt.Errorf("failed to stop services: %w", err)
	}

	ui.Success("Services stopped successfully!")
	return nil
}

// runComposeLogs executes the compose logs command.
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
//   - Docker Compose log retrieval
func runComposeLogs(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	ui.Info("Showing service logs...")

	// Load configuration to get project name
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

	// Show logs
	if err := showComposeLogs(ctx, runner, config.ProjectName, serviceFilter, followLogs); err != nil {
		return fmt.Errorf("failed to show logs: %w", err)
	}

	return nil
}

// startComposeServices starts Docker Compose services.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//   - projectName: Project name for Docker Compose project
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Docker Compose service startup
func startComposeServices(ctx context.Context, runner *toolrunner.Runner, projectName string) error {
	composeFile := "deploy/compose/compose.yaml"

	// Check if compose file exists
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("compose file not found: %s", composeFile)
	}

	// Build command arguments - always use detached mode
	// Use -p to specify project name so network names match exactly
	args := []string{"-f", composeFile, "-p", projectName, "up", "-d"}

	// Execute docker compose command
	result, err := runner.DockerCompose(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}

	// Always output command result
	if result.Stdout != "" {
		fmt.Print(result.Stdout)
	}
	if result.Stderr != "" {
		fmt.Fprint(os.Stderr, result.Stderr)
	}

	return nil
}

// stopComposeServices stops Docker Compose services.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//   - projectName: Project name for Docker Compose project
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Docker Compose service shutdown
func stopComposeServices(ctx context.Context, runner *toolrunner.Runner, projectName string) error {
	composeFile := "deploy/compose/compose.yaml"

	// Check if compose file exists
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("compose file not found: %s", composeFile)
	}

	// Execute docker compose command
	args := []string{"-f", composeFile, "-p", projectName, "down"}
	result, err := runner.DockerCompose(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to stop services: %w", err)
	}

	// Always output command result
	if result.Stdout != "" {
		fmt.Print(result.Stdout)
	}
	if result.Stderr != "" {
		fmt.Fprint(os.Stderr, result.Stderr)
	}

	return nil
}

// showComposeLogs shows Docker Compose service logs.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//   - projectName: Project name for Docker Compose project
//   - serviceFilter: Service name filter
//   - follow: Whether to follow logs
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Docker Compose log retrieval
func showComposeLogs(ctx context.Context, runner *toolrunner.Runner, projectName string, serviceFilter string, follow bool) error {
	composeFile := "deploy/compose/compose.yaml"

	// Check if compose file exists
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("compose file not found: %s", composeFile)
	}

	// Build command arguments
	args := []string{"-f", composeFile, "-p", projectName, "logs"}
	if follow {
		args = append(args, "-f")
	}
	if serviceFilter != "" {
		args = append(args, serviceFilter)
	}

	// Execute docker compose command
	result, err := runner.DockerCompose(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to show logs: %w", err)
	}

	// Print logs to stdout
	fmt.Print(result.Stdout)

	return nil
}

// runComposeGenerate executes the compose generate command.
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
//   - Docker Compose configuration generation
func runComposeGenerate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	ui.Info("Generating Docker Compose configuration...")

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

	// Create project file system
	fs := projectfs.NewProjectFS(".")
	fs.SetVerbose(true)

	// Create backend generator
	runner := toolrunner.NewRunner(".")
	runner.SetVerbose(true)
	backendGen := generators.NewBackendGenerator(fs, runner)

	// Generate docker-compose.yaml
	if err := backendGen.GenerateCompose(ctx, config); err != nil {
		return fmt.Errorf("failed to generate docker-compose.yaml: %w", err)
	}

	ui.Success("Docker Compose configuration generated successfully!")
	ui.Info("Next steps:")
	ui.Info("  1. Build base image: docker build -t localhost:5000/eggybyte-go-alpine:latest -f build/Dockerfile.eggybyte-go-alpine .")
	ui.Info("  2. Build backend services: go build -o server ./cmd/server (in each backend service directory)")
	ui.Info("  3. Start services: docker-compose up")

	return nil
}

// runComposeProxy executes the compose proxy command.
//
// Parameters:
//   - cmd: Cobra command
//   - args: Command arguments (service name, service port)
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Port availability check and Docker container creation
func runComposeProxy(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if len(args) < 2 {
		return fmt.Errorf("usage: egg compose proxy <service-name> <service-port> [--local-port <port>]")
	}

	serviceName := args[0]
	servicePortStr := args[1]
	servicePort, err := strconv.Atoi(servicePortStr)
	if err != nil {
		return fmt.Errorf("invalid service port: %s", servicePortStr)
	}

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

	// Verify service exists
	var serviceExists bool
	if _, exists := config.Backend[serviceName]; exists {
		serviceExists = true
	} else if _, exists := config.Frontend[serviceName]; exists {
		serviceExists = true
	}

	if !serviceExists {
		return fmt.Errorf("service '%s' not found in configuration", serviceName)
	}

	// Get network name (Docker Compose format: <project-name>_<network-name>)
	networkName := getComposeNetworkName(config.ProjectName)

	// Create tool runner
	runner := toolrunner.NewRunner(".")
	runner.SetVerbose(true)

	// Create port proxy manager
	manager := portproxy.NewManager(runner, config.ProjectName, networkName)

	// Create proxy
	ui.Info("Creating port proxy for %s:%d...", serviceName, servicePort)
	proxyInfo, err := manager.CreateProxy(ctx, serviceName, servicePort, localPort)
	if err != nil {
		return fmt.Errorf("failed to create port proxy: %w", err)
	}

	ui.Success("Port proxy created successfully!")
	ui.Info("  Service: %s:%d", serviceName, servicePort)
	ui.Info("  Local:   localhost:%d", proxyInfo.LocalPort)
	ui.Info("  Proxy:   %s", proxyInfo.ProxyName)
	ui.Info("")
	ui.Info("Access the service at: http://localhost:%d", proxyInfo.LocalPort)
	if proxyInfo.LocalPort != servicePort {
		ui.Info("(Note: Port %d was unavailable, using %d instead)", servicePort, proxyInfo.LocalPort)
	}

	return nil
}

// runComposeProxyAll executes the compose proxy-all command.
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
//   - Multiple port availability checks and Docker container creation
func runComposeProxyAll(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

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

	// Get network name (Docker Compose format: <project-name>_<network-name>)
	networkName := getComposeNetworkName(config.ProjectName)

	// Create tool runner
	runner := toolrunner.NewRunner(".")
	runner.SetVerbose(true)

	// Create port proxy manager
	manager := portproxy.NewManager(runner, config.ProjectName, networkName)

	ui.Info("Creating port proxies for all services...")

	proxiesCreated := 0
	proxiesFailed := 0

	// Create proxies for backend services
	for name, service := range config.Backend {
		ports := service.Ports
		if ports == nil {
			ports = &config.BackendDefaults.Ports
		}

		// HTTP port
		ui.Info("  Creating proxy for %s HTTP port %d...", name, ports.HTTP)
		proxyInfo, err := manager.CreateProxy(ctx, name, ports.HTTP, 0)
		if err != nil {
			ui.Error("    Failed: %v", err)
			proxiesFailed++
			continue
		}
		proxiesCreated++
		if proxyInfo.LocalPort != ports.HTTP {
			ui.Info("    Mapped to localhost:%d (port %d was unavailable)", proxyInfo.LocalPort, ports.HTTP)
		} else {
			ui.Info("    Mapped to localhost:%d", proxyInfo.LocalPort)
		}

		// Health port
		ui.Info("  Creating proxy for %s Health port %d...", name, ports.Health)
		proxyInfo, err = manager.CreateProxy(ctx, name, ports.Health, 0)
		if err != nil {
			ui.Error("    Failed: %v", err)
			proxiesFailed++
			continue
		}
		proxiesCreated++
		if proxyInfo.LocalPort != ports.Health {
			ui.Info("    Mapped to localhost:%d (port %d was unavailable)", proxyInfo.LocalPort, ports.Health)
		} else {
			ui.Info("    Mapped to localhost:%d", proxyInfo.LocalPort)
		}

		// Metrics port
		ui.Info("  Creating proxy for %s Metrics port %d...", name, ports.Metrics)
		proxyInfo, err = manager.CreateProxy(ctx, name, ports.Metrics, 0)
		if err != nil {
			ui.Error("    Failed: %v", err)
			proxiesFailed++
			continue
		}
		proxiesCreated++
		if proxyInfo.LocalPort != ports.Metrics {
			ui.Info("    Mapped to localhost:%d (port %d was unavailable)", proxyInfo.LocalPort, ports.Metrics)
		} else {
			ui.Info("    Mapped to localhost:%d", proxyInfo.LocalPort)
		}
	}

	// Create proxies for frontend services (port 3000)
	for name := range config.Frontend {
		frontendPort := 3000
		ui.Info("  Creating proxy for %s port %d...", name, frontendPort)
		proxyInfo, err := manager.CreateProxy(ctx, name, frontendPort, 0)
		if err != nil {
			ui.Error("    Failed: %v", err)
			proxiesFailed++
			continue
		}
		proxiesCreated++
		if proxyInfo.LocalPort != frontendPort {
			ui.Info("    Mapped to localhost:%d (port %d was unavailable)", proxyInfo.LocalPort, frontendPort)
		} else {
			ui.Info("    Mapped to localhost:%d", proxyInfo.LocalPort)
		}
	}

	ui.Info("")
	ui.Success("Port proxies created: %d successful, %d failed", proxiesCreated, proxiesFailed)

	// List all created proxies
	ui.Info("")
	ui.Info("Summary of port mappings:")
	proxies, err := manager.ListProxies(ctx)
	if err != nil {
		ui.Warning("Failed to list proxies: %v", err)
	} else {
		for _, proxy := range proxies {
			ui.Info("  %s:%d -> localhost:%d (%s)", proxy.ServiceName, proxy.ServicePort, proxy.LocalPort, proxy.ProxyName)
		}
	}

	return nil
}

// runComposeProxyStop executes the compose proxy-stop command.
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
//   - Docker container stopping
func runComposeProxyStop(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

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

	// Get network name (Docker Compose format: <project-name>_<network-name>)
	networkName := getComposeNetworkName(config.ProjectName)

	// Create port proxy manager
	manager := portproxy.NewManager(runner, config.ProjectName, networkName)

	ui.Info("Stopping all port proxies...")

	if err := manager.StopAllProxies(ctx); err != nil {
		return fmt.Errorf("failed to stop proxies: %w", err)
	}

	ui.Success("All port proxies stopped successfully!")

	return nil
}
