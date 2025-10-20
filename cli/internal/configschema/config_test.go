package configschema

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary test config file
	testConfig := `config_version: "1.0"
project_name: "test-project"
version: "v1.0.0"
module_prefix: "github.com/test/test-project"
docker_registry: "ghcr.io/test"

backend_defaults:
  ports:
    http: 8080
    health: 8081
    metrics: 9091

backend: {}
frontend: {}

database:
  enabled: false
`

	// Write test config to temporary file
	tmpFile, err := os.CreateTemp("", "egg-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testConfig); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	tmpFile.Close()

	// Test loading configuration
	config, diags := Load(tmpFile.Name())
	if config == nil {
		t.Fatal("Expected config to be loaded")
	}

	if diags.HasErrors() {
		t.Fatalf("Expected no errors, got: %v", diags.Items())
	}

	// Verify configuration values
	if config.ProjectName != "test-project" {
		t.Errorf("Expected project name 'test-project', got '%s'", config.ProjectName)
	}

	if config.Version != "v1.0.0" {
		t.Errorf("Expected version 'v1.0.0', got '%s'", config.Version)
	}

	if config.BackendDefaults.Ports.HTTP != 8080 {
		t.Errorf("Expected HTTP port 8080, got %d", config.BackendDefaults.Ports.HTTP)
	}
}

func TestLoadNonExistent(t *testing.T) {
	config, diags := Load("/nonexistent/file.yaml")
	if config != nil {
		t.Error("Expected config to be nil for nonexistent file")
	}

	if !diags.HasErrors() {
		t.Error("Expected errors for nonexistent file")
	}
}

func TestApplyDefaults(t *testing.T) {
	config := &Config{
		ProjectName: "test",
	}

	applyDefaults(config)

	if config.ConfigVersion == "" {
		t.Error("Expected default config version to be set")
	}

	if config.BackendDefaults.Ports.HTTP != 8080 {
		t.Errorf("Expected default HTTP port 8080, got %d", config.BackendDefaults.Ports.HTTP)
	}

	if config.Database.Image != "mysql:9.4" {
		t.Errorf("Expected default database image 'mysql:9.4', got '%s'", config.Database.Image)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &Config{
				ProjectName:  "test-project",
				ModulePrefix: "github.com/test/test-project",
			},
			expectError: false,
		},
		{
			name: "missing project name",
			config: &Config{
				ModulePrefix: "github.com/test/test-project",
			},
			expectError: true,
		},
		{
			name: "invalid project name",
			config: &Config{
				ProjectName:  "TEST_PROJECT",
				ModulePrefix: "github.com/test/test-project",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := NewDiagnostics()
			validateConfig(tt.config, diags)

			hasErrors := diags.HasErrors()
			if hasErrors != tt.expectError {
				t.Errorf("Expected error: %v, got: %v", tt.expectError, hasErrors)
			}
		})
	}
}
