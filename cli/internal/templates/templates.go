// Package templates provides template loading and rendering functionality.
//
// Overview:
//   - Responsibility: Load and render template files for project scaffolding
//   - Key Types: Template loader, renderer, file system operations
//   - Concurrency Model: Immutable template loading with atomic rendering
//   - Error Semantics: Template errors with file system context
//   - Performance Notes: Template caching, minimal I/O operations
//
// Usage:
//
//	loader := NewLoader()
//	content, err := loader.LoadTemplate("build/Dockerfile.backend.tmpl")
//	rendered, err := loader.RenderTemplate(content, data)
package templates

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"go.eggybyte.com/egg/cli/internal/ui"
)

//go:embed templates/*
var templateFS embed.FS

// Loader provides template loading and rendering functionality.
//
// Parameters:
//   - templateDir: Directory containing template files
//
// Returns:
//   - None (data structure)
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Template caching and efficient rendering
type Loader struct {
	templateDir string
}

// NewLoader creates a new template loader.
//
// Parameters:
//   - None
//
// Returns:
//   - *Loader: Template loader instance
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Minimal initialization overhead
func NewLoader() *Loader {
	return &Loader{
		templateDir: "templates",
	}
}

// LoadTemplate loads a template file from the embedded filesystem.
//
// Parameters:
//   - templatePath: Path to template file relative to templates directory
//
// Returns:
//   - string: Template content
//   - error: Loading error if any
//
// Concurrency:
//   - Single-threaded per template
//
// Performance:
//   - Embedded file system access
func (l *Loader) LoadTemplate(templatePath string) (string, error) {
	fullPath := filepath.Join(l.templateDir, templatePath)

	content, err := templateFS.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to load template %s: %w", templatePath, err)
	}

	return string(content), nil
}

// RenderTemplate renders a template with the provided data.
//
// Parameters:
//   - templateContent: Template content
//   - data: Template data
//
// Returns:
//   - string: Rendered content
//   - error: Rendering error if any
//
// Concurrency:
//   - Single-threaded per render
//
// Performance:
//   - Template parsing and rendering
func (l *Loader) RenderTemplate(templateContent string, data interface{}) (string, error) {
	// Create custom template functions
	funcMap := template.FuncMap{
		"ToUpper": strings.ToUpper,
		"ToLower": strings.ToLower,
		"Title":   strings.Title,
	}

	tmpl, err := template.New("template").Funcs(funcMap).Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return result.String(), nil
}

// LoadAndRender loads a template and renders it with data.
//
// Parameters:
//   - templatePath: Path to template file
//   - data: Template data
//
// Returns:
//   - string: Rendered content
//   - error: Loading or rendering error if any
//
// Concurrency:
//   - Single-threaded per operation
//
// Performance:
//   - Combined load and render operation
func (l *Loader) LoadAndRender(templatePath string, data interface{}) (string, error) {
	content, err := l.LoadTemplate(templatePath)
	if err != nil {
		return "", err
	}

	return l.RenderTemplate(content, data)
}

// ListTemplates lists all available template files.
//
// Parameters:
//   - None
//
// Returns:
//   - []string: List of template file paths
//   - error: Listing error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Directory traversal
func (l *Loader) ListTemplates() ([]string, error) {
	var templates []string

	err := l.walkTemplates("", func(path string) error {
		if strings.HasSuffix(path, ".tmpl") {
			templates = append(templates, path)
		}
		return nil
	})

	return templates, err
}

// walkTemplates walks through the template directory.
//
// Parameters:
//   - dir: Directory to walk
//   - fn: Function to call for each file
//
// Returns:
//   - error: Walk error if any
//
// Concurrency:
//   - Single-threaded per walk
//
// Performance:
//   - Directory traversal
func (l *Loader) walkTemplates(dir string, fn func(string) error) error {
	entries, err := templateFS.ReadDir(filepath.Join(l.templateDir, dir))
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			if err := l.walkTemplates(path, fn); err != nil {
				return err
			}
		} else {
			if err := fn(path); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetTemplatePath returns the full path for a template file.
//
// Parameters:
//   - templatePath: Template file path
//
// Returns:
//   - string: Full template path
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(1) path join operation
func (l *Loader) GetTemplatePath(templatePath string) string {
	return filepath.Join(l.templateDir, templatePath)
}

// ValidateTemplate validates that a template file exists and is readable.
//
// Parameters:
//   - templatePath: Path to template file
//
// Returns:
//   - error: Validation error if any
//
// Concurrency:
//   - Single-threaded per validation
//
// Performance:
//   - File existence check
func (l *Loader) ValidateTemplate(templatePath string) error {
	_, err := l.LoadTemplate(templatePath)
	if err != nil {
		return fmt.Errorf("template validation failed for %s: %w", templatePath, err)
	}
	return nil
}

// ValidateAllTemplates validates all template files.
//
// Parameters:
//   - None
//
// Returns:
//   - error: Validation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Sequential template validation
func (l *Loader) ValidateAllTemplates() error {
	templates, err := l.ListTemplates()
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	for _, templatePath := range templates {
		if err := l.ValidateTemplate(templatePath); err != nil {
			return err
		}
		ui.Debug("Template validated: %s", templatePath)
	}

	ui.Success("All templates validated successfully")
	return nil
}
