// Package lint provides code and configuration linting for egg projects.
//
// Overview:
//   - Responsibility: Validate project structure, configuration, and code quality
//   - Key Types: Linter rules, validators, checkers
//   - Concurrency Model: Immutable validation with parallel rule execution
//   - Error Semantics: Structured linting errors with suggestions
//   - Performance Notes: Parallel rule execution, cached validation results
//
// Usage:
//
//	linter := NewLinter()
//	results, err := linter.Check(config, fs)
package lint

import (
	"fmt"
	"path/filepath"
	"strings"

	"go.eggybyte.com/egg/cli/internal/configschema"
	"go.eggybyte.com/egg/cli/internal/projectfs"
	"go.eggybyte.com/egg/cli/internal/ref"
	"go.eggybyte.com/egg/cli/internal/ui"
)

// Linter provides project linting functionality.
//
// Parameters:
//   - None (stateless linter)
//
// Returns:
//   - None (data structure)
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Stateless, efficient validation
type Linter struct {
	refParser *ref.Parser
}

// LintResult represents the result of a linting operation.
//
// Parameters:
//   - Rule: Linting rule name
//   - Level: Severity level
//   - Message: Human-readable message
//   - Path: Configuration path
//   - Suggestion: Fix suggestion
//
// Returns:
//   - None (data structure)
//
// Concurrency:
//   - Immutable after creation
//
// Performance:
//   - Minimal memory footprint
type LintResult struct {
	Rule       string `json:"rule"`
	Level      string `json:"level"`
	Message    string `json:"message"`
	Path       string `json:"path,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// LintResults represents a collection of linting results.
//
// Parameters:
//   - Results: List of linting results
//   - ErrorCount: Number of error-level issues
//   - WarningCount: Number of warning-level issues
//   - InfoCount: Number of info-level issues
//
// Returns:
//   - None (data structure)
//
// Concurrency:
//   - Immutable after creation
//
// Performance:
//   - Minimal memory footprint
type LintResults struct {
	Results      []LintResult `json:"results"`
	ErrorCount   int          `json:"error_count"`
	WarningCount int          `json:"warning_count"`
	InfoCount    int          `json:"info_count"`
}

// NewLinter creates a new project linter.
//
// Parameters:
//   - None
//
// Returns:
//   - *Linter: Linter instance
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Minimal initialization overhead
func NewLinter() *Linter {
	return &Linter{
		refParser: ref.NewParser(),
	}
}

// Check performs comprehensive project linting.
//
// Parameters:
//   - config: Project configuration
//   - fs: Project file system
//
// Returns:
//   - *LintResults: Linting results
//   - error: Linting error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Parallel rule execution
func (l *Linter) Check(config *configschema.Config, fs *projectfs.ProjectFS) (*LintResults, error) {
	ui.Info("Running project linting...")

	results := &LintResults{
		Results: make([]LintResult, 0),
	}

	// Check project structure
	if err := l.checkProjectStructure(fs, results); err != nil {
		return nil, fmt.Errorf("failed to check project structure: %w", err)
	}

	// Check configuration
	if err := l.checkConfiguration(config, results); err != nil {
		return nil, fmt.Errorf("failed to check configuration: %w", err)
	}

	// Check backend services
	if err := l.checkBackendServices(config, fs, results); err != nil {
		return nil, fmt.Errorf("failed to check backend services: %w", err)
	}

	// Check frontend services
	if err := l.checkFrontendServices(config, fs, results); err != nil {
		return nil, fmt.Errorf("failed to check frontend services: %w", err)
	}

	// Check database configuration
	if err := l.checkDatabaseConfiguration(config, results); err != nil {
		return nil, fmt.Errorf("failed to check database configuration: %w", err)
	}

	// Check Kubernetes resources
	if err := l.checkKubernetesResources(config, results); err != nil {
		return nil, fmt.Errorf("failed to check Kubernetes resources: %w", err)
	}

	// Count results by level
	for _, result := range results.Results {
		switch result.Level {
		case "error":
			results.ErrorCount++
		case "warning":
			results.WarningCount++
		case "info":
			results.InfoCount++
		}
	}

	ui.Success("Linting completed: %d errors, %d warnings, %d info",
		results.ErrorCount, results.WarningCount, results.InfoCount)

	return results, nil
}

// checkProjectStructure validates the project directory structure.
//
// Parameters:
//   - fs: Project file system
//   - results: Linting results collection
//
// Returns:
//   - error: Validation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Directory traversal
func (l *Linter) checkProjectStructure(fs *projectfs.ProjectFS, results *LintResults) error {
	// Check required directories
	requiredDirs := []string{
		"api",
		"backend",
		"frontend",
		"gen",
		"build",
		"deploy",
	}

	for _, dir := range requiredDirs {
		exists, err := fs.DirectoryExists(dir)
		if err != nil {
			return fmt.Errorf("failed to check directory %s: %w", dir, err)
		}

		if !exists {
			results.Results = append(results.Results, LintResult{
				Rule:       "project-structure",
				Level:      "error",
				Message:    fmt.Sprintf("Required directory missing: %s", dir),
				Path:       dir,
				Suggestion: fmt.Sprintf("Create directory: mkdir -p %s", dir),
			})
		}
	}

	// Check egg.yaml
	exists, err := fs.FileExists("egg.yaml")
	if err != nil {
		return fmt.Errorf("failed to check egg.yaml: %w", err)
	}

	if !exists {
		results.Results = append(results.Results, LintResult{
			Rule:       "project-structure",
			Level:      "error",
			Message:    "Configuration file missing: egg.yaml",
			Path:       "egg.yaml",
			Suggestion: "Run 'egg init' to create egg.yaml",
		})
	}

	return nil
}

// checkConfiguration validates the project configuration.
//
// Parameters:
//   - config: Project configuration
//   - results: Linting results collection
//
// Returns:
//   - error: Validation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Configuration validation
func (l *Linter) checkConfiguration(config *configschema.Config, results *LintResults) error {
	// Check project name
	if config.ProjectName == "" {
		results.Results = append(results.Results, LintResult{
			Rule:       "configuration",
			Level:      "error",
			Message:    "Project name is required",
			Path:       "project_name",
			Suggestion: "Set a valid project name",
		})
	}

	// Check module prefix
	if config.ModulePrefix == "" {
		results.Results = append(results.Results, LintResult{
			Rule:       "configuration",
			Level:      "error",
			Message:    "Module prefix is required",
			Path:       "module_prefix",
			Suggestion: "Set a valid Go module prefix",
		})
	}

	// Check docker registry
	if config.DockerRegistry == "" {
		results.Results = append(results.Results, LintResult{
			Rule:       "configuration",
			Level:      "warning",
			Message:    "Docker registry not specified",
			Path:       "docker_registry",
			Suggestion: "Set a valid Docker registry URL",
		})
	}

	// Check build configuration
	if len(config.Build.Platforms) == 0 {
		results.Results = append(results.Results, LintResult{
			Rule:       "configuration",
			Level:      "warning",
			Message:    "No build platforms specified",
			Path:       "build.platforms",
			Suggestion: "Specify target platforms (e.g., linux/amd64, linux/arm64)",
		})
	}

	return nil
}

// checkBackendServices validates backend service configurations.
//
// Parameters:
//   - config: Project configuration
//   - fs: Project file system
//   - results: Linting results collection
//
// Returns:
//   - error: Validation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Service validation
func (l *Linter) checkBackendServices(config *configschema.Config, fs *projectfs.ProjectFS, results *LintResults) error {
	for name, service := range config.Backend {
		// Check service name
		if !isValidServiceName(name) {
			results.Results = append(results.Results, LintResult{
				Rule:       "backend-service",
				Level:      "error",
				Message:    fmt.Sprintf("Invalid service name: %s", name),
				Path:       fmt.Sprintf("backend.%s", name),
				Suggestion: "Use lowercase letters, numbers, hyphens, and underscores only",
			})
		}

		// Image name is now auto-calculated from project_name and service_name
		// No validation needed

		// Check port inheritance
		if service.Ports != nil {
			ports := service.Ports
			defaults := config.BackendDefaults.Ports

			if ports.HTTP == defaults.HTTP && ports.Health == defaults.Health && ports.Metrics == defaults.Metrics {
				results.Results = append(results.Results, LintResult{
					Rule:       "backend-service",
					Level:      "info",
					Message:    "Ports match defaults, consider removing custom port configuration",
					Path:       fmt.Sprintf("backend.%s.ports", name),
					Suggestion: "Remove ports section to use defaults",
				})
			}
		}

		// Check service directory structure
		serviceDir := filepath.Join("backend", name)
		exists, err := fs.DirectoryExists(serviceDir)
		if err != nil {
			return fmt.Errorf("failed to check service directory %s: %w", serviceDir, err)
		}

		if !exists {
			results.Results = append(results.Results, LintResult{
				Rule:       "backend-service",
				Level:      "error",
				Message:    fmt.Sprintf("Service directory missing: %s", serviceDir),
				Path:       fmt.Sprintf("backend.%s", name),
				Suggestion: fmt.Sprintf("Run 'egg create backend %s' to create service", name),
			})
		} else {
			// Check go.mod
			goModPath := filepath.Join(serviceDir, "go.mod")
			exists, err := fs.FileExists(goModPath)
			if err != nil {
				return fmt.Errorf("failed to check go.mod: %w", err)
			}

			if !exists {
				results.Results = append(results.Results, LintResult{
					Rule:       "backend-service",
					Level:      "error",
					Message:    "go.mod file missing",
					Path:       goModPath,
					Suggestion: "Initialize Go module",
				})
			}

			// Check main.go
			mainGoPath := filepath.Join(serviceDir, "cmd", "server", "main.go")
			exists, err = fs.FileExists(mainGoPath)
			if err != nil {
				return fmt.Errorf("failed to check main.go: %w", err)
			}

			if !exists {
				results.Results = append(results.Results, LintResult{
					Rule:       "backend-service",
					Level:      "error",
					Message:    "main.go file missing",
					Path:       mainGoPath,
					Suggestion: "Create main.go entry point",
				})
			}
		}

		// Check Kubernetes service names
		if service.Kubernetes.Service.ClusterIP.Name == "" {
			results.Results = append(results.Results, LintResult{
				Rule:       "backend-service",
				Level:      "warning",
				Message:    "Missing clusterIP service name",
				Path:       fmt.Sprintf("backend.%s.kubernetes.service.clusterIP.name", name),
				Suggestion: "Set a descriptive service name",
			})
		}

		if service.Kubernetes.Service.Headless.Name == "" {
			results.Results = append(results.Results, LintResult{
				Rule:       "backend-service",
				Level:      "warning",
				Message:    "Missing headless service name",
				Path:       fmt.Sprintf("backend.%s.kubernetes.service.headless.name", name),
				Suggestion: "Set a descriptive service name",
			})
		}

		// Check environment variable expressions
		for key, value := range service.Env.Kubernetes {
			if err := l.checkExpression(value, fmt.Sprintf("backend.%s.env.kubernetes.%s", name, key), config, results); err != nil {
				return fmt.Errorf("failed to check expression: %w", err)
			}
		}

		for key, value := range service.Env.Docker {
			if err := l.checkExpression(value, fmt.Sprintf("backend.%s.env.docker.%s", name, key), config, results); err != nil {
				return fmt.Errorf("failed to check expression: %w", err)
			}
		}
	}

	return nil
}

// checkFrontendServices validates frontend service configurations.
//
// Parameters:
//   - config: Project configuration
//   - fs: Project file system
//   - results: Linting results collection
//
// Returns:
//   - error: Validation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Service validation
func (l *Linter) checkFrontendServices(config *configschema.Config, fs *projectfs.ProjectFS, results *LintResults) error {
	for name, service := range config.Frontend {
		// Check service name
		if !isValidServiceName(name) {
			results.Results = append(results.Results, LintResult{
				Rule:       "frontend-service",
				Level:      "error",
				Message:    fmt.Sprintf("Invalid service name: %s", name),
				Path:       fmt.Sprintf("frontend.%s", name),
				Suggestion: "Use lowercase letters, numbers, hyphens, and underscores only",
			})
		}

		// Image name is now auto-calculated from project_name and service_name
		// No validation needed

		// Check platforms
		if len(service.Platforms) == 0 {
			results.Results = append(results.Results, LintResult{
				Rule:       "frontend-service",
				Level:      "warning",
				Message:    "No platforms specified",
				Path:       fmt.Sprintf("frontend.%s.platforms", name),
				Suggestion: "Specify target platforms (e.g., web, mobile)",
			})
		}

		// Check service directory structure
		serviceDir := filepath.Join("frontend", name)
		exists, err := fs.DirectoryExists(serviceDir)
		if err != nil {
			return fmt.Errorf("failed to check service directory %s: %w", serviceDir, err)
		}

		if !exists {
			results.Results = append(results.Results, LintResult{
				Rule:       "frontend-service",
				Level:      "error",
				Message:    fmt.Sprintf("Service directory missing: %s", serviceDir),
				Path:       fmt.Sprintf("frontend.%s", name),
				Suggestion: fmt.Sprintf("Run 'egg create frontend %s' to create service", name),
			})
		} else {
			// Check pubspec.yaml
			pubspecPath := filepath.Join(serviceDir, "pubspec.yaml")
			exists, err := fs.FileExists(pubspecPath)
			if err != nil {
				return fmt.Errorf("failed to check pubspec.yaml: %w", err)
			}

			if !exists {
				results.Results = append(results.Results, LintResult{
					Rule:       "frontend-service",
					Level:      "error",
					Message:    "pubspec.yaml file missing",
					Path:       pubspecPath,
					Suggestion: "Initialize Flutter project",
				})
			}
		}
	}

	return nil
}

// checkDatabaseConfiguration validates database configuration.
//
// Parameters:
//   - config: Project configuration
//   - results: Linting results collection
//
// Returns:
//   - error: Validation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Configuration validation
func (l *Linter) checkDatabaseConfiguration(config *configschema.Config, results *LintResults) error {
	if !config.Database.Enabled {
		return nil
	}

	// Check required fields
	if config.Database.Database == "" {
		results.Results = append(results.Results, LintResult{
			Rule:       "database-configuration",
			Level:      "error",
			Message:    "Database name is required when enabled",
			Path:       "database.database",
			Suggestion: "Set a valid database name",
		})
	}

	if config.Database.User == "" {
		results.Results = append(results.Results, LintResult{
			Rule:       "database-configuration",
			Level:      "error",
			Message:    "Database user is required when enabled",
			Path:       "database.user",
			Suggestion: "Set a valid database user",
		})
	}

	if config.Database.Password == "" {
		results.Results = append(results.Results, LintResult{
			Rule:       "database-configuration",
			Level:      "error",
			Message:    "Database password is required when enabled",
			Path:       "database.password",
			Suggestion: "Set a valid database password",
		})
	}

	if config.Database.RootPassword == "" {
		results.Results = append(results.Results, LintResult{
			Rule:       "database-configuration",
			Level:      "error",
			Message:    "Database root password is required when enabled",
			Path:       "database.root_password",
			Suggestion: "Set a valid root password",
		})
	}

	// Check port
	if config.Database.Port <= 0 || config.Database.Port > 65535 {
		results.Results = append(results.Results, LintResult{
			Rule:       "database-configuration",
			Level:      "error",
			Message:    "Invalid database port",
			Path:       "database.port",
			Suggestion: "Use port numbers between 1 and 65535",
		})
	}

	return nil
}

// checkKubernetesResources validates Kubernetes resource definitions.
//
// Parameters:
//   - config: Project configuration
//   - results: Linting results collection
//
// Returns:
//   - error: Validation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Resource validation
func (l *Linter) checkKubernetesResources(config *configschema.Config, results *LintResults) error {
	// Check ConfigMap names
	for name := range config.Kubernetes.Resources.ConfigMaps {
		if !isValidResourceName(name) {
			results.Results = append(results.Results, LintResult{
				Rule:       "kubernetes-resources",
				Level:      "error",
				Message:    fmt.Sprintf("Invalid ConfigMap name: %s", name),
				Path:       fmt.Sprintf("kubernetes.resources.configmaps.%s", name),
				Suggestion: "Use lowercase letters, numbers, and hyphens only",
			})
		}
	}

	// Check Secret names
	for name := range config.Kubernetes.Resources.Secrets {
		if !isValidResourceName(name) {
			results.Results = append(results.Results, LintResult{
				Rule:       "kubernetes-resources",
				Level:      "error",
				Message:    fmt.Sprintf("Invalid Secret name: %s", name),
				Path:       fmt.Sprintf("kubernetes.resources.secrets.%s", name),
				Suggestion: "Use lowercase letters, numbers, and hyphens only",
			})
		}
	}

	return nil
}

// checkExpression validates reference expressions.
//
// Parameters:
//   - value: Expression value
//   - path: Configuration path
//   - config: Project configuration
//   - results: Linting results collection
//
// Returns:
//   - error: Validation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Expression parsing
func (l *Linter) checkExpression(value, path string, config *configschema.Config, results *LintResults) error {
	// Check if value contains expressions
	if !strings.Contains(value, "${") {
		return nil
	}

	// Parse expressions
	expressions, err := l.refParser.ParseAll(value)
	if err != nil {
		results.Results = append(results.Results, LintResult{
			Rule:       "expression-validation",
			Level:      "error",
			Message:    fmt.Sprintf("Invalid expression: %s", err),
			Path:       path,
			Suggestion: "Check expression syntax",
		})
		return nil
	}

	// Validate each expression
	for _, expr := range expressions {
		switch expr.Type {
		case ref.TypeConfigMap:
			// Check if ConfigMap exists
			if _, exists := config.Kubernetes.Resources.ConfigMaps[expr.Resource]; !exists {
				results.Results = append(results.Results, LintResult{
					Rule:       "expression-validation",
					Level:      "error",
					Message:    fmt.Sprintf("ConfigMap not found: %s", expr.Resource),
					Path:       path,
					Suggestion: fmt.Sprintf("Define ConfigMap '%s' in kubernetes.resources.configmaps", expr.Resource),
				})
			}
		case ref.TypeConfigMapValue:
			// Check if ConfigMap exists
			if _, exists := config.Kubernetes.Resources.ConfigMaps[expr.Resource]; !exists {
				results.Results = append(results.Results, LintResult{
					Rule:       "expression-validation",
					Level:      "error",
					Message:    fmt.Sprintf("ConfigMap not found: %s", expr.Resource),
					Path:       path,
					Suggestion: fmt.Sprintf("Define ConfigMap '%s' in kubernetes.resources.configmaps", expr.Resource),
				})
			}
		case ref.TypeSecret:
			// Check if Secret exists
			if _, exists := config.Kubernetes.Resources.Secrets[expr.Resource]; !exists {
				results.Results = append(results.Results, LintResult{
					Rule:       "expression-validation",
					Level:      "error",
					Message:    fmt.Sprintf("Secret not found: %s", expr.Resource),
					Path:       path,
					Suggestion: fmt.Sprintf("Define Secret '%s' in kubernetes.resources.secrets", expr.Resource),
				})
			}
		case ref.TypeService:
			// Check if Service exists
			if _, exists := config.Backend[expr.Resource]; !exists {
				if _, exists := config.Frontend[expr.Resource]; !exists {
					results.Results = append(results.Results, LintResult{
						Rule:       "expression-validation",
						Level:      "error",
						Message:    fmt.Sprintf("Service not found: %s", expr.Resource),
						Path:       path,
						Suggestion: fmt.Sprintf("Define service '%s' in backend or frontend", expr.Resource),
					})
				}
			}
		}
	}

	return nil
}

// Validation helper functions

func isValidServiceName(name string) bool {
	if name == "" || len(name) > 50 {
		return false
	}
	for _, r := range name {
		// Allow lowercase letters, numbers, hyphens, and underscores
		// Underscores are allowed for Dart/Flutter naming compatibility
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func isValidResourceName(name string) bool {
	return isValidServiceName(name)
}
