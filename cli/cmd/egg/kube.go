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
//	egg kube template [-n <namespace>]
//	egg kube apply [-n <namespace>]
//	egg kube uninstall [-n <namespace>]
package egg

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/eggybyte-technology/egg/cli/internal/configschema"
	"github.com/eggybyte-technology/egg/cli/internal/projectfs"
	"github.com/eggybyte-technology/egg/cli/internal/ref"
	"github.com/eggybyte-technology/egg/cli/internal/render/helm"
	"github.com/eggybyte-technology/egg/cli/internal/toolrunner"
	"github.com/eggybyte-technology/egg/cli/internal/ui"
	"github.com/spf13/cobra"
)

// kubeCmd represents the kube command.
var kubeCmd = &cobra.Command{
	Use:   "kube",
	Short: "Manage Kubernetes deployments",
	Long: `Manage Kubernetes deployments with Helm.

This command provides:
- Helm chart generation and templating
- Kubernetes manifest application
- Namespace management
- Resource cleanup and uninstallation

Examples:
  egg kube template
  egg kube template -n production
  egg kube apply -n production
  egg kube uninstall -n production`,
}

// kubeTemplateCmd represents the kube template command.
var kubeTemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "Generate Helm templates",
	Long: `Generate Helm templates from egg configuration.

This command:
- Renders Helm charts for all services
- Generates Kubernetes manifests
- Applies configuration expressions
- Creates deployment-ready templates

Example:
  egg kube template
  egg kube template -n production`,
	RunE: runKubeTemplate,
}

// kubeApplyCmd represents the kube apply command.
var kubeApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply Kubernetes manifests",
	Long: `Apply Kubernetes manifests to cluster.

This command:
- Applies Helm charts to Kubernetes cluster
- Creates namespaces and resources
- Manages service deployments
- Handles configuration updates

Example:
  egg kube apply
  egg kube apply -n production`,
	RunE: runKubeApply,
}

// kubeUninstallCmd represents the kube uninstall command.
var kubeUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Kubernetes resources",
	Long: `Uninstall Kubernetes resources and clean up.

This command:
- Removes Helm releases
- Deletes namespaces and resources
- Cleans up persistent volumes
- Removes configuration secrets

Example:
  egg kube uninstall
  egg kube uninstall -n production`,
	RunE: runKubeUninstall,
}

var (
	namespace string
)

func init() {
	rootCmd.AddCommand(kubeCmd)
	kubeCmd.AddCommand(kubeTemplateCmd)
	kubeCmd.AddCommand(kubeApplyCmd)
	kubeCmd.AddCommand(kubeUninstallCmd)

	kubeTemplateCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	kubeApplyCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	kubeUninstallCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
}

// runKubeTemplate executes the kube template command.
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
//   - Helm chart generation and templating
func runKubeTemplate(cmd *cobra.Command, args []string) error {
	_ = context.Background()

	ui.Info("Generating Helm templates...")

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

	// Create Helm renderer
	helmRenderer := helm.NewRenderer(fs, refParser)

	// Render Helm charts
	if err := helmRenderer.Render(config); err != nil {
		return fmt.Errorf("failed to render Helm charts: %w", err)
	}

	ui.Success("Helm templates generated successfully!")
	ui.Info("Generated charts:")

	// List generated charts
	helmDir := "deploy/helm"
	if entries, err := fs.ListDirectories(helmDir); err == nil {
		for _, chart := range entries {
			ui.Info("  - %s", chart)
		}
	}

	ui.Info("Next steps:")
	ui.Info("  1. Review generated templates in deploy/helm/")
	ui.Info("  2. Apply to cluster: egg kube apply -n %s", namespace)
	ui.Info("  3. Monitor deployment: kubectl get pods -n %s", namespace)

	return nil
}

// runKubeApply executes the kube apply command.
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
//   - Kubernetes manifest application
func runKubeApply(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	ui.Info("Applying Kubernetes manifests to namespace: %s", namespace)

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

	// Create namespace if it doesn't exist
	if err := createNamespace(ctx, runner, namespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Apply Helm charts
	if err := applyHelmCharts(ctx, runner, config, namespace); err != nil {
		return fmt.Errorf("failed to apply Helm charts: %w", err)
	}

	ui.Success("Kubernetes manifests applied successfully!")
	ui.Info("Deployment status:")
	ui.Info("  Namespace: %s", namespace)
	ui.Info("  Project: %s", config.ProjectName)

	ui.Info("Next steps:")
	ui.Info("  1. Check pod status: kubectl get pods -n %s", namespace)
	ui.Info("  2. Check services: kubectl get services -n %s", namespace)
	ui.Info("  3. View logs: kubectl logs -n %s -l app.kubernetes.io/name=<service>", namespace)

	return nil
}

// runKubeUninstall executes the kube uninstall command.
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
//   - Kubernetes resource cleanup
func runKubeUninstall(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	ui.Info("Uninstalling Kubernetes resources from namespace: %s", namespace)

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

	// Uninstall Helm charts
	if err := uninstallHelmCharts(ctx, runner, config, namespace); err != nil {
		return fmt.Errorf("failed to uninstall Helm charts: %w", err)
	}

	ui.Success("Kubernetes resources uninstalled successfully!")
	ui.Info("Cleaned up resources in namespace: %s", namespace)

	return nil
}

// createNamespace creates a Kubernetes namespace if it doesn't exist.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//   - namespace: Namespace name
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Kubernetes namespace creation
func createNamespace(ctx context.Context, runner *toolrunner.Runner, namespace string) error {
	// Check if namespace exists
	result, err := runner.Kubectl(ctx, "get", "namespace", namespace)
	if err == nil && result.ExitCode == 0 {
		ui.Debug("Namespace %s already exists", namespace)
		return nil
	}

	// Create namespace
	ui.Info("Creating namespace: %s", namespace)
	result, err = runner.Kubectl(ctx, "create", "namespace", namespace)
	if err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	if runner.GetVerbose() {
		ui.Debug("Namespace creation output: %s", result.Stdout)
	}

	return nil
}

// applyHelmCharts applies Helm charts to Kubernetes cluster.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//   - config: Project configuration
//   - namespace: Kubernetes namespace
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Helm chart installation
func applyHelmCharts(ctx context.Context, runner *toolrunner.Runner, config *configschema.Config, namespace string) error {
	helmDir := "deploy/helm"

	// Apply backend services
	for name := range config.Backend {
		chartPath := filepath.Join(helmDir, name)
		releaseName := fmt.Sprintf("%s-%s", config.ProjectName, name)

		ui.Info("Applying backend service: %s", name)
		if err := applyHelmChart(ctx, runner, chartPath, releaseName, namespace); err != nil {
			return fmt.Errorf("failed to apply chart for %s: %w", name, err)
		}
	}

	// Apply frontend services
	for name := range config.Frontend {
		chartPath := filepath.Join(helmDir, name)
		releaseName := fmt.Sprintf("%s-%s", config.ProjectName, name)

		ui.Info("Applying frontend service: %s", name)
		if err := applyHelmChart(ctx, runner, chartPath, releaseName, namespace); err != nil {
			return fmt.Errorf("failed to apply chart for %s: %w", name, err)
		}
	}

	// Apply database if enabled
	if config.Database.Enabled {
		chartPath := filepath.Join(helmDir, "mysql")
		releaseName := fmt.Sprintf("%s-mysql", config.ProjectName)

		ui.Info("Applying database service: mysql")
		if err := applyHelmChart(ctx, runner, chartPath, releaseName, namespace); err != nil {
			return fmt.Errorf("failed to apply database chart: %w", err)
		}
	}

	// Apply resources
	chartPath := filepath.Join(helmDir, "resources")
	releaseName := fmt.Sprintf("%s-resources", config.ProjectName)

	ui.Info("Applying Kubernetes resources")
	if err := applyHelmChart(ctx, runner, chartPath, releaseName, namespace); err != nil {
		return fmt.Errorf("failed to apply resources chart: %w", err)
	}

	return nil
}

// applyHelmChart applies a single Helm chart.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//   - chartPath: Chart directory path
//   - releaseName: Helm release name
//   - namespace: Kubernetes namespace
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Single Helm chart installation
func applyHelmChart(ctx context.Context, runner *toolrunner.Runner, chartPath, releaseName, namespace string) error {
	args := []string{
		"upgrade", "--install",
		releaseName,
		chartPath,
		"--namespace", namespace,
		"--create-namespace",
	}

	result, err := runner.Helm(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to apply Helm chart: %w", err)
	}

	if runner.GetVerbose() {
		ui.Debug("Helm chart output: %s", result.Stdout)
	}

	return nil
}

// uninstallHelmCharts uninstalls Helm charts from Kubernetes cluster.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//   - config: Project configuration
//   - namespace: Kubernetes namespace
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Helm chart uninstallation
func uninstallHelmCharts(ctx context.Context, runner *toolrunner.Runner, config *configschema.Config, namespace string) error {
	// Uninstall backend services
	for name := range config.Backend {
		releaseName := fmt.Sprintf("%s-%s", config.ProjectName, name)

		ui.Info("Uninstalling backend service: %s", name)
		if err := uninstallHelmChart(ctx, runner, releaseName, namespace); err != nil {
			return fmt.Errorf("failed to uninstall chart for %s: %w", name, err)
		}
	}

	// Uninstall frontend services
	for name := range config.Frontend {
		releaseName := fmt.Sprintf("%s-%s", config.ProjectName, name)

		ui.Info("Uninstalling frontend service: %s", name)
		if err := uninstallHelmChart(ctx, runner, releaseName, namespace); err != nil {
			return fmt.Errorf("failed to uninstall chart for %s: %w", name, err)
		}
	}

	// Uninstall database if enabled
	if config.Database.Enabled {
		releaseName := fmt.Sprintf("%s-mysql", config.ProjectName)

		ui.Info("Uninstalling database service: mysql")
		if err := uninstallHelmChart(ctx, runner, releaseName, namespace); err != nil {
			return fmt.Errorf("failed to uninstall database chart: %w", err)
		}
	}

	// Uninstall resources
	releaseName := fmt.Sprintf("%s-resources", config.ProjectName)

	ui.Info("Uninstalling Kubernetes resources")
	if err := uninstallHelmChart(ctx, runner, releaseName, namespace); err != nil {
		return fmt.Errorf("failed to uninstall resources chart: %w", err)
	}

	return nil
}

// uninstallHelmChart uninstalls a single Helm chart.
//
// Parameters:
//   - ctx: Context for cancellation
//   - runner: Tool runner
//   - releaseName: Helm release name
//   - namespace: Kubernetes namespace
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Single Helm chart uninstallation
func uninstallHelmChart(ctx context.Context, runner *toolrunner.Runner, releaseName, namespace string) error {
	args := []string{
		"uninstall",
		releaseName,
		"--namespace", namespace,
	}

	result, err := runner.Helm(ctx, args...)
	if err != nil {
		// Ignore errors for non-existent releases
		if strings.Contains(result.Stderr, "not found") {
			ui.Debug("Release %s not found, skipping", releaseName)
			return nil
		}
		return fmt.Errorf("failed to uninstall Helm chart: %w", err)
	}

	if runner.GetVerbose() {
		ui.Debug("Helm uninstall output: %s", result.Stdout)
	}

	return nil
}
