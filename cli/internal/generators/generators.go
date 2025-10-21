// Package generators provides code generation for API, backend, and frontend services.
//
// Overview:
//   - Responsibility: Generate project scaffolding, API definitions, service templates
//   - Key Types: API generator, backend generator, frontend generator
//   - Concurrency Model: Sequential generation with atomic file writes
//   - Error Semantics: Generation errors with rollback support
//   - Performance Notes: Template-based generation, minimal I/O operations
//
// Usage:
//
//	apiGen := NewAPIGenerator(fs, runner)
//	err := apiGen.Init()
//	err := apiGen.Generate()
package generators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eggybyte-technology/egg/cli/internal/configschema"
	"github.com/eggybyte-technology/egg/cli/internal/projectfs"
	"github.com/eggybyte-technology/egg/cli/internal/templates"
	"github.com/eggybyte-technology/egg/cli/internal/toolrunner"
	"github.com/eggybyte-technology/egg/cli/internal/ui"
)

// APIGenerator provides API definition generation.
//
// Parameters:
//   - fs: Project file system
//   - runner: Tool runner for external commands
//   - loader: Template loader
//
// Returns:
//   - None (data structure)
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Template-based generation
type APIGenerator struct {
	fs     *projectfs.ProjectFS
	runner *toolrunner.Runner
	loader *templates.Loader
}

// BackendGenerator provides backend service generation.
//
// Parameters:
//   - fs: Project file system
//   - runner: Tool runner for external commands
//   - loader: Template loader
//
// Returns:
//   - None (data structure)
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Template-based generation
type BackendGenerator struct {
	fs     *projectfs.ProjectFS
	runner *toolrunner.Runner
	loader *templates.Loader
}

// FrontendGenerator provides frontend service generation.
//
// Parameters:
//   - fs: Project file system
//   - runner: Tool runner for external commands
//
// Returns:
//   - None (data structure)
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Template-based generation
type FrontendGenerator struct {
	fs     *projectfs.ProjectFS
	runner *toolrunner.Runner
}

// NewAPIGenerator creates a new API generator.
//
// Parameters:
//   - fs: Project file system
//   - runner: Tool runner for external commands
//
// Returns:
//   - *APIGenerator: API generator instance
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Minimal initialization overhead
func NewAPIGenerator(fs *projectfs.ProjectFS, runner *toolrunner.Runner) *APIGenerator {
	return &APIGenerator{
		fs:     fs,
		runner: runner,
		loader: templates.NewLoader(),
	}
}

// NewBackendGenerator creates a new backend generator.
//
// Parameters:
//   - fs: Project file system
//   - runner: Tool runner for external commands
//
// Returns:
//   - *BackendGenerator: Backend generator instance
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Minimal initialization overhead
func NewBackendGenerator(fs *projectfs.ProjectFS, runner *toolrunner.Runner) *BackendGenerator {
	return &BackendGenerator{
		fs:     fs,
		runner: runner,
		loader: templates.NewLoader(),
	}
}

// NewFrontendGenerator creates a new frontend generator.
//
// Parameters:
//   - fs: Project file system
//   - runner: Tool runner for external commands
//
// Returns:
//   - *FrontendGenerator: Frontend generator instance
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Minimal initialization overhead
func NewFrontendGenerator(fs *projectfs.ProjectFS, runner *toolrunner.Runner) *FrontendGenerator {
	return &FrontendGenerator{
		fs:     fs,
		runner: runner,
	}
}

// Init initializes API definitions and configuration.
//
// Parameters:
//   - ctx: Context for cancellation
//
// Returns:
//   - error: Initialization error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - File creation and tool execution
func (g *APIGenerator) Init(ctx context.Context) error {
	ui.Info("Initializing API definitions...")

	// Create API directory structure
	if err := g.fs.CreateDirectory("api"); err != nil {
		return fmt.Errorf("failed to create api directory: %w", err)
	}

	// Write buf.yaml from template
	bufYAML, err := g.loader.LoadTemplate("api/buf.yaml.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load buf.yaml template: %w", err)
	}
	if err := g.fs.WriteFile("api/buf.yaml", bufYAML, 0644); err != nil {
		return fmt.Errorf("failed to write buf.yaml: %w", err)
	}

	// Write buf.gen.yaml from template
	bufGenYAML, err := g.loader.LoadTemplate("api/buf.gen.yaml.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load buf.gen.yaml template: %w", err)
	}
	if err := g.fs.WriteFile("api/buf.gen.yaml", bufGenYAML, 0644); err != nil {
		return fmt.Errorf("failed to write buf.gen.yaml: %w", err)
	}

	// Note: gen directories will be created automatically by buf generate
	ui.Success("API definitions initialized")
	return nil
}

// Generate generates code from API definitions.
//
// Parameters:
//   - ctx: Context for cancellation
//
// Returns:
//   - error: Generation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Code generation time depends on protobuf complexity
func (g *APIGenerator) Generate(ctx context.Context) error {
	ui.Info("Generating code from API definitions...")

	// Change to api directory
	apiRunner := toolrunner.NewRunner(filepath.Join(g.fs.GetRootDir(), "api"))
	apiRunner.SetVerbose(true)

	// Run buf generate
	if err := apiRunner.BufGenerate(ctx); err != nil {
		return fmt.Errorf("failed to generate code with buf: %w", err)
	}

	ui.Success("Code generation completed")
	return nil
}

// Create creates a new backend service.
//
// Parameters:
//   - ctx: Context for cancellation
//   - name: Service name
//   - config: Project configuration
//   - useLocalModules: Whether to use local egg modules for development
//
// Returns:
//   - error: Creation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Service scaffolding and module initialization
func (g *BackendGenerator) Create(ctx context.Context, name string, config *configschema.Config, useLocalModules bool) error {
	ui.Info("Creating backend service: %s", name)

	// Validate service name
	if !isValidServiceName(name) {
		return fmt.Errorf("invalid service name: %s", name)
	}

	// Create service directory structure
	serviceDir := filepath.Join("backend", name)
	if err := g.fs.CreateDirectory(serviceDir); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	// Create cmd/server directory
	cmdDir := filepath.Join(serviceDir, "cmd", "server")
	if err := g.fs.CreateDirectory(cmdDir); err != nil {
		return fmt.Errorf("failed to create cmd directory: %w", err)
	}

	// Create internal directory structure
	internalDirs := []string{
		filepath.Join(serviceDir, "internal", "config"),
		filepath.Join(serviceDir, "internal", "handler"),
		filepath.Join(serviceDir, "internal", "model"),
		filepath.Join(serviceDir, "internal", "repository"),
		filepath.Join(serviceDir, "internal", "service"),
	}
	for _, dir := range internalDirs {
		if err := g.fs.CreateDirectory(dir); err != nil {
			return fmt.Errorf("failed to create internal directory %s: %w", dir, err)
		}
	}

	// Get service configuration
	serviceConfig, exists := config.Backend[name]
	if !exists {
		return fmt.Errorf("service configuration not found: %s", name)
	}

	// Initialize Go module
	modulePath := fmt.Sprintf("%s/backend/%s", config.ModulePrefix, name)
	serviceRunner := toolrunner.NewRunner(filepath.Join(g.fs.GetRootDir(), serviceDir))
	serviceRunner.SetVerbose(true)

	if err := serviceRunner.GoModInit(ctx, modulePath); err != nil {
		return fmt.Errorf("failed to initialize Go module: %w", err)
	}

	// Add egg dependencies
	if useLocalModules {
		// Use local modules for development
		ui.Info("Adding local egg modules for development...")

		// Get the egg project root (assuming we're running from within the egg project)
		eggRoot, err := g.findEggProjectRoot()
		if err != nil {
			return fmt.Errorf("failed to find egg project root: %w", err)
		}

		// Add replace directives to use local modules directly
		replaceDeps := map[string]string{
			"github.com/eggybyte-technology/egg/bootstrap": filepath.Join(eggRoot, "bootstrap"),
			"github.com/eggybyte-technology/egg/runtimex":  filepath.Join(eggRoot, "runtimex"),
			"github.com/eggybyte-technology/egg/connectx":  filepath.Join(eggRoot, "connectx"),
			"github.com/eggybyte-technology/egg/configx":   filepath.Join(eggRoot, "configx"),
			"github.com/eggybyte-technology/egg/obsx":      filepath.Join(eggRoot, "obsx"),
			"github.com/eggybyte-technology/egg/core":      filepath.Join(eggRoot, "core"),
			"github.com/eggybyte-technology/egg/storex":    filepath.Join(eggRoot, "storex"),
		}

		for modulePath, localPath := range replaceDeps {
			if _, err := serviceRunner.Go(ctx, "mod", "edit", "-replace", fmt.Sprintf("%s=%s", modulePath, localPath)); err != nil {
				return fmt.Errorf("failed to add replace directive for %s: %w", modulePath, err)
			}
		}

		// Add required dependencies explicitly
		requiredDeps := []string{
			"github.com/eggybyte-technology/egg/bootstrap@latest",
			"github.com/eggybyte-technology/egg/runtimex@latest",
			"github.com/eggybyte-technology/egg/connectx@latest",
			"github.com/eggybyte-technology/egg/configx@latest",
			"github.com/eggybyte-technology/egg/obsx@latest",
			"github.com/eggybyte-technology/egg/core@latest",
			"github.com/eggybyte-technology/egg/storex@latest",
			"connectrpc.com/connect@latest",
			"gorm.io/gorm@latest",
			"gorm.io/driver/mysql@latest",
			"github.com/google/uuid@latest",
		}
		for _, dep := range requiredDeps {
			if _, err := serviceRunner.Go(ctx, "get", dep); err != nil {
				return fmt.Errorf("failed to add dependency %s: %w", dep, err)
			}
		}
	} else {
		// Use published modules (when available)
		ui.Info("Using published egg modules...")
		eggDeps := []string{
			"github.com/eggybyte-technology/egg/bootstrap@latest",
			"github.com/eggybyte-technology/egg/runtimex@latest",
			"github.com/eggybyte-technology/egg/connectx@latest",
			"github.com/eggybyte-technology/egg/configx@latest",
			"github.com/eggybyte-technology/egg/obsx@latest",
			"github.com/eggybyte-technology/egg/core@latest",
			"github.com/eggybyte-technology/egg/storex@latest",
			"connectrpc.com/connect@latest",
			"gorm.io/gorm@latest",
			"gorm.io/driver/mysql@latest",
			"github.com/google/uuid@latest",
		}
		for _, dep := range eggDeps {
			if _, err := serviceRunner.Go(ctx, "get", dep); err != nil {
				return fmt.Errorf("failed to add dependency %s: %w", dep, err)
			}
		}
	}

	// Generate service files first
	if err := g.generateBackendFiles(name, serviceConfig, config); err != nil {
		return fmt.Errorf("failed to generate service files: %w", err)
	}

	// Tidy module after generating files with imports
	if err := serviceRunner.GoModTidy(ctx); err != nil {
		return fmt.Errorf("failed to tidy module: %w", err)
	}

	// Update backend go.work using go commands
	if err := g.updateBackendWorkspace(ctx, name, config); err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	ui.Success("Backend service created: %s", name)
	return nil
}

// generateBackendFiles generates the backend service files.
//
// Parameters:
//   - name: Service name
//   - serviceConfig: Service configuration
//   - config: Project configuration
//
// Returns:
//   - error: Generation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template-based file generation
func (g *BackendGenerator) generateBackendFiles(name string, serviceConfig configschema.BackendService, config *configschema.Config) error {
	data := map[string]interface{}{
		"ModulePrefix":     config.ModulePrefix,
		"ServiceName":      name,
		"ServiceNameCamel": camelCaseServiceName(name),
		"ServiceNameVar":   serviceNameToVar(name),
	}

	// Generate main.go from template
	mainGo, err := g.loader.LoadAndRender("backend/main.go.tmpl", data)
	if err != nil {
		return fmt.Errorf("failed to load and render main.go template: %w", err)
	}
	if err := g.fs.WriteFile(filepath.Join("backend", name, "cmd", "server", "main.go"), mainGo, 0644); err != nil {
		return fmt.Errorf("failed to write main.go: %w", err)
	}

	// Generate config/app_config.go from template
	appConfigGo, err := g.loader.LoadAndRender("backend/app_config.go.tmpl", data)
	if err != nil {
		return fmt.Errorf("failed to load and render app_config.go template: %w", err)
	}
	if err := g.fs.WriteFile(filepath.Join("backend", name, "internal", "config", "app_config.go"), appConfigGo, 0644); err != nil {
		return fmt.Errorf("failed to write app_config.go: %w", err)
	}

	// Generate service placeholder from template
	serviceGo, err := g.loader.LoadAndRender("backend/service.go.tmpl", data)
	if err != nil {
		return fmt.Errorf("failed to load and render service.go template: %w", err)
	}
	if err := g.fs.WriteFile(filepath.Join("backend", name, "internal", "service", "service.go"), serviceGo, 0644); err != nil {
		return fmt.Errorf("failed to write service.go: %w", err)
	}

	// Generate handler placeholder from template
	handlerGo, err := g.loader.LoadAndRender("backend/handler.go.tmpl", data)
	if err != nil {
		return fmt.Errorf("failed to load and render handler.go template: %w", err)
	}
	if err := g.fs.WriteFile(filepath.Join("backend", name, "internal", "handler", "handler.go"), handlerGo, 0644); err != nil {
		return fmt.Errorf("failed to write handler.go: %w", err)
	}

	// Generate model from template
	modelGo, err := g.loader.LoadAndRender("backend/model.go.tmpl", data)
	if err != nil {
		return fmt.Errorf("failed to load and render model.go template: %w", err)
	}
	if err := g.fs.WriteFile(filepath.Join("backend", name, "internal", "model", "model.go"), modelGo, 0644); err != nil {
		return fmt.Errorf("failed to write model.go: %w", err)
	}

	// Generate repository from template
	repositoryGo, err := g.loader.LoadAndRender("backend/repository.go.tmpl", data)
	if err != nil {
		return fmt.Errorf("failed to load and render repository.go template: %w", err)
	}
	if err := g.fs.WriteFile(filepath.Join("backend", name, "internal", "repository", "repository.go"), repositoryGo, 0644); err != nil {
		return fmt.Errorf("failed to write repository.go: %w", err)
	}

	return nil
}

// updateBackendWorkspace updates the backend go.work file.
//
// Parameters:
//   - ctx: Context for cancellation
//   - name: Service name
//   - config: Project configuration
//
// Returns:
//   - error: Update error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Workspace file update
func (g *BackendGenerator) updateBackendWorkspace(ctx context.Context, name string, config *configschema.Config) error {
	// Check if go.work exists
	workPath := filepath.Join("backend", "go.work")
	exists, err := g.fs.FileExists(workPath)
	if err != nil {
		return fmt.Errorf("failed to check go.work existence: %w", err)
	}

	backendRunner := toolrunner.NewRunner(filepath.Join(g.fs.GetRootDir(), "backend"))
	backendRunner.SetVerbose(true)

	if !exists {
		// Create new workspace in backend directory
		if err := backendRunner.GoWorkInit(ctx, fmt.Sprintf("./%s", name)); err != nil {
			return fmt.Errorf("failed to initialize workspace: %w", err)
		}
	} else {
		// Add to existing workspace
		if err := backendRunner.GoWorkUse(ctx, fmt.Sprintf("./%s", name)); err != nil {
			return fmt.Errorf("failed to add module to workspace: %w", err)
		}
	}

	return nil
}

// Create creates a new frontend service.
//
// Parameters:
//   - ctx: Context for cancellation
//   - name: Service name
//   - platforms: Target platforms (web, android, ios)
//   - config: Project configuration
//
// Returns:
//   - error: Creation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Flutter project creation
func (g *FrontendGenerator) Create(ctx context.Context, name string, platforms []string, config *configschema.Config) error {
	ui.Info("Creating frontend service: %s (platforms: %v)", name, platforms)

	// Validate service name
	if !isValidServiceName(name) {
		return fmt.Errorf("invalid service name: %s", name)
	}

	// Convert service name to valid Dart package name
	// Dart requires lowercase letters, numbers, and underscores only
	dartPackageName := dartifyServiceName(name)
	if dartPackageName != name {
		ui.Info("Converting service name to Dart-compatible package name: %s -> %s", name, dartPackageName)
	}

	// Validate Dart package name
	if !isValidDartPackageName(dartPackageName) {
		return fmt.Errorf("invalid Dart package name: %s (must use lowercase letters, numbers, and underscores only)", dartPackageName)
	}

	// Create Flutter project
	frontendRunner := toolrunner.NewRunner(filepath.Join(g.fs.GetRootDir(), "frontend"))
	frontendRunner.SetVerbose(true)

	if err := frontendRunner.FlutterCreate(ctx, dartPackageName, platforms); err != nil {
		return fmt.Errorf("failed to create Flutter project: %w", err)
	}

	ui.Success("Frontend service created: %s", dartPackageName)
	ui.Info("To use a specific Flutter version, use FVM: https://fvm.app")
	return nil
}

// GenerateCompose generates docker-compose.yaml for the project.
//
// Parameters:
//   - ctx: Context for cancellation
//   - config: Project configuration
//
// Returns:
//   - error: Generation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template-based file generation
func (g *BackendGenerator) GenerateCompose(ctx context.Context, config *configschema.Config) error {
	ui.Info("Generating docker-compose.yaml...")

	data := map[string]interface{}{
		"ProjectName":     config.ProjectName,
		"BackendServices": config.Backend,
	}

	// Generate docker-compose.yaml from template
	composeYAML, err := g.loader.LoadAndRender("compose/docker-compose.yaml.tmpl", data)
	if err != nil {
		return fmt.Errorf("failed to load and render docker-compose.yaml template: %w", err)
	}
	if err := g.fs.WriteFile("docker-compose.yaml", composeYAML, 0644); err != nil {
		return fmt.Errorf("failed to write docker-compose.yaml: %w", err)
	}

	// Generate Dockerfile.backend from template
	dockerfileBackend, err := g.loader.LoadAndRender("build/Dockerfile.backend.tmpl", data)
	if err != nil {
		return fmt.Errorf("failed to load and render Dockerfile.backend template: %w", err)
	}
	if err := g.fs.WriteFile("build/Dockerfile.backend", dockerfileBackend, 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile.backend: %w", err)
	}

	ui.Success("docker-compose.yaml generated")
	return nil
}

// findEggProjectRoot finds the root directory of the egg project.
//
// Parameters:
//   - None
//
// Returns:
//   - string: Path to egg project root
//   - error: Error if egg project root not found
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Directory traversal up to project root
func (g *BackendGenerator) findEggProjectRoot() (string, error) {
	currentDir := g.fs.GetRootDir()

	// Get absolute path
	absDir, err := filepath.Abs(currentDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}
	currentDir = absDir

	if g.fs.GetVerbose() {
		ui.Debug("Searching for egg project root starting from: %s", currentDir)
	}

	// Look for go.work file which indicates the egg project root
	for {
		if g.fs.GetVerbose() {
			ui.Debug("Checking directory: %s", currentDir)
		}

		// Use os.Stat directly since we're checking absolute paths
		goWorkPath := filepath.Join(currentDir, "go.work")
		if _, err := os.Stat(goWorkPath); err == nil {
			if g.fs.GetVerbose() {
				ui.Debug("Found go.work at: %s", goWorkPath)
			}
			// Also check if this directory contains egg modules
			eggModules := []string{"runtimex", "connectx", "configx", "obsx", "cli"}
			hasEggModules := true
			for _, module := range eggModules {
				modulePath := filepath.Join(currentDir, module)
				if _, err := os.Stat(modulePath); err != nil {
					if g.fs.GetVerbose() {
						ui.Debug("Missing egg module: %s", modulePath)
					}
					hasEggModules = false
					break
				} else if g.fs.GetVerbose() {
					ui.Debug("Found egg module: %s", modulePath)
				}
			}
			if hasEggModules {
				if g.fs.GetVerbose() {
					ui.Debug("Found egg project root: %s", currentDir)
				}
				return currentDir, nil
			}
		}

		// Check if we've reached the filesystem root
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			if g.fs.GetVerbose() {
				ui.Debug("Reached filesystem root: %s", currentDir)
			}
			break
		}
		currentDir = parent
	}

	return "", fmt.Errorf("egg project root not found (looking for go.work file and egg modules)")
}

// Validation helper functions

func isValidServiceName(name string) bool {
	if name == "" || len(name) > 50 {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// dartifyServiceName converts a service name to a valid Dart package name.
// It replaces hyphens with underscores and ensures the name follows Dart naming conventions.
//
// Parameters:
//   - name: Service name to convert
//
// Returns:
//   - string: Valid Dart package name
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - O(n) string conversion
func dartifyServiceName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

// isValidDartPackageName checks if a name is a valid Dart package name.
// Dart package names must:
// - Use only lowercase letters, numbers, and underscores
// - Not start with a number
// - Not be a reserved Dart keyword
//
// Parameters:
//   - name: Package name to validate
//
// Returns:
//   - bool: True if valid Dart package name
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - O(n) validation
func isValidDartPackageName(name string) bool {
	if name == "" || len(name) > 50 {
		return false
	}

	// Check if starts with a number
	if name[0] >= '0' && name[0] <= '9' {
		return false
	}

	// Check if all characters are valid
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}

	// Check for Dart reserved keywords
	dartReservedKeywords := []string{
		"abstract", "as", "assert", "async", "await", "break", "case", "catch",
		"class", "const", "continue", "covariant", "default", "deferred", "do",
		"dynamic", "else", "enum", "export", "extends", "extension", "external",
		"factory", "false", "final", "finally", "for", "function", "get", "hide",
		"if", "implements", "import", "in", "interface", "is", "late", "library",
		"mixin", "new", "null", "on", "operator", "part", "required", "rethrow",
		"return", "set", "show", "static", "super", "switch", "sync", "this",
		"throw", "true", "try", "typedef", "var", "void", "while", "with", "yield",
	}

	for _, keyword := range dartReservedKeywords {
		if name == keyword {
			return false
		}
	}

	return true
}

// camelCaseServiceName converts a service name to CamelCase for Go identifiers.
// It replaces hyphens and underscores with camelCase boundaries and capitalizes the first letter.
//
// Parameters:
//   - name: Service name to convert
//
// Returns:
//   - string: CamelCase service name
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - O(n) string conversion
func camelCaseServiceName(name string) string {
	if name == "" {
		return ""
	}

	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_'
	})

	if len(parts) == 0 {
		return ""
	}

	// Capitalize the first part
	result := strings.Title(strings.ToLower(parts[0]))
	for _, part := range parts[1:] {
		if part != "" {
			result += strings.Title(strings.ToLower(part))
		}
	}

	return result
}

// serviceNameToVar converts a service name to a valid Go variable name.
// It replaces hyphens and underscores with underscores and ensures the name is valid.
//
// Parameters:
//   - name: Service name to convert
//
// Returns:
//   - string: Valid Go variable name
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - O(n) string conversion
func serviceNameToVar(name string) string {
	if name == "" {
		return ""
	}

	// Replace hyphens and underscores with underscores
	result := strings.ReplaceAll(name, "-", "_")
	result = strings.ReplaceAll(result, "_", "_")

	// Ensure it starts with a letter
	if len(result) > 0 && (result[0] < 'a' || result[0] > 'z') {
		result = "service_" + result
	}

	return result
}
