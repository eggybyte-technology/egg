// Package generators provides code generation for standalone services.
//
// Overview:
//   - Responsibility: Generate standalone service scaffolding
//   - Key Types: StandaloneGenerator for service creation
//   - Concurrency Model: Sequential generation with atomic file writes
//   - Error Semantics: Generation errors with rollback support
//   - Performance Notes: Template-based generation, minimal I/O operations
//
// Usage:
//
//	gen := NewStandaloneGenerator(fs, runner)
//	err := gen.Create(ctx, name, modulePath, protoTemplate, useLocalModules)
package generators

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"go.eggybyte.com/egg/cli/internal/configschema"
	"go.eggybyte.com/egg/cli/internal/projectfs"
	"go.eggybyte.com/egg/cli/internal/templates"
	"go.eggybyte.com/egg/cli/internal/toolrunner"
	"go.eggybyte.com/egg/cli/internal/ui"
)

// StandaloneGenerator provides standalone service generation.
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
type StandaloneGenerator struct {
	fs     *projectfs.ProjectFS
	runner *toolrunner.Runner
	loader *templates.Loader
}

// NewStandaloneGenerator creates a new standalone service generator.
//
// Parameters:
//   - fs: Project file system
//   - runner: Tool runner for external commands
//
// Returns:
//   - *StandaloneGenerator: Generator instance
//
// Concurrency:
//   - Safe to call from multiple goroutines
//
// Performance:
//   - Minimal initialization cost
func NewStandaloneGenerator(fs *projectfs.ProjectFS, runner *toolrunner.Runner) *StandaloneGenerator {
	return &StandaloneGenerator{
		fs:     fs,
		runner: runner,
		loader: templates.NewLoader(),
	}
}

// Create creates a new standalone service.
//
// Parameters:
//   - ctx: Context for cancellation
//   - name: Service name
//   - modulePath: Go module path (e.g., "github.com/org/my-service")
//   - protoTemplate: Proto template type (echo, crud, or none)
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
func (g *StandaloneGenerator) Create(ctx context.Context, name, modulePath string, protoTemplate string, useLocalModules bool) error {
	ui.Info("Creating standalone service: %s", name)

	// Validate service name
	if !isValidServiceName(name) {
		return fmt.Errorf("invalid service name: %s", name)
	}

	// Create service directory structure (same as user-service)
	dirs := []string{
		"api",
		"cmd/server",
		"gen/go",
		"internal/config",
		"internal/handler",
		"internal/model",
		"internal/repository",
		"internal/service",
	}

	for _, dir := range dirs {
		if err := g.fs.CreateDirectory(dir); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Initialize Go module
	serviceRunner := toolrunner.NewRunner(g.fs.GetRootDir())
	serviceRunner.SetVerbose(true)

	if err := serviceRunner.GoModInit(ctx, modulePath); err != nil {
		return fmt.Errorf("failed to initialize Go module: %w", err)
	}

	// Add egg dependencies (same logic as BackendGenerator)
	if err := g.addDependencies(ctx, serviceRunner, useLocalModules); err != nil {
		return fmt.Errorf("failed to add dependencies: %w", err)
	}

	// Create a dummy config for template data
	dummyConfig := &configschema.Config{
		ModulePrefix: modulePath,
		Backend: map[string]configschema.BackendService{
			name: {
				Ports: &configschema.PortConfig{
					HTTP:    8080,
					Health:  8081,
					Metrics: 9091,
				},
			},
		},
	}

	serviceConfig := dummyConfig.Backend[name]

	// Prepare template data
	templateData := g.prepareTemplateData(name, modulePath, serviceConfig, protoTemplate)

	// Generate API configuration files
	if err := g.generateAPIConfig(); err != nil {
		return fmt.Errorf("failed to generate API configuration: %w", err)
	}

	// Generate proto file if requested
	if protoTemplate != "none" {
		if err := g.generateProtoFile(name, protoTemplate, templateData); err != nil {
			return fmt.Errorf("failed to generate proto file: %w", err)
		}
	}

	// Generate service files
	if err := g.generateServiceFiles(name, templateData); err != nil {
		return fmt.Errorf("failed to generate service files: %w", err)
	}

	// Generate Dockerfile
	if err := g.generateDockerfile(name, modulePath); err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	// Generate .env.example
	if err := g.generateEnvExample(name); err != nil {
		return fmt.Errorf("failed to generate .env.example: %w", err)
	}

	// Generate .gitignore
	if err := g.generateGitignore(); err != nil {
		return fmt.Errorf("failed to generate .gitignore: %w", err)
	}

	// Generate .dockerignore
	if err := g.generateDockerignore(); err != nil {
		return fmt.Errorf("failed to generate .dockerignore: %w", err)
	}

	// Tidy module
	if useLocalModules {
		if _, err := serviceRunner.GoWithEnv(ctx, map[string]string{
			"GOPROXY": "direct",
			"GOSUMDB": "off",
		}, "mod", "tidy"); err != nil {
			ui.Warning("go mod tidy failed: %v", err)
			ui.Warning("This may indicate missing dependencies or import issues")
		}
	} else {
		if err := serviceRunner.GoModTidy(ctx); err != nil {
			ui.Warning("go mod tidy failed: %v", err)
			ui.Warning("This may indicate missing dependencies or import issues")
		}
	}

	ui.Success("Standalone service created: %s", name)
	ui.Info("Next steps:")
	ui.Info("  1. Define your API in api/%s/v1/%s.proto", name, name)
	ui.Info("  2. Generate code: buf generate")
	ui.Info("  3. Implement service logic in internal/")
	ui.Info("  4. Build: egg standalone build")
	ui.Info("  5. Run: egg standalone run")

	return nil
}

// addDependencies adds egg framework dependencies.
//
// Uses the same logic as BackendGenerator.Create() to ensure consistency.
func (g *StandaloneGenerator) addDependencies(ctx context.Context, runner *toolrunner.Runner, useLocalModules bool) error {
	if useLocalModules {
		// Use local dev versions for development
		ui.Info("Adding local egg modules for development...")

		requiredDeps := []string{
			"go.eggybyte.com/egg/core@v0.0.0-dev",
			"go.eggybyte.com/egg/logx@v0.0.0-dev",
			"go.eggybyte.com/egg/configx@v0.0.0-dev",
			"go.eggybyte.com/egg/obsx@v0.0.0-dev",
			"go.eggybyte.com/egg/connectx@v0.0.0-dev",
			"go.eggybyte.com/egg/runtimex@v0.0.0-dev",
			"go.eggybyte.com/egg/servicex@v0.0.0-dev",
			"go.eggybyte.com/egg/storex@v0.0.0-dev",
			"connectrpc.com/connect@latest",
			"gorm.io/gorm@latest",
			"gorm.io/driver/mysql@latest",
			"github.com/google/uuid@latest",
		}

		for _, dep := range requiredDeps {
			if _, err := runner.GoWithEnv(ctx, map[string]string{
				"GOPROXY": "direct",
				"GOSUMDB": "off",
			}, "get", dep); err != nil {
				ui.Warning("Failed to add dependency %s: %v", dep, err)
				continue
			}
		}
	} else {
		// Use published modules with framework version
		ui.Info("Using published egg modules...")

		frameworkVersion := getFrameworkVersion()
		if frameworkVersion == "latest" {
			ui.Warning("Using 'latest' framework version (development build). Published CLI releases will use specific versions.")
		}

		eggDeps := []string{
			fmt.Sprintf("go.eggybyte.com/egg/core@%s", frameworkVersion),
			fmt.Sprintf("go.eggybyte.com/egg/logx@%s", frameworkVersion),
			fmt.Sprintf("go.eggybyte.com/egg/configx@%s", frameworkVersion),
			fmt.Sprintf("go.eggybyte.com/egg/obsx@%s", frameworkVersion),
			fmt.Sprintf("go.eggybyte.com/egg/connectx@%s", frameworkVersion),
			fmt.Sprintf("go.eggybyte.com/egg/runtimex@%s", frameworkVersion),
			fmt.Sprintf("go.eggybyte.com/egg/servicex@%s", frameworkVersion),
			fmt.Sprintf("go.eggybyte.com/egg/storex@%s", frameworkVersion),
		}
		thirdPartyDeps := []string{
			"connectrpc.com/connect@latest",
			"gorm.io/gorm@latest",
			"gorm.io/driver/mysql@latest",
			"github.com/google/uuid@latest",
		}

		ui.Info("Adding egg framework dependencies (GOPROXY=https://goproxy.cn,direct)...")
		for _, dep := range eggDeps {
			if _, err := runner.GoWithEnv(ctx, map[string]string{"GOPROXY": "https://goproxy.cn,direct"}, "get", dep); err != nil {
				ui.Warning("Failed to add dependency %s: %v", dep, err)
				continue
			}
		}

		ui.Info("Adding third-party dependencies...")
		for _, dep := range thirdPartyDeps {
			if _, err := runner.Go(ctx, "get", dep); err != nil {
				ui.Warning("Failed to add dependency %s: %v", dep, err)
				continue
			}
		}
	}

	return nil
}

// prepareTemplateData prepares data for template rendering.
func (g *StandaloneGenerator) prepareTemplateData(name, modulePath string, serviceConfig configschema.BackendService, protoTemplate string) *TemplateData {
	serviceName := strings.ToLower(name)
	// Convert service-name to ServiceName (camel case)
	serviceNameCamel := ""
	parts := strings.Split(serviceName, "-")
	for _, part := range parts {
		if len(part) > 0 {
			serviceNameCamel += strings.ToUpper(part[:1]) + part[1:]
		}
	}
	// Convert service-name to serviceName (lower camel case)
	serviceNameVar := serviceName
	if len(parts) > 1 {
		serviceNameVar = parts[0]
		for i := 1; i < len(parts); i++ {
			if len(parts[i]) > 0 {
				serviceNameVar += strings.ToUpper(parts[i][:1]) + parts[i][1:]
			}
		}
	}

	// Extract organization and project from module path for proto package
	// e.g., "github.com/org/project" -> "org.project"
	protoPackage := extractProtoPackage(modulePath)

	// Determine if database is needed based on proto template
	hasDatabase := protoTemplate == "crud"

	return &TemplateData{
		ModulePrefix:      modulePath,
		ServiceModulePath: modulePath,
		ServiceName:       serviceName,
		ServiceNameCamel:  serviceNameCamel,
		ServiceNameVar:    serviceNameVar,
		ProtoPackage:      protoPackage,
		BinaryName:        "server",
		HTTPPort:          serviceConfig.Ports.HTTP,
		HealthPort:        serviceConfig.Ports.Health,
		MetricsPort:       serviceConfig.Ports.Metrics,
		ProtoTemplate:     protoTemplate,
		HasDatabase:       hasDatabase,
	}
}

// generateAPIConfig generates buf.yaml and buf.gen.yaml.
func (g *StandaloneGenerator) generateAPIConfig() error {
	ui.Info("Generating API configuration...")

	// Generate buf.yaml
	bufYAML, err := g.loader.LoadTemplate("api/buf.yaml.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load buf.yaml template: %w", err)
	}
	if err := g.fs.WriteFile("api/buf.yaml", bufYAML, 0644); err != nil {
		return fmt.Errorf("failed to write buf.yaml: %w", err)
	}

	// Generate buf.gen.yaml
	bufGenYAML, err := g.loader.LoadTemplate("api/buf.gen.yaml.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load buf.gen.yaml template: %w", err)
	}
	if err := g.fs.WriteFile("api/buf.gen.yaml", bufGenYAML, 0644); err != nil {
		return fmt.Errorf("failed to write buf.gen.yaml: %w", err)
	}

	ui.Success("API configuration generated")
	return nil
}

// generateProtoFile generates proto file from template.
func (g *StandaloneGenerator) generateProtoFile(name, protoTemplate string, templateData *TemplateData) error {
	ui.Info("Generating proto file from %s template...", protoTemplate)

	var protoContent string
	var err error

	switch protoTemplate {
	case "echo":
		protoContent, err = g.loader.LoadAndRender("api/proto_echo.tmpl", templateData)
	case "crud":
		protoContent, err = g.loader.LoadAndRender("api/proto_crud.tmpl", templateData)
	default:
		return fmt.Errorf("unsupported proto template: %s", protoTemplate)
	}

	if err != nil {
		return fmt.Errorf("failed to render proto template: %w", err)
	}

	protoDir := filepath.Join("api", name, "v1")
	if err := g.fs.CreateDirectory(protoDir); err != nil {
		return fmt.Errorf("failed to create proto directory: %w", err)
	}

	protoFile := filepath.Join(protoDir, name+".proto")
	if err := g.fs.WriteFile(protoFile, protoContent, 0644); err != nil {
		return fmt.Errorf("failed to write proto file: %w", err)
	}

	ui.Success("Proto file generated: %s", protoFile)
	return nil
}

// generateServiceFiles generates all service code files.
func (g *StandaloneGenerator) generateServiceFiles(name string, templateData *TemplateData) error {
	ui.Info("Generating service files...")

	files := []struct {
		template string
		output   string
	}{
		{"backend/main.go.tmpl", "cmd/server/main.go"},
		{"backend/app_config.go.tmpl", "internal/config/app_config.go"},
		{"backend/handler.go.tmpl", fmt.Sprintf("internal/handler/%s_handler.go", name)},
		{"backend/service.go.tmpl", fmt.Sprintf("internal/service/%s_service.go", name)},
		{"backend/errors.go.tmpl", "internal/model/errors.go"},
	}

	// Add repository and model files for crud template
	if templateData.ProtoTemplate == "crud" {
		files = append(files,
			struct {
				template string
				output   string
			}{"backend/repository.go.tmpl", fmt.Sprintf("internal/repository/%s_repository.go", name)},
			struct {
				template string
				output   string
			}{"backend/model.go.tmpl", fmt.Sprintf("internal/model/%s.go", name)},
		)
	}

	for _, file := range files {
		content, err := g.loader.LoadAndRender(file.template, templateData)
		if err != nil {
			return fmt.Errorf("failed to render %s: %w", file.template, err)
		}

		if err := g.fs.WriteFile(file.output, content, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", file.output, err)
		}
	}

	ui.Success("Service files generated")
	return nil
}

// generateDockerfile generates Dockerfile for standalone service.
func (g *StandaloneGenerator) generateDockerfile(name, modulePath string) error {
	ui.Info("Generating Dockerfile...")

	data := map[string]interface{}{
		"ServiceName": name,
		"ModulePath":  modulePath,
	}

	content, err := g.loader.LoadAndRender("standalone/Dockerfile.tmpl", data)
	if err != nil {
		return fmt.Errorf("failed to render Dockerfile template: %w", err)
	}

	if err := g.fs.WriteFile("Dockerfile", content, 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	ui.Success("Dockerfile generated")
	return nil
}

// generateEnvExample generates .env.example file.
func (g *StandaloneGenerator) generateEnvExample(name string) error {
	ui.Info("Generating .env.example...")

	data := map[string]interface{}{
		"ServiceName": name,
	}

	content, err := g.loader.LoadAndRender("standalone/env.example.tmpl", data)
	if err != nil {
		return fmt.Errorf("failed to render .env.example template: %w", err)
	}

	if err := g.fs.WriteFile(".env.example", content, 0644); err != nil {
		return fmt.Errorf("failed to write .env.example: %w", err)
	}

	ui.Success(".env.example generated")
	return nil
}

// generateGitignore generates .gitignore file.
func (g *StandaloneGenerator) generateGitignore() error {
	ui.Info("Generating .gitignore...")

	content, err := g.loader.LoadTemplate(".gitignore.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load .gitignore template: %w", err)
	}

	if err := g.fs.WriteFile(".gitignore", content, 0644); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	ui.Success(".gitignore generated")
	return nil
}

// generateDockerignore generates .dockerignore file.
func (g *StandaloneGenerator) generateDockerignore() error {
	ui.Info("Generating .dockerignore...")

	content, err := g.loader.LoadTemplate(".dockerignore.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load .dockerignore template: %w", err)
	}

	if err := g.fs.WriteFile(".dockerignore", content, 0644); err != nil {
		return fmt.Errorf("failed to write .dockerignore: %w", err)
	}

	ui.Success(".dockerignore generated")
	return nil
}

// Helper functions

// extractProtoPackage extracts proto package from module path.
// Example: "github.com/org/project" -> "org.project"
func extractProtoPackage(modulePath string) string {
	parts := strings.Split(modulePath, "/")
	if len(parts) >= 3 {
		// github.com/org/project -> org.project
		return strings.Join(parts[1:], ".")
	}
	// Fallback to just the last part
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "default"
}

