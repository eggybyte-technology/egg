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
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.eggybyte.com/egg/cli/internal/projectfs"
	"go.eggybyte.com/egg/cli/internal/templates"
	"go.eggybyte.com/egg/cli/internal/ui"
	"gopkg.in/yaml.v3"
)

// initCmd represents the init command.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new egg project",
	Long: `Initialize a new egg project with the required directory structure and configuration.

This command creates:
- Project directory structure (api/, backend/, frontend/, gen/, docker/, deploy/)
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

	// Check if project directory already exists
	if _, err := os.Stat(initProjectName); err == nil {
		return fmt.Errorf("project directory '%s' already exists. Use --force to overwrite or choose a different name", initProjectName)
	}

	// Create project directory
	if err := os.MkdirAll(initProjectName, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Create project file system (inside the project directory)
	fs := projectfs.NewProjectFS(initProjectName)
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

	// Generate docker configuration
	if err := generateDockerConfiguration(fs); err != nil {
		return fmt.Errorf("failed to generate docker configuration: %w", err)
	}

	// Generate .gitignore
	if err := generateGitignore(fs); err != nil {
		return fmt.Errorf("failed to generate .gitignore: %w", err)
	}

	// Generate .dockerignore
	if err := generateDockerignore(fs); err != nil {
		return fmt.Errorf("failed to generate .dockerignore: %w", err)
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
		"docker",
		"deploy/compose",
		"deploy/helm",
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
  go_builder_image: "ghcr.io/eggybyte-technology/eggybyte-go-builder:go1.25.1-alpine3.22"
  go_runtime_image: "ghcr.io/eggybyte-technology/eggybyte-go-alpine:go1.25.1-alpine3.22"
  nginx_image: "nginx:1.27.2-alpine"

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

# Backend services configuration
# Note: Image names are automatically computed as <project_name>-<service_name>
backend: {}

# Frontend services configuration
# Note: Image names are automatically computed as <project_name>-<service_name>
frontend: {}

database:
  enabled: true
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

// generateDockerConfiguration generates docker configuration files.
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
func generateDockerConfiguration(fs *projectfs.ProjectFS) error {
	ui.Info("Generating docker configuration...")

	loader := templates.NewLoader()

	// Read egg.yaml for template data
	yamlData, err := fs.ReadFile("egg.yaml")
	if err != nil {
		return fmt.Errorf("failed to read egg.yaml: %w", err)
	}

	// Parse config
	type Config struct {
		ProjectName  string `yaml:"project_name"`
		ModulePrefix string `yaml:"module_prefix"`
		Version      string `yaml:"version"`
	}
	var config Config
	if err := yaml.Unmarshal([]byte(yamlData), &config); err != nil {
		return fmt.Errorf("failed to parse egg.yaml: %w", err)
	}

	// Template data
	data := map[string]interface{}{
		"ProjectName":  config.ProjectName,
		"ModulePrefix": config.ModulePrefix,
		"Version":      config.Version,
	}

	// Generate backend Dockerfile from template
	backendDockerfile, err := loader.LoadAndRender("docker/Dockerfile.backend.tmpl", data)
	if err != nil {
		return fmt.Errorf("failed to load and render Dockerfile.backend template: %w", err)
	}
	if err := fs.WriteFile("docker/Dockerfile.backend", backendDockerfile, 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile.backend: %w", err)
	}

	// Generate frontend Dockerfile from template
	frontendDockerfile, err := loader.LoadAndRender("docker/Dockerfile.frontend.tmpl", data)
	if err != nil {
		return fmt.Errorf("failed to load and render Dockerfile.frontend template: %w", err)
	}
	if err := fs.WriteFile("docker/Dockerfile.frontend", frontendDockerfile, 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile.frontend: %w", err)
	}

	// Note: Runtime Dockerfile (eggybyte-go-alpine) is no longer generated by CLI
	// Users should use the pre-built runtime image directly

	// Generate nginx.conf from template
	nginxConf, err := loader.LoadTemplate("docker/nginx.conf.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load nginx.conf template: %w", err)
	}
	if err := fs.WriteFile("docker/nginx.conf", nginxConf, 0644); err != nil {
		return fmt.Errorf("failed to write nginx.conf: %w", err)
	}

	ui.Success("Docker configuration generated")
	return nil
}

// generateGitignore generates the .gitignore file.
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
func generateGitignore(fs *projectfs.ProjectFS) error {
	ui.Info("Generating .gitignore...")

	loader := templates.NewLoader()

	// Generate .gitignore from template
	gitignore, err := loader.LoadTemplate(".gitignore.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load .gitignore template: %w", err)
	}
	if err := fs.WriteFile(".gitignore", gitignore, 0644); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	ui.Success(".gitignore generated")
	return nil
}

// generateDockerignore generates the .dockerignore file.
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
func generateDockerignore(fs *projectfs.ProjectFS) error {
	ui.Info("Generating .dockerignore...")

	loader := templates.NewLoader()

	// Generate .dockerignore from template
	dockerignore, err := loader.LoadTemplate(".dockerignore.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load .dockerignore template: %w", err)
	}
	if err := fs.WriteFile(".dockerignore", dockerignore, 0644); err != nil {
		return fmt.Errorf("failed to write .dockerignore: %w", err)
	}

	ui.Success(".dockerignore generated")
	return nil
}
