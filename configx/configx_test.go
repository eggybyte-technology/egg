// Package configx provides tests for configuration management.
package configx

import (
	"context"
	"os"
	"testing"
	"time"

	"go.eggybyte.com/egg/configx/internal"
	"go.eggybyte.com/egg/core/log"
)

// testLogger is a test logger implementation.
type testLogger struct {
	logs []string
}

func (l *testLogger) With(kv ...any) log.Logger              { return l }
func (l *testLogger) Debug(msg string, kv ...any)            { l.logs = append(l.logs, "DEBUG: "+msg) }
func (l *testLogger) Info(msg string, kv ...any)             { l.logs = append(l.logs, "INFO: "+msg) }
func (l *testLogger) Warn(msg string, kv ...any)             { l.logs = append(l.logs, "WARN: "+msg) }
func (l *testLogger) Error(err error, msg string, kv ...any) { l.logs = append(l.logs, "ERROR: "+msg) }

func TestEnvSource(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_KEY1", "value1")
	os.Setenv("TEST_KEY2", "value2")
	os.Setenv("APP_TEST_KEY3", "value3")
	defer func() {
		os.Unsetenv("TEST_KEY1")
		os.Unsetenv("TEST_KEY2")
		os.Unsetenv("APP_TEST_KEY3")
	}()

	tests := []struct {
		name     string
		opts     EnvOptions
		expected map[string]string
	}{
		{
			name: "no prefix",
			opts: EnvOptions{},
			expected: map[string]string{
				"TEST_KEY1":     "value1",
				"TEST_KEY2":     "value2",
				"APP_TEST_KEY3": "value3",
			},
		},
		{
			name: "with prefix",
			opts: EnvOptions{Prefix: "APP_"},
			expected: map[string]string{
				"TEST_KEY3": "value3",
			},
		},
		{
			name: "lowercase",
			opts: EnvOptions{Lowercase: true},
			expected: map[string]string{
				"test_key1":     "value1",
				"test_key2":     "value2",
				"app_test_key3": "value3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := NewEnvSource(tt.opts)
			config, err := source.Load(context.Background())
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			// Check that we have at least the expected keys
			for key, expectedValue := range tt.expected {
				if value, exists := config[key]; !exists {
					t.Errorf("Expected key %s not found", key)
				} else if value != expectedValue {
					t.Errorf("Key %s = %v, want %v", key, value, expectedValue)
				}
			}
		})
	}
}

func TestManager(t *testing.T) {
	logger := &testLogger{}

	// Create a simple source for testing
	source := NewEnvSource(EnvOptions{})

	manager, err := NewManager(context.Background(), Options{
		Logger:   logger,
		Sources:  []Source{source},
		Debounce: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Test Snapshot
	snapshot := manager.Snapshot()
	if snapshot == nil {
		t.Error("Snapshot() returned nil")
	}

	// Test Value
	value, exists := manager.Value("NONEXISTENT_KEY")
	if exists {
		t.Error("Value() should return false for non-existent key")
	}
	if value != "" {
		t.Error("Value() should return empty string for non-existent key")
	}

	// Test Bind
	type TestConfig struct {
		ServiceName string `env:"SERVICE_NAME" default:"test"`
		HTTPPort    string `env:"HTTP_PORT" default:":8080"`
	}

	var cfg TestConfig
	err = manager.Bind(&cfg)
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}

	if cfg.ServiceName != "test" {
		t.Errorf("ServiceName = %v, want test", cfg.ServiceName)
	}
	if cfg.HTTPPort != ":8080" {
		t.Errorf("HTTPPort = %v, want :8080", cfg.HTTPPort)
	}
}

func TestBuildSources(t *testing.T) {
	logger := &testLogger{}

	// Test with no ConfigMap names
	sources, err := internal.BuildSources(context.Background(), logger)
	if err != nil {
		t.Fatalf("BuildSources() error = %v", err)
	}

	if len(sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(sources))
	}

	// Test with ConfigMap names
	os.Setenv("APP_CONFIGMAP_NAME", "app-config")
	os.Setenv("CACHE_CONFIGMAP_NAME", "cache-config")
	defer func() {
		os.Unsetenv("APP_CONFIGMAP_NAME")
		os.Unsetenv("CACHE_CONFIGMAP_NAME")
	}()

	sources, err = internal.BuildSources(context.Background(), logger)
	if err != nil {
		t.Fatalf("BuildSources() error = %v", err)
	}

	if len(sources) != 3 { // Env + 2 ConfigMaps
		t.Errorf("Expected 3 sources, got %d", len(sources))
	}
}

func TestBaseConfig(t *testing.T) {
	logger := &testLogger{}
	source := NewEnvSource(EnvOptions{})

	manager, err := NewManager(context.Background(), Options{
		Logger:  logger,
		Sources: []Source{source},
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	var cfg BaseConfig
	err = manager.Bind(&cfg)
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}

	// Check default values
	if cfg.ServiceName != "app" {
		t.Errorf("ServiceName = %v, want app", cfg.ServiceName)
	}
	if cfg.ServiceVersion != "0.0.0" {
		t.Errorf("ServiceVersion = %v, want 0.0.0", cfg.ServiceVersion)
	}
	if cfg.Env != "dev" {
		t.Errorf("Env = %v, want dev", cfg.Env)
	}
	if cfg.HTTPPort != ":8080" {
		t.Errorf("HTTPPort = %v, want :8080", cfg.HTTPPort)
	}
	if cfg.HealthPort != ":8081" {
		t.Errorf("HealthPort = %v, want :8081", cfg.HealthPort)
	}
	if cfg.MetricsPort != ":9091" {
		t.Errorf("MetricsPort = %v, want :9091", cfg.MetricsPort)
	}
}

func TestOnUpdate(t *testing.T) {
	logger := &testLogger{}
	source := NewEnvSource(EnvOptions{})

	manager, err := NewManager(context.Background(), Options{
		Logger:   logger,
		Sources:  []Source{source},
		Debounce: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	updateCount := 0
	unsubscribe := manager.OnUpdate(func(snapshot map[string]string) {
		updateCount++
	})

	// Wait a bit to see if any updates come through
	time.Sleep(100 * time.Millisecond)

	// Unsubscribe
	unsubscribe()

	// Wait a bit more
	time.Sleep(100 * time.Millisecond)

	// Update count should still be 0 since env source doesn't send updates
	if updateCount != 0 {
		t.Errorf("Expected 0 updates, got %d", updateCount)
	}
}
