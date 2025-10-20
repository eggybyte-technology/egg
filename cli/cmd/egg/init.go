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
//	egg init [flags]
package egg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/eggybyte-technology/egg/cli/internal/projectfs"
	"github.com/eggybyte-technology/egg/cli/internal/templates"
	"github.com/eggybyte-technology/egg/cli/internal/ui"
	"github.com/spf13/cobra"
)

// initCmd represents the init command.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new egg project",
	Long: `Initialize a new egg project with the required directory structure and configuration.

This command creates:
- Project directory structure (api/, backend/, frontend/, gen/, build/, deploy/)
- egg.yaml configuration file
- Basic buf configuration for API definitions
- Go workspace configuration for backend services

Example:
  egg init
  egg init --project-name my-platform
  egg init --module-prefix github.com/myorg/my-platform`,
	RunE: runInit,
}

var (
	initProjectName    string
	initModulePrefix   string
	initDockerRegistry string
	initVersion        string
)

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVar(&initProjectName, "project-name", "", "Project name (default: current directory name)")
	initCmd.Flags().StringVar(&initModulePrefix, "module-prefix", "", "Go module prefix (default: github.com/eggybyte-technology/<project-name>)")
	initCmd.Flags().StringVar(&initDockerRegistry, "docker-registry", "ghcr.io/eggybyte-technology", "Docker registry URL")
	initCmd.Flags().StringVar(&initVersion, "version", "v1.0.0", "Project version")
}

// runInit executes the init command.
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
//   - File system operations and template rendering
func runInit(cmd *cobra.Command, args []string) error {
	_ = context.Background()

	// Get current directory as default project name
	if initProjectName == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		initProjectName = filepath.Base(wd)
	}

	// Set default module prefix
	if initModulePrefix == "" {
		initModulePrefix = fmt.Sprintf("github.com/eggybyte-technology/%s", initProjectName)
	}

	ui.Info("Initializing egg project: %s", initProjectName)

	// Create project file system
	fs := projectfs.NewProjectFS(".")
	fs.SetVerbose(true)

	// Create directory structure
	if err := createProjectStructure(fs); err != nil {
		return fmt.Errorf("failed to create project structure: %w", err)
	}

	// Generate egg.yaml
	if err := generateEggYAML(fs, initProjectName, initModulePrefix, initDockerRegistry, initVersion); err != nil {
		return fmt.Errorf("failed to generate egg.yaml: %w", err)
	}

	// Generate API configuration
	if err := generateAPIConfiguration(fs); err != nil {
		return fmt.Errorf("failed to generate API configuration: %w", err)
	}

	// Generate backend workspace
	if err := generateBackendWorkspace(fs); err != nil {
		return fmt.Errorf("failed to generate backend workspace: %w", err)
	}

	// Generate build configuration
	if err := generateBuildConfiguration(fs); err != nil {
		return fmt.Errorf("failed to generate build configuration: %w", err)
	}

	ui.Success("Project initialized successfully!")
	ui.Info("Next steps:")
	ui.Info("  1. Create a backend service: egg create backend <name>")
	ui.Info("  2. Initialize API definitions: egg api init")
	ui.Info("  3. Generate code: egg api generate")
	ui.Info("  4. Start development: egg compose up")

	return nil
}

// createProjectStructure creates the required directory structure.
//
// Parameters:
//   - fs: Project file system
//
// Returns:
//   - error: Creation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Directory creation operations
func createProjectStructure(fs *projectfs.ProjectFS) error {
	ui.Info("Creating project structure...")

	directories := []string{
		"api",
		"backend",
		"frontend",
		"gen/go",
		"gen/dart",
		"gen/ts",
		"gen/openapi",
		"build",
		"deploy",
	}

	for _, dir := range directories {
		if err := fs.CreateDirectory(dir); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	ui.Success("Project structure created")
	return nil
}

// generateEggYAML generates the egg.yaml configuration file.
//
// Parameters:
//   - fs: Project file system
//   - projectName: Project name
//   - modulePrefix: Go module prefix
//   - dockerRegistry: Docker registry URL
//   - version: Project version
//
// Returns:
//   - error: Generation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering and file I/O
func generateEggYAML(fs *projectfs.ProjectFS, projectName, modulePrefix, dockerRegistry, version string) error {
	ui.Info("Generating egg.yaml...")

	eggYAML := fmt.Sprintf(`config_version: "1.0"
project_name: "%s"
version: "%s"
module_prefix: "%s"
docker_registry: "%s"

build:
  platforms: ["linux/amd64", "linux/arm64"]
  go_runtime_image: "eggybyte-go-alpine"

env:
  global:
    LOG_LEVEL: "info"
    KUBERNETES_NAMESPACE: "prod"
  backend:
    DATABASE_DSN: "user:pass@tcp(mysql:3306)/app?charset=utf8mb4&parseTime=True"
  frontend:
    FLUTTER_BASE_HREF: "/"

backend_defaults:
  ports:
    http: 8080
    health: 8081
    metrics: 9091

kubernetes:
  resources:
    configmaps:
      global-config:
        FEATURE_A: "on"
    secrets:
      jwtkey:
        KEY: "super-secret"

backend: {}

frontend: {}

database:
  enabled: false
  image: "mysql:9.4"
  port: 3306
  root_password: "rootpass"
  database: "app"
  user: "user"
  password: "pass"
`, projectName, version, modulePrefix, dockerRegistry)

	if err := fs.WriteFile("egg.yaml", eggYAML, 0644); err != nil {
		return fmt.Errorf("failed to write egg.yaml: %w", err)
	}

	ui.Success("egg.yaml generated")
	return nil
}

// generateAPIConfiguration generates basic API configuration.
//
// Parameters:
//   - fs: Project file system
//
// Returns:
//   - error: Generation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering and file I/O
func generateAPIConfiguration(fs *projectfs.ProjectFS) error {
	ui.Info("Generating API configuration...")

	loader := templates.NewLoader()

	// Generate buf.yaml from template
	bufYAML, err := loader.LoadTemplate("api/buf.yaml.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load buf.yaml template: %w", err)
	}
	if err := fs.WriteFile("api/buf.yaml", bufYAML, 0644); err != nil {
		return fmt.Errorf("failed to write buf.yaml: %w", err)
	}

	// Generate buf.gen.yaml from template
	bufGenYAML, err := loader.LoadTemplate("api/buf.gen.yaml.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load buf.gen.yaml template: %w", err)
	}
	if err := fs.WriteFile("api/buf.gen.yaml", bufGenYAML, 0644); err != nil {
		return fmt.Errorf("failed to write buf.gen.yaml: %w", err)
	}

	ui.Success("API configuration generated")
	return nil
}

// generateBackendWorkspace generates the backend Go workspace.
//
// Parameters:
//   - fs: Project file system
//
// Returns:
//   - error: Generation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - File I/O operations
func generateBackendWorkspace(fs *projectfs.ProjectFS) error {
	ui.Info("Generating backend workspace...")

	// Note: go.work will be created using go work init command when first service is created
	// This function just ensures the backend directory exists
	ui.Success("Backend workspace prepared")
	return nil
}

// generateBuildConfiguration generates build configuration files.
//
// Parameters:
//   - fs: Project file system
//
// Returns:
//   - error: Generation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering and file I/O
func generateBuildConfiguration(fs *projectfs.ProjectFS) error {
	ui.Info("Generating build configuration...")

	loader := templates.NewLoader()

	// Generate backend Dockerfile from template
	backendDockerfile, err := loader.LoadTemplate("build/Dockerfile.backend.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load Dockerfile.backend template: %w", err)
	}
	if err := fs.WriteFile("build/Dockerfile.backend", backendDockerfile, 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile.backend: %w", err)
	}

	// Generate frontend Dockerfile from template
	frontendDockerfile, err := loader.LoadTemplate("build/Dockerfile.frontend.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load Dockerfile.frontend template: %w", err)
	}
	if err := fs.WriteFile("build/Dockerfile.frontend", frontendDockerfile, 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile.frontend: %w", err)
	}

	// Generate runtime Dockerfile from template
	runtimeDockerfile, err := loader.LoadTemplate("build/Dockerfile.eggybyte-go-alpine.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load Dockerfile.eggybyte-go-alpine template: %w", err)
	}
	if err := fs.WriteFile("build/Dockerfile.eggybyte-go-alpine", runtimeDockerfile, 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile.eggybyte-go-alpine: %w", err)
	}

	// Generate nginx.conf from template
	nginxConf, err := loader.LoadTemplate("build/nginx.conf.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load nginx.conf template: %w", err)
	}
	if err := fs.WriteFile("build/nginx.conf", nginxConf, 0644); err != nil {
		return fmt.Errorf("failed to write nginx.conf: %w", err)
	}

	ui.Success("Build configuration generated")
	return nil
}
