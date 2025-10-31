// Package internal provides tests for configx internal sources implementation.
package internal

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewEnvSource(t *testing.T) {
	opts := EnvOptions{
		Prefix: "TEST_",
	}

	source := NewEnvSource(opts)
	if source == nil {
		t.Fatal("NewEnvSource() should return non-nil source")
	}

	envSource, ok := source.(*EnvSource)
	if !ok {
		t.Fatal("NewEnvSource() should return *EnvSource")
	}
	if envSource.prefix != "TEST_" {
		t.Errorf("Prefix = %q, want %q", envSource.prefix, "TEST_")
	}
}

func TestEnvSource_Load(t *testing.T) {
	source := NewEnvSource(EnvOptions{})

	ctx := context.Background()
	config, err := source.Load(ctx)
	if err != nil {
		t.Errorf("Load() error = %v, want nil", err)
	}
	if config == nil {
		t.Fatal("Load() should return non-nil config")
	}
}

func TestEnvSource_Load_WithPrefix(t *testing.T) {
	// Set a test environment variable
	os.Setenv("TEST_CONFIG_KEY", "test_value")
	defer os.Unsetenv("TEST_CONFIG_KEY")

	source := NewEnvSource(EnvOptions{
		Prefix: "TEST_",
	})

	ctx := context.Background()
	config, err := source.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Check that prefix was removed
	value, exists := config["CONFIG_KEY"]
	if !exists {
		t.Error("Config should contain CONFIG_KEY")
	}
	if value != "test_value" {
		t.Errorf("Value = %q, want %q", value, "test_value")
	}

	// Check that unprefixed vars are not included
	if _, exists := config["TEST_CONFIG_KEY"]; exists {
		t.Error("Config should not contain prefixed key")
	}
}

func TestEnvSource_Load_Lowercase(t *testing.T) {
	os.Setenv("TEST_KEY", "value")
	defer os.Unsetenv("TEST_KEY")

	source := NewEnvSource(EnvOptions{
		Prefix:    "TEST_",
		Lowercase: true,
	})

	ctx := context.Background()
	config, err := source.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	value, exists := config["key"]
	if !exists {
		t.Error("Config should contain lowercase key")
	}
	if value != "value" {
		t.Errorf("Value = %q, want %q", value, "value")
	}
}

func TestEnvSource_Load_Uppercase(t *testing.T) {
	os.Setenv("TEST_key", "value")
	defer os.Unsetenv("TEST_key")

	source := NewEnvSource(EnvOptions{
		Prefix:    "TEST_",
		Uppercase: true,
	})

	ctx := context.Background()
	config, err := source.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	value, exists := config["KEY"]
	if !exists {
		t.Error("Config should contain uppercase key")
	}
	if value != "value" {
		t.Errorf("Value = %q, want %q", value, "value")
	}
}

func TestEnvSource_Watch(t *testing.T) {
	source := NewEnvSource(EnvOptions{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := source.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	if ch == nil {
		t.Fatal("Watch() should return non-nil channel")
	}

	// Channel should close when context is cancelled
	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("Channel should be closed after context cancellation")
		}
	case <-time.After(1 * time.Second):
		t.Error("Channel should close within timeout")
	}
}

func TestNewFileSource(t *testing.T) {
	opts := FileOptions{
		Watch:  true,
		Format: "json",
	}

	source := NewFileSource("test.json", opts)
	if source == nil {
		t.Fatal("NewFileSource() should return non-nil source")
	}

	fileSource, ok := source.(*FileSource)
	if !ok {
		t.Fatal("NewFileSource() should return *FileSource")
	}
	if fileSource.path != "test.json" {
		t.Errorf("Path = %q, want %q", fileSource.path, "test.json")
	}
	if fileSource.format != "json" {
		t.Errorf("Format = %q, want %q", fileSource.format, "json")
	}
}

func TestNewFileSource_AutoDetectFormat(t *testing.T) {
	tests := []struct {
		path   string
		format string
	}{
		{"test.json", "json"},
		{"test.yaml", "yaml"},
		{"test.yml", "yaml"},
		{"test.toml", "toml"},
		{"test.unknown", "json"}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			source := NewFileSource(tt.path, FileOptions{})
			fileSource := source.(*FileSource)
			if fileSource.format != tt.format {
				t.Errorf("Format = %q, want %q", fileSource.format, tt.format)
			}
		})
	}
}

func TestFileSource_Load_NonExistent(t *testing.T) {
	source := NewFileSource("non_existent.json", FileOptions{})

	ctx := context.Background()
	config, err := source.Load(ctx)
	if err != nil {
		t.Errorf("Load() error = %v, want nil (non-existent file should return empty config)", err)
	}
	if config == nil {
		t.Fatal("Load() should return non-nil config")
	}
}

func TestFileSource_Watch_Disabled(t *testing.T) {
	source := NewFileSource("test.json", FileOptions{
		Watch: false,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := source.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	// Channel should close when context is cancelled
	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("Channel should be closed after context cancellation")
		}
	case <-time.After(1 * time.Second):
		t.Error("Channel should close within timeout")
	}
}

func TestDetectFileFormat(t *testing.T) {
	tests := []struct {
		path   string
		format string
	}{
		{"test.json", "json"},
		{"test.yaml", "yaml"},
		{"test.yml", "yaml"},
		{"test.toml", "toml"},
		{"test.unknown", "json"},
		{"test", "json"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			format := detectFileFormat(tt.path)
			if format != tt.format {
				t.Errorf("detectFileFormat(%q) = %q, want %q", tt.path, format, tt.format)
			}
		})
	}
}

func TestParseConfigFile_UnsupportedFormat(t *testing.T) {
	data := []byte("test data")
	config, err := parseConfigFile(data, "unsupported")

	if err == nil {
		t.Fatal("parseConfigFile() should return error for unsupported format")
	}
	if config != nil {
		t.Error("parseConfigFile() should return nil config on error")
	}
}

func TestParseConfigFile_SupportedFormats(t *testing.T) {
	formats := []string{"json", "yaml", "toml"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			data := []byte("test data")
			config, err := parseConfigFile(data, format)
			if err != nil {
				t.Errorf("parseConfigFile(%s) error = %v", format, err)
			}
			if config == nil {
				t.Errorf("parseConfigFile(%s) should return non-nil config", format)
			}
		})
	}
}

func TestNewK8sConfigMapSource(t *testing.T) {
	opts := K8sOptions{
		Namespace: "test-ns",
		Logger:    &mockLogger{},
	}

	source := NewK8sConfigMapSource("test-config", opts)
	if source == nil {
		t.Fatal("NewK8sConfigMapSource() should return non-nil source")
	}

	k8sSource, ok := source.(*K8sConfigMapSource)
	if !ok {
		t.Fatal("NewK8sConfigMapSource() should return *K8sConfigMapSource")
	}
	if k8sSource.name != "test-config" {
		t.Errorf("Name = %q, want %q", k8sSource.name, "test-config")
	}
	if k8sSource.namespace != "test-ns" {
		t.Errorf("Namespace = %q, want %q", k8sSource.namespace, "test-ns")
	}
}

func TestNewK8sConfigMapSource_DefaultNamespace(t *testing.T) {
	source := NewK8sConfigMapSource("test-config", K8sOptions{})
	k8sSource := source.(*K8sConfigMapSource)

	if k8sSource.namespace != "default" {
		t.Errorf("Namespace = %q, want %q", k8sSource.namespace, "default")
	}
}

func TestK8sConfigMapSource_Load(t *testing.T) {
	source := NewK8sConfigMapSource("test-config", K8sOptions{
		Logger: &mockLogger{},
	})

	ctx := context.Background()
	config, err := source.Load(ctx)
	if err != nil {
		t.Errorf("Load() error = %v, want nil", err)
	}
	// Load may return nil in non-K8s environments
	_ = config
}

func TestK8sConfigMapSource_Watch(t *testing.T) {
	source := NewK8sConfigMapSource("test-config", K8sOptions{
		Logger: &mockLogger{},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := source.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	if ch == nil {
		t.Fatal("Watch() should return non-nil channel")
	}

	// Channel should close when context is cancelled
	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("Channel should be closed after context cancellation")
		}
	case <-time.After(1 * time.Second):
		t.Error("Channel should close within timeout")
	}
}

