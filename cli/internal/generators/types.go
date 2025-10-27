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

// TemplateData holds data for rendering all backend service templates.
//
// This structure provides all necessary data for template rendering including
// naming conventions, module paths, and configuration values.
//
// Parameters:
//   - ModulePrefix: Go module prefix (e.g., "github.com/org/project")
//   - ServiceName: Service name in lowercase (e.g., "user")
//   - ServiceNameCamel: Service name in CamelCase (e.g., "User")
//   - ServiceNameVar: Service name as variable (e.g., "user")
//   - ProtoPackage: Proto package prefix (e.g., "org.project")
//   - BinaryName: Binary name (default: "server")
//   - HTTPPort: HTTP port (default: 8080)
//   - HealthPort: Health port (default: 8081)
//   - MetricsPort: Metrics port (default: 9091)
//
// Usage:
//
//	data := &TemplateData{
//	    ModulePrefix:     "github.com/org/project",
//	    ServiceName:      "user",
//	    ServiceNameCamel: "User",
//	    ServiceNameVar:   "user",
//	    ProtoPackage:     "org.project",
//	    BinaryName:       "server",
//	    HTTPPort:         8080,
//	    HealthPort:       8081,
//	    MetricsPort:      9091,
//	}
//
// Concurrency:
//
//	Safe for concurrent read access after initialization.
type TemplateData struct {
	ModulePrefix     string // Go module prefix
	ServiceName      string // Service name (lowercase, e.g., "user")
	ServiceNameCamel string // Service name (CamelCase, e.g., "User")
	ServiceNameVar   string // Service name for variables (e.g., "user")
	ProtoPackage     string // Proto package prefix (e.g., "org.project")
	BinaryName       string // Binary name (default: "server")
	HTTPPort         int    // HTTP port (default: 8080)
	HealthPort       int    // Health port (default: 8081)
	MetricsPort      int    // Metrics port (default: 9091)
}

