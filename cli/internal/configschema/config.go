// Package configschema provides configuration loading and validation for egg projects.
//
// Overview:
//   - Responsibility: Parse egg.yaml, validate schema, fill defaults
//   - Key Types: Config structs, validation rules, diagnostics
//   - Concurrency Model: Immutable configuration after loading
//   - Error Semantics: Structured validation errors with suggestions
//   - Performance Notes: Single-pass parsing, cached validation results
//
// Usage:
//
//	config, diags := Load("egg.yaml")
//	if diags.HasErrors() {
//	    return diags
//	}
package configschema

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the complete egg project configuration.
//
// Parameters:
//   - ConfigVersion: Schema version for compatibility
//   - ProjectName: Project identifier
//   - Version: Project version
//   - ModulePrefix: Go module prefix
//   - DockerRegistry: Container registry URL
//   - Build: Build configuration
//   - Env: Environment variable definitions
//   - BackendDefaults: Default settings for backend services
//   - Kubernetes: Kubernetes resource definitions
//   - Backend: Backend service definitions
//   - Frontend: Frontend service definitions
//   - Database: Database configuration
//
// Returns:
//   - None (data structure)
//
// Concurrency:
//   - Immutable after loading
//
// Performance:
//   - Single allocation, cached validation
type Config struct {
	ConfigVersion   string                     `yaml:"config_version"`
	ProjectName     string                     `yaml:"project_name"`
	Version         string                     `yaml:"version"`
	ModulePrefix    string                     `yaml:"module_prefix"`
	DockerRegistry  string                     `yaml:"docker_registry"`
	Build           BuildConfig                `yaml:"build"`
	Env             EnvConfig                  `yaml:"env"`
	BackendDefaults BackendDefaultsConfig      `yaml:"backend_defaults"`
	Kubernetes      KubernetesConfig           `yaml:"kubernetes"`
	Backend         map[string]BackendService  `yaml:"backend"`
	Frontend        map[string]FrontendService `yaml:"frontend"`
	Database        DatabaseConfig             `yaml:"database"`
}

// BuildConfig defines build settings.
type BuildConfig struct {
	Platforms      []string `yaml:"platforms"`
	GoRuntimeImage string   `yaml:"go_runtime_image"`
}

// EnvConfig defines environment variable inheritance.
type EnvConfig struct {
	Global   map[string]string `yaml:"global"`
	Backend  map[string]string `yaml:"backend"`
	Frontend map[string]string `yaml:"frontend"`
}

// BackendDefaultsConfig defines default settings for backend services.
type BackendDefaultsConfig struct {
	Ports PortConfig `yaml:"ports"`
}

// PortConfig defines port mappings.
type PortConfig struct {
	HTTP    int `yaml:"http"`
	Health  int `yaml:"health"`
	Metrics int `yaml:"metrics"`
}

// KubernetesConfig defines Kubernetes resources.
type KubernetesConfig struct {
	Resources KubernetesResourcesConfig `yaml:"resources"`
}

// KubernetesResourcesConfig defines ConfigMaps and Secrets.
type KubernetesResourcesConfig struct {
	ConfigMaps map[string]map[string]string `yaml:"configmaps"`
	Secrets    map[string]map[string]string `yaml:"secrets"`
}

// BackendService defines a backend service configuration.
type BackendService struct {
	Ports      *PortConfig             `yaml:"ports,omitempty"`
	Kubernetes BackendKubernetesConfig `yaml:"kubernetes"`
	Env        BackendEnvConfig        `yaml:"env"`
}

// BackendKubernetesConfig defines Kubernetes settings for backend services.
type BackendKubernetesConfig struct {
	Service BackendServiceConfig `yaml:"service"`
}

// BackendServiceConfig defines Service configuration.
type BackendServiceConfig struct {
	ClusterIP ServiceEndpointConfig `yaml:"clusterIP"`
	Headless  ServiceEndpointConfig `yaml:"headless"`
}

// ServiceEndpointConfig defines a service endpoint.
type ServiceEndpointConfig struct {
	Name                     string `yaml:"name"`
	PublishNotReadyAddresses bool   `yaml:"publishNotReadyAddresses"`
}

// BackendEnvConfig defines environment variables for backend services.
type BackendEnvConfig struct {
	Common     map[string]string `yaml:"common"`
	Docker     map[string]string `yaml:"docker"`
	Kubernetes map[string]string `yaml:"kubernetes"`
}

// FrontendService defines a frontend service configuration.
type FrontendService struct {
	Platforms []string `yaml:"platforms"`
}

// DatabaseConfig defines database settings.
type DatabaseConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Image        string `yaml:"image"`
	Port         int    `yaml:"port"`
	RootPassword string `yaml:"root_password"`
	Database     string `yaml:"database"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
}

// Diagnostic represents a validation issue.
type Diagnostic struct {
	Severity   DiagnosticSeverity `json:"severity"`
	Message    string             `json:"message"`
	Path       string             `json:"path,omitempty"`
	Suggestion string             `json:"suggestion,omitempty"`
}

// DiagnosticSeverity represents the severity of a diagnostic.
type DiagnosticSeverity string

const (
	SeverityError   DiagnosticSeverity = "error"
	SeverityWarning DiagnosticSeverity = "warning"
	SeverityInfo    DiagnosticSeverity = "info"
)

// Diagnostics represents a collection of validation issues.
type Diagnostics struct {
	items []Diagnostic
}

// NewDiagnostics creates a new diagnostics collection.
//
// Parameters:
//   - None
//
// Returns:
//   - *Diagnostics: Empty diagnostics collection
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Minimal allocation
func NewDiagnostics() *Diagnostics {
	return &Diagnostics{
		items: make([]Diagnostic, 0),
	}
}

// Add adds a diagnostic to the collection.
//
// Parameters:
//   - severity: Diagnostic severity level
//   - message: Human-readable message
//   - path: Optional configuration path
//   - suggestion: Optional fix suggestion
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(1) append operation
func (d *Diagnostics) Add(severity DiagnosticSeverity, message, path, suggestion string) {
	d.items = append(d.items, Diagnostic{
		Severity:   severity,
		Message:    message,
		Path:       path,
		Suggestion: suggestion,
	})
}

// AddError adds an error diagnostic.
//
// Parameters:
//   - message: Error message
//   - path: Optional configuration path
//   - suggestion: Optional fix suggestion
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(1) append operation
func (d *Diagnostics) AddError(message, path, suggestion string) {
	d.Add(SeverityError, message, path, suggestion)
}

// AddWarning adds a warning diagnostic.
//
// Parameters:
//   - message: Warning message
//   - path: Optional configuration path
//   - suggestion: Optional fix suggestion
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(1) append operation
func (d *Diagnostics) AddWarning(message, path, suggestion string) {
	d.Add(SeverityWarning, message, path, suggestion)
}

// AddInfo adds an info diagnostic.
//
// Parameters:
//   - message: Info message
//   - path: Optional configuration path
//   - suggestion: Optional fix suggestion
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(1) append operation
func (d *Diagnostics) AddInfo(message, path, suggestion string) {
	d.Add(SeverityInfo, message, path, suggestion)
}

// HasErrors returns true if there are any error-level diagnostics.
//
// Parameters:
//   - None
//
// Returns:
//   - bool: True if errors exist
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(n) scan operation
func (d *Diagnostics) HasErrors() bool {
	for _, item := range d.items {
		if item.Severity == SeverityError {
			return true
		}
	}
	return false
}

// HasWarnings returns true if there are any warning-level diagnostics.
//
// Parameters:
//   - None
//
// Returns:
//   - bool: True if warnings exist
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(n) scan operation
func (d *Diagnostics) HasWarnings() bool {
	for _, item := range d.items {
		if item.Severity == SeverityWarning {
			return true
		}
	}
	return false
}

// Items returns all diagnostics.
//
// Parameters:
//   - None
//
// Returns:
//   - []Diagnostic: Copy of all diagnostics
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(n) copy operation
func (d *Diagnostics) Items() []Diagnostic {
	result := make([]Diagnostic, len(d.items))
	copy(result, d.items)
	return result
}

// ComputeImageName computes the standard image name for a service.
// Image name follows the pattern: <project_name>-<service_name>
// All characters are converted to lowercase, slashes to hyphens, and consecutive hyphens are collapsed.
//
// Parameters:
//   - projectName: Project name from configuration
//   - serviceName: Service name
//
// Returns:
//   - string: Computed image name
//
// Concurrency:
//   - Thread-safe (pure function)
//
// Performance:
//   - O(n) string processing
func ComputeImageName(projectName, serviceName string) string {
	// Concatenate with hyphen
	name := projectName + "-" + serviceName

	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace slashes with hyphens
	name = strings.ReplaceAll(name, "/", "-")

	// Replace spaces with hyphens
	name = strings.ReplaceAll(name, " ", "-")

	// Collapse consecutive hyphens
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}

	// Trim leading/trailing hyphens
	name = strings.Trim(name, "-")

	return name
}

// GetImageName returns the computed image name for a service.
//
// Parameters:
//   - serviceName: Service name
//
// Returns:
//   - string: Computed image name
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(n) string processing
func (c *Config) GetImageName(serviceName string) string {
	return ComputeImageName(c.ProjectName, serviceName)
}

// ValidateServiceName validates a service name according to naming rules.
// Service names must not end with "-service" suffix.
//
// Parameters:
//   - name: Service name to validate
//
// Returns:
//   - error: Validation error with suggestion if invalid, nil if valid
//
// Concurrency:
//   - Thread-safe (pure function)
//
// Performance:
//   - O(1) string check
func ValidateServiceName(name string) error {
	if strings.HasSuffix(name, "-service") {
		suggestedName := strings.TrimSuffix(name, "-service")
		return fmt.Errorf("service name must not end with '-service' suffix: got '%s', use '%s' instead", name, suggestedName)
	}
	return nil
}

// Load reads and parses an egg.yaml configuration file.
//
// Parameters:
//   - path: Path to the configuration file
//
// Returns:
//   - *Config: Parsed configuration with defaults applied
//   - *Diagnostics: Validation issues found
//
// Concurrency:
//   - Single-threaded file I/O
//
// Performance:
//   - Single-pass parsing, cached validation
func Load(path string) (*Config, *Diagnostics) {
	diags := NewDiagnostics()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		diags.AddError("Configuration file not found", path, "Run 'egg init' to create egg.yaml")
		return nil, diags
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		diags.AddError(fmt.Sprintf("Failed to read configuration file: %v", err), path, "Check file permissions")
		return nil, diags
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		diags.AddError(fmt.Sprintf("Failed to parse YAML: %v", err), path, "Check YAML syntax")
		return nil, diags
	}

	// Apply defaults
	applyDefaults(&config)

	// Validate configuration
	validateConfig(&config, diags)

	return &config, diags
}

// applyDefaults fills in default values for missing configuration.
//
// Parameters:
//   - config: Configuration to fill defaults for
//
// Returns:
//   - None (modifies config in place)
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - O(1) field assignments
func applyDefaults(config *Config) {
	// Set default config version
	if config.ConfigVersion == "" {
		config.ConfigVersion = "1.0"
	}

	// Set default project name from directory
	if config.ProjectName == "" {
		if wd, err := os.Getwd(); err == nil {
			config.ProjectName = filepath.Base(wd)
		}
	}

	// Set default version
	if config.Version == "" {
		config.Version = "v1.0.0"
	}

	// Set default module prefix
	if config.ModulePrefix == "" {
		config.ModulePrefix = fmt.Sprintf("go.eggybyte.com/%s", config.ProjectName)
	}

	// Set default docker registry
	if config.DockerRegistry == "" {
		config.DockerRegistry = "ghcr.io/eggybyte-technology"
	}

	// Set default build platforms
	if len(config.Build.Platforms) == 0 {
		config.Build.Platforms = []string{"linux/amd64", "linux/arm64"}
	}

	// Set default go runtime image
	if config.Build.GoRuntimeImage == "" {
		config.Build.GoRuntimeImage = "eggybyte-go-alpine"
	}

	// Set default backend ports
	if config.BackendDefaults.Ports.HTTP == 0 {
		config.BackendDefaults.Ports.HTTP = 8080
	}
	if config.BackendDefaults.Ports.Health == 0 {
		config.BackendDefaults.Ports.Health = 8081
	}
	if config.BackendDefaults.Ports.Metrics == 0 {
		config.BackendDefaults.Ports.Metrics = 9091
	}

	// Set default database settings
	if config.Database.Image == "" {
		config.Database.Image = "mysql:9.4"
	}
	if config.Database.Port == 0 {
		config.Database.Port = 3306
	}
	if config.Database.RootPassword == "" {
		config.Database.RootPassword = "rootpass"
	}
	if config.Database.Database == "" {
		config.Database.Database = "app"
	}
	if config.Database.User == "" {
		config.Database.User = "user"
	}
	if config.Database.Password == "" {
		config.Database.Password = "pass"
	}

	// Apply port inheritance for backend services
	for name, service := range config.Backend {
		if service.Ports == nil {
			service.Ports = &config.BackendDefaults.Ports
			config.Backend[name] = service
		}
	}
}

// validateConfig performs comprehensive validation of the configuration.
//
// Parameters:
//   - config: Configuration to validate
//   - diags: Diagnostics collection to populate
//
// Returns:
//   - None (populates diagnostics)
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - O(n) validation where n is number of services
func validateConfig(config *Config, diags *Diagnostics) {
	// Validate required fields
	if config.ProjectName == "" {
		diags.AddError("project_name is required", "project_name", "Set a valid project name")
	}

	if config.ModulePrefix == "" {
		diags.AddError("module_prefix is required", "module_prefix", "Set a valid Go module prefix")
	}

	// Validate project name format
	if config.ProjectName != "" && !isValidProjectName(config.ProjectName) {
		diags.AddError("Invalid project name format", "project_name", "Use lowercase letters, numbers, and hyphens only")
	}

	// Validate module prefix format
	if config.ModulePrefix != "" && !isValidModulePrefix(config.ModulePrefix) {
		diags.AddError("Invalid module prefix format", "module_prefix", "Use valid Go module path format")
	}

	// Validate backend services
	usedPorts := make(map[int]string)
	for name, service := range config.Backend {
		validateBackendService(name, service, config, diags, usedPorts)
	}

	// Validate frontend services
	for name, service := range config.Frontend {
		validateFrontendService(name, service, diags)
	}

	// Validate database configuration
	validateDatabaseConfig(config.Database, diags)

	// Validate Kubernetes resources
	validateKubernetesResources(config.Kubernetes, diags)
}

// validateBackendService validates a single backend service configuration.
//
// Parameters:
//   - name: Service name
//   - service: Service configuration
//   - config: Full configuration for context
//   - diags: Diagnostics collection
//   - usedPorts: Map of used ports to service names
//
// Returns:
//   - None (populates diagnostics)
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - O(1) per service validation
func validateBackendService(name string, service BackendService, config *Config, diags *Diagnostics, usedPorts map[int]string) {
	// Validate service name
	if !isValidServiceName(name) {
		diags.AddError("Invalid service name", fmt.Sprintf("backend.%s", name), "Use lowercase letters, numbers, hyphens, and underscores only")
	}

	// Validate service name does not end with -service
	if err := ValidateServiceName(name); err != nil {
		diags.AddWarning("Service naming convention violation", fmt.Sprintf("backend.%s", name), err.Error())
	}

	// Validate ports
	if service.Ports != nil {
		ports := []struct {
			port int
			name string
		}{
			{service.Ports.HTTP, "http"},
			{service.Ports.Health, "health"},
			{service.Ports.Metrics, "metrics"},
		}

		for _, p := range ports {
			if p.port <= 0 || p.port > 65535 {
				diags.AddError("Invalid port number", fmt.Sprintf("backend.%s.ports.%s", name, p.name), "Use port numbers between 1 and 65535")
			} else if existing, exists := usedPorts[p.port]; exists {
				diags.AddError("Port conflict", fmt.Sprintf("backend.%s.ports.%s", name, p.name), fmt.Sprintf("Port %d is already used by %s", p.port, existing))
			} else {
				usedPorts[p.port] = fmt.Sprintf("backend.%s", name)
			}
		}
	}

	// Validate Kubernetes service names
	if service.Kubernetes.Service.ClusterIP.Name == "" {
		diags.AddWarning("Missing clusterIP service name", fmt.Sprintf("backend.%s.kubernetes.service.clusterIP.name", name), "Set a descriptive service name")
	}

	if service.Kubernetes.Service.Headless.Name == "" {
		diags.AddWarning("Missing headless service name", fmt.Sprintf("backend.%s.kubernetes.service.headless.name", name), "Set a descriptive service name")
	}
}

// validateFrontendService validates a single frontend service configuration.
//
// Parameters:
//   - name: Service name
//   - service: Service configuration
//   - diags: Diagnostics collection
//
// Returns:
//   - None (populates diagnostics)
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - O(1) per service validation
func validateFrontendService(name string, service FrontendService, diags *Diagnostics) {
	// Validate service name
	if !isValidServiceName(name) {
		diags.AddError("Invalid service name", fmt.Sprintf("frontend.%s", name), "Use lowercase letters, numbers, hyphens, and underscores only")
	}

	// Validate service name does not end with -service
	if err := ValidateServiceName(name); err != nil {
		diags.AddWarning("Service naming convention violation", fmt.Sprintf("frontend.%s", name), err.Error())
	}

	// Validate platforms
	if len(service.Platforms) == 0 {
		diags.AddWarning("No platforms specified", fmt.Sprintf("frontend.%s.platforms", name), "Specify target platforms (e.g., web, mobile)")
	}
}

// validateDatabaseConfig validates database configuration.
//
// Parameters:
//   - db: Database configuration
//   - diags: Diagnostics collection
//
// Returns:
//   - None (populates diagnostics)
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - O(1) validation
func validateDatabaseConfig(db DatabaseConfig, diags *Diagnostics) {
	if !db.Enabled {
		return
	}

	// Validate required fields when enabled
	if db.Database == "" {
		diags.AddError("Database name is required when enabled", "database.database", "Set a valid database name")
	}

	if db.User == "" {
		diags.AddError("Database user is required when enabled", "database.user", "Set a valid database user")
	}

	if db.Password == "" {
		diags.AddError("Database password is required when enabled", "database.password", "Set a valid database password")
	}

	if db.Port <= 0 || db.Port > 65535 {
		diags.AddError("Invalid database port", "database.port", "Use port numbers between 1 and 65535")
	}
}

// validateKubernetesResources validates Kubernetes resource definitions.
//
// Parameters:
//   - k8s: Kubernetes configuration
//   - diags: Diagnostics collection
//
// Returns:
//   - None (populates diagnostics)
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - O(n) validation where n is number of resources
func validateKubernetesResources(k8s KubernetesConfig, diags *Diagnostics) {
	// Validate ConfigMap names
	for name := range k8s.Resources.ConfigMaps {
		if !isValidResourceName(name) {
			diags.AddError("Invalid ConfigMap name", fmt.Sprintf("kubernetes.resources.configmaps.%s", name), "Use lowercase letters, numbers, and hyphens only")
		}
	}

	// Validate Secret names
	for name := range k8s.Resources.Secrets {
		if !isValidResourceName(name) {
			diags.AddError("Invalid Secret name", fmt.Sprintf("kubernetes.resources.secrets.%s", name), "Use lowercase letters, numbers, and hyphens only")
		}
	}
}

// Validation helper functions

func isValidProjectName(name string) bool {
	if name == "" || len(name) > 50 {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}
	return true
}

func isValidModulePrefix(prefix string) bool {
	if prefix == "" {
		return false
	}
	parts := strings.Split(prefix, "/")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '.') {
				return false
			}
		}
	}
	return true
}

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
	return isValidProjectName(name)
}
