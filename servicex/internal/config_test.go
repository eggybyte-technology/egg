// Package internal provides tests for internal configuration logic.
package internal

import (
	"log/slog"
	"testing"

	"go.eggybyte.com/egg/configx"
)

// TestNewServiceConfig tests default service configuration creation.
func TestNewServiceConfig(t *testing.T) {
	cfg := NewServiceConfig()

	if cfg == nil {
		t.Fatal("NewServiceConfig returned nil")
	}

	// Check defaults
	if cfg.ServiceName != "app" {
		t.Errorf("ServiceName = %s, want app", cfg.ServiceName)
	}
	if cfg.ServiceVersion != "0.0.0" {
		t.Errorf("ServiceVersion = %s, want 0.0.0", cfg.ServiceVersion)
	}
	if !cfg.EnableMetrics {
		t.Error("EnableMetrics should be true by default")
	}
	if cfg.HTTPPort != 8080 {
		t.Errorf("HTTPPort = %d, want 8080", cfg.HTTPPort)
	}
	if cfg.HealthPort != 8081 {
		t.Errorf("HealthPort = %d, want 8081", cfg.HealthPort)
	}
	if cfg.MetricsPort != 9091 {
		t.Errorf("MetricsPort = %d, want 9091", cfg.MetricsPort)
	}
	if cfg.DefaultTimeoutMs != 30000 {
		t.Errorf("DefaultTimeoutMs = %d, want 30000", cfg.DefaultTimeoutMs)
	}
	if cfg.SlowRequestMillis != 1000 {
		t.Errorf("SlowRequestMillis = %d, want 1000", cfg.SlowRequestMillis)
	}
}

// TestParseLogLevel tests log level parsing.
func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      slog.Level
		wantError bool
	}{
		{"debug lowercase", "debug", slog.LevelDebug, false},
		{"debug uppercase", "DEBUG", slog.LevelDebug, false},
		{"info lowercase", "info", slog.LevelInfo, false},
		{"info uppercase", "INFO", slog.LevelInfo, false},
		{"warn lowercase", "warn", slog.LevelWarn, false},
		{"warn uppercase", "WARN", slog.LevelWarn, false},
		{"warning lowercase", "warning", slog.LevelWarn, false},
		{"warning uppercase", "WARNING", slog.LevelWarn, false},
		{"error lowercase", "error", slog.LevelError, false},
		{"error uppercase", "ERROR", slog.LevelError, false},
		{"unknown level", "invalid", slog.LevelInfo, true},
		{"empty string", "", slog.LevelInfo, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLogLevel(tt.input)

			if tt.wantError && err == nil {
				t.Errorf("ParseLogLevel(%q) expected error but got nil", tt.input)
			}
			if !tt.wantError && err != nil {
				t.Errorf("ParseLogLevel(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestExtractBaseConfig tests BaseConfig extraction.
func TestExtractBaseConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		wantOk   bool
		wantName string // Expected service name if BaseConfig extracted
	}{
		{
			name:   "nil input",
			input:  nil,
			wantOk: false,
		},
		{
			name:   "direct BaseConfig",
			input:  &configx.BaseConfig{ServiceName: "test-service"},
			wantOk: true,
		},
		{
			name: "struct with embedded BaseConfig",
			input: &struct {
				configx.BaseConfig
				CustomField string
			}{
				BaseConfig: configx.BaseConfig{ServiceName: "embedded-service"},
			},
			wantOk: false, // Embedded doesn't implement GetBaseConfig() unless explicitly added
		},
		{
			name:   "non-BaseConfig struct",
			input:  &struct{ Name string }{Name: "test"},
			wantOk: false,
		},
		{
			name:   "empty struct",
			input:  &struct{}{},
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractBaseConfig(tt.input)

			if ok != tt.wantOk {
				t.Errorf("ExtractBaseConfig() ok = %v, want %v", ok, tt.wantOk)
			}

			if tt.wantOk && got == nil {
				t.Error("ExtractBaseConfig() returned nil when expecting BaseConfig")
			}
		})
	}
}

// TestMetricsConfig tests MetricsConfig structure.
func TestMetricsConfig(t *testing.T) {
	tests := []struct {
		name   string
		config MetricsConfig
	}{
		{
			name: "all enabled",
			config: MetricsConfig{
				EnableRuntime: true,
				EnableProcess: true,
				EnableDB:      true,
				EnableClient:  true,
			},
		},
		{
			name: "all disabled",
			config: MetricsConfig{
				EnableRuntime: false,
				EnableProcess: false,
				EnableDB:      false,
				EnableClient:  false,
			},
		},
		{
			name: "mixed",
			config: MetricsConfig{
				EnableRuntime: true,
				EnableProcess: false,
				EnableDB:      true,
				EnableClient:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.config

			if cfg.EnableRuntime != tt.config.EnableRuntime {
				t.Errorf("EnableRuntime mismatch")
			}
			if cfg.EnableProcess != tt.config.EnableProcess {
				t.Errorf("EnableProcess mismatch")
			}
			if cfg.EnableDB != tt.config.EnableDB {
				t.Errorf("EnableDB mismatch")
			}
			if cfg.EnableClient != tt.config.EnableClient {
				t.Errorf("EnableClient mismatch")
			}
		})
	}
}

// TestDatabaseConfig tests DatabaseConfig structure.
func TestDatabaseConfig(t *testing.T) {
	cfg := DatabaseConfig{
		Driver:          "mysql",
		DSN:             "user:pass@tcp(localhost:3306)/test",
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: 3600,
		PingTimeout:     5000,
	}

	if cfg.Driver != "mysql" {
		t.Errorf("Driver = %s, want mysql", cfg.Driver)
	}
	if cfg.DSN == "" {
		t.Error("DSN should not be empty")
	}
	if cfg.MaxIdleConns != 10 {
		t.Errorf("MaxIdleConns = %d, want 10", cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns != 100 {
		t.Errorf("MaxOpenConns = %d, want 100", cfg.MaxOpenConns)
	}
}

// TestServiceConfig tests ServiceConfig structure and defaults.
func TestServiceConfig(t *testing.T) {
	cfg := NewServiceConfig()

	// Test modifying config
	cfg.ServiceName = "custom-service"
	cfg.ServiceVersion = "2.0.0"
	cfg.EnableMetrics = false
	cfg.HTTPPort = 9000

	if cfg.ServiceName != "custom-service" {
		t.Errorf("ServiceName = %s, want custom-service", cfg.ServiceName)
	}
	if cfg.ServiceVersion != "2.0.0" {
		t.Errorf("ServiceVersion = %s, want 2.0.0", cfg.ServiceVersion)
	}
	if cfg.EnableMetrics {
		t.Error("EnableMetrics should be false")
	}
	if cfg.HTTPPort != 9000 {
		t.Errorf("HTTPPort = %d, want 9000", cfg.HTTPPort)
	}
}

// TestServiceConfigWithMetricsConfig tests MetricsConfig integration.
func TestServiceConfigWithMetricsConfig(t *testing.T) {
	cfg := NewServiceConfig()

	cfg.MetricsConfig = &MetricsConfig{
		EnableRuntime: true,
		EnableProcess: true,
		EnableDB:      true,
		EnableClient:  false,
	}

	if cfg.MetricsConfig == nil {
		t.Fatal("MetricsConfig should not be nil")
	}
	if !cfg.MetricsConfig.EnableRuntime {
		t.Error("EnableRuntime should be true")
	}
	if !cfg.MetricsConfig.EnableProcess {
		t.Error("EnableProcess should be true")
	}
	if !cfg.MetricsConfig.EnableDB {
		t.Error("EnableDB should be true")
	}
	if cfg.MetricsConfig.EnableClient {
		t.Error("EnableClient should be false")
	}
}
