// Package internal provides tests for configx internal implementation.
package internal

import (
	"testing"
	"time"
)

func TestBindToStruct_BasicTypes(t *testing.T) {
	type Config struct {
		StringField string `env:"STRING_FIELD"`
		IntField    int    `env:"INT_FIELD"`
		UintField   uint   `env:"UINT_FIELD"`
		BoolField   bool   `env:"BOOL_FIELD"`
		FloatField  float64 `env:"FLOAT_FIELD"`
	}

	snapshot := map[string]string{
		"STRING_FIELD": "test-value",
		"INT_FIELD":    "42",
		"UINT_FIELD":   "100",
		"BOOL_FIELD":   "true",
		"FLOAT_FIELD":  "3.14",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err != nil {
		t.Fatalf("BindToStruct() error = %v", err)
	}

	if cfg.StringField != "test-value" {
		t.Errorf("StringField = %q, want %q", cfg.StringField, "test-value")
	}
	if cfg.IntField != 42 {
		t.Errorf("IntField = %d, want 42", cfg.IntField)
	}
	if cfg.UintField != 100 {
		t.Errorf("UintField = %d, want 100", cfg.UintField)
	}
	if cfg.BoolField != true {
		t.Errorf("BoolField = %v, want true", cfg.BoolField)
	}
	if cfg.FloatField != 3.14 {
		t.Errorf("FloatField = %f, want 3.14", cfg.FloatField)
	}
}

func TestBindToStruct_IntTypes(t *testing.T) {
	type Config struct {
		Int8Field  int8  `env:"INT8_FIELD"`
		Int16Field int16 `env:"INT16_FIELD"`
		Int32Field int32 `env:"INT32_FIELD"`
		Int64Field int64 `env:"INT64_FIELD"`
	}

	snapshot := map[string]string{
		"INT8_FIELD":  "127",
		"INT16_FIELD": "32767",
		"INT32_FIELD": "2147483647",
		"INT64_FIELD": "9223372036854775807",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err != nil {
		t.Fatalf("BindToStruct() error = %v", err)
	}

	if cfg.Int8Field != 127 {
		t.Errorf("Int8Field = %d, want 127", cfg.Int8Field)
	}
	if cfg.Int16Field != 32767 {
		t.Errorf("Int16Field = %d, want 32767", cfg.Int16Field)
	}
	if cfg.Int32Field != 2147483647 {
		t.Errorf("Int32Field = %d, want 2147483647", cfg.Int32Field)
	}
	if cfg.Int64Field != 9223372036854775807 {
		t.Errorf("Int64Field = %d, want 9223372036854775807", cfg.Int64Field)
	}
}

func TestBindToStruct_UintTypes(t *testing.T) {
	type Config struct {
		Uint8Field  uint8  `env:"UINT8_FIELD"`
		Uint16Field uint16 `env:"UINT16_FIELD"`
		Uint32Field uint32 `env:"UINT32_FIELD"`
		Uint64Field uint64 `env:"UINT64_FIELD"`
	}

	snapshot := map[string]string{
		"UINT8_FIELD":  "255",
		"UINT16_FIELD": "65535",
		"UINT32_FIELD": "4294967295",
		"UINT64_FIELD": "18446744073709551615",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err != nil {
		t.Fatalf("BindToStruct() error = %v", err)
	}

	if cfg.Uint8Field != 255 {
		t.Errorf("Uint8Field = %d, want 255", cfg.Uint8Field)
	}
	if cfg.Uint16Field != 65535 {
		t.Errorf("Uint16Field = %d, want 65535", cfg.Uint16Field)
	}
	if cfg.Uint32Field != 4294967295 {
		t.Errorf("Uint32Field = %d, want 4294967295", cfg.Uint32Field)
	}
	if cfg.Uint64Field != 18446744073709551615 {
		t.Errorf("Uint64Field = %d, want 18446744073709551615", cfg.Uint64Field)
	}
}

func TestBindToStruct_Duration(t *testing.T) {
	type Config struct {
		Timeout time.Duration `env:"TIMEOUT"`
	}

	snapshot := map[string]string{
		"TIMEOUT": "5s",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err != nil {
		t.Fatalf("BindToStruct() error = %v", err)
	}

	if cfg.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", cfg.Timeout)
	}
}

func TestBindToStruct_DefaultValues(t *testing.T) {
	type Config struct {
		StringField string `env:"STRING_FIELD" default:"default-value"`
		IntField    int    `env:"INT_FIELD" default:"100"`
		BoolField   bool   `env:"BOOL_FIELD" default:"true"`
	}

	snapshot := map[string]string{}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err != nil {
		t.Fatalf("BindToStruct() error = %v", err)
	}

	if cfg.StringField != "default-value" {
		t.Errorf("StringField = %q, want %q", cfg.StringField, "default-value")
	}
	if cfg.IntField != 100 {
		t.Errorf("IntField = %d, want 100", cfg.IntField)
	}
	if cfg.BoolField != true {
		t.Errorf("BoolField = %v, want true", cfg.BoolField)
	}
}

func TestBindToStruct_DefaultsOverridden(t *testing.T) {
	type Config struct {
		StringField string `env:"STRING_FIELD" default:"default-value"`
		IntField    int    `env:"INT_FIELD" default:"100"`
	}

	snapshot := map[string]string{
		"STRING_FIELD": "override-value",
		"INT_FIELD":    "200",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err != nil {
		t.Fatalf("BindToStruct() error = %v", err)
	}

	if cfg.StringField != "override-value" {
		t.Errorf("StringField = %q, want %q", cfg.StringField, "override-value")
	}
	if cfg.IntField != 200 {
		t.Errorf("IntField = %d, want 200", cfg.IntField)
	}
}

func TestBindToStruct_NestedStruct(t *testing.T) {
	type DatabaseConfig struct {
		Host string `env:"DB_HOST"`
		Port int    `env:"DB_PORT"`
	}

	type Config struct {
		Database DatabaseConfig `env:"DB_"`
		Service  string         `env:"SERVICE_NAME"`
	}

	snapshot := map[string]string{
		"DB_HOST":      "localhost",
		"DB_PORT":      "5432",
		"SERVICE_NAME": "my-service",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err != nil {
		t.Fatalf("BindToStruct() error = %v", err)
	}

	if cfg.Database.Host != "localhost" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "localhost")
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Database.Port = %d, want 5432", cfg.Database.Port)
	}
	if cfg.Service != "my-service" {
		t.Errorf("Service = %q, want %q", cfg.Service, "my-service")
	}
}

func TestBindToStruct_EmbeddedStruct(t *testing.T) {
	type BaseConfig struct {
		ServiceName string `env:"SERVICE_NAME"`
		Version     string `env:"VERSION"`
	}

	type Config struct {
		BaseConfig
		Port int `env:"PORT"`
	}

	snapshot := map[string]string{
		"SERVICE_NAME": "my-service",
		"VERSION":      "1.0.0",
		"PORT":         "8080",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err != nil {
		t.Fatalf("BindToStruct() error = %v", err)
	}

	if cfg.ServiceName != "my-service" {
		t.Errorf("ServiceName = %q, want %q", cfg.ServiceName, "my-service")
	}
	if cfg.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", cfg.Version, "1.0.0")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
}

func TestBindToStruct_EmptyValue(t *testing.T) {
	type Config struct {
		StringField string `env:"STRING_FIELD"`
		IntField    int    `env:"INT_FIELD"`
	}

	snapshot := map[string]string{
		"STRING_FIELD": "",
		"INT_FIELD":    "",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err != nil {
		t.Fatalf("BindToStruct() error = %v", err)
	}

	// Empty values should keep zero values
	if cfg.StringField != "" {
		t.Errorf("StringField = %q, want empty string", cfg.StringField)
	}
	if cfg.IntField != 0 {
		t.Errorf("IntField = %d, want 0", cfg.IntField)
	}
}

func TestBindToStruct_UnexportedFields(t *testing.T) {
	type Config struct {
		ExportedField   string `env:"EXPORTED_FIELD"`
		unexportedField string `env:"UNEXPORTED_FIELD"`
	}

	snapshot := map[string]string{
		"EXPORTED_FIELD":   "value1",
		"UNEXPORTED_FIELD": "value2",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err != nil {
		t.Fatalf("BindToStruct() error = %v", err)
	}

	if cfg.ExportedField != "value1" {
		t.Errorf("ExportedField = %q, want %q", cfg.ExportedField, "value1")
	}
	// Unexported field should remain zero value
	if cfg.unexportedField != "" {
		t.Errorf("unexportedField should not be set, got %q", cfg.unexportedField)
	}
}

func TestBindToStruct_NoEnvTag(t *testing.T) {
	type Config struct {
		StringField string `env:"STRING_FIELD"`
		NoTagField  string
	}

	snapshot := map[string]string{
		"STRING_FIELD": "value",
		"NO_TAG_FIELD": "should-not-be-set",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err != nil {
		t.Fatalf("BindToStruct() error = %v", err)
	}

	if cfg.StringField != "value" {
		t.Errorf("StringField = %q, want %q", cfg.StringField, "value")
	}
	// Field without env tag should remain zero value
	if cfg.NoTagField != "" {
		t.Errorf("NoTagField should not be set, got %q", cfg.NoTagField)
	}
}

func TestBindToStruct_InvalidTarget(t *testing.T) {
	tests := []struct {
		name   string
		target interface{}
	}{
		{
			name:   "non-pointer",
			target: struct{}{},
		},
		{
			name:   "pointer to non-struct",
			target: &[]string{},
		},
		{
			name:   "nil",
			target: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := BindToStruct(map[string]string{}, tt.target, nil)
			if err == nil {
				t.Error("BindToStruct() should return error for invalid target")
			}
		})
	}
}

func TestBindToStruct_InvalidIntValue(t *testing.T) {
	type Config struct {
		IntField int `env:"INT_FIELD"`
	}

	snapshot := map[string]string{
		"INT_FIELD": "not-a-number",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err == nil {
		t.Error("BindToStruct() should return error for invalid int value")
	}
}

func TestBindToStruct_InvalidUintValue(t *testing.T) {
	type Config struct {
		UintField uint `env:"UINT_FIELD"`
	}

	snapshot := map[string]string{
		"UINT_FIELD": "not-a-number",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err == nil {
		t.Error("BindToStruct() should return error for invalid uint value")
	}
}

func TestBindToStruct_InvalidBoolValue(t *testing.T) {
	type Config struct {
		BoolField bool `env:"BOOL_FIELD"`
	}

	snapshot := map[string]string{
		"BOOL_FIELD": "not-a-bool",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err == nil {
		t.Error("BindToStruct() should return error for invalid bool value")
	}
}

func TestBindToStruct_InvalidFloatValue(t *testing.T) {
	type Config struct {
		FloatField float64 `env:"FLOAT_FIELD"`
	}

	snapshot := map[string]string{
		"FLOAT_FIELD": "not-a-float",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err == nil {
		t.Error("BindToStruct() should return error for invalid float value")
	}
}

func TestBindToStruct_InvalidDurationValue(t *testing.T) {
	type Config struct {
		Timeout time.Duration `env:"TIMEOUT"`
	}

	snapshot := map[string]string{
		"TIMEOUT": "not-a-duration",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err == nil {
		t.Error("BindToStruct() should return error for invalid duration value")
	}
}

func TestBindToStruct_UnsupportedType(t *testing.T) {
	type Config struct {
		SliceField []string `env:"SLICE_FIELD"`
	}

	snapshot := map[string]string{
		"SLICE_FIELD": "value",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err == nil {
		t.Error("BindToStruct() should return error for unsupported type")
	}
}

func TestBindToStruct_NestedStructError(t *testing.T) {
	type DatabaseConfig struct {
		Port int `env:"DB_PORT"`
	}

	type Config struct {
		Database DatabaseConfig `env:"DB_"`
	}

	snapshot := map[string]string{
		"DB_PORT": "not-a-number",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err == nil {
		t.Error("BindToStruct() should return error for nested struct field error")
	}
}

func TestBindToStruct_Float32(t *testing.T) {
	type Config struct {
		Float32Field float32 `env:"FLOAT32_FIELD"`
	}

	snapshot := map[string]string{
		"FLOAT32_FIELD": "3.14",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err != nil {
		t.Fatalf("BindToStruct() error = %v", err)
	}

	if cfg.Float32Field != 3.14 {
		t.Errorf("Float32Field = %f, want 3.14", cfg.Float32Field)
	}
}

func TestBindToStruct_BoolFalse(t *testing.T) {
	type Config struct {
		BoolField bool `env:"BOOL_FIELD"`
	}

	snapshot := map[string]string{
		"BOOL_FIELD": "false",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err != nil {
		t.Fatalf("BindToStruct() error = %v", err)
	}

	if cfg.BoolField != false {
		t.Errorf("BoolField = %v, want false", cfg.BoolField)
	}
}

func TestBindToStruct_BoolTrue(t *testing.T) {
	type Config struct {
		BoolField bool `env:"BOOL_FIELD"`
	}

	snapshot := map[string]string{
		"BOOL_FIELD": "true",
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, nil)
	if err != nil {
		t.Fatalf("BindToStruct() error = %v", err)
	}

	if cfg.BoolField != true {
		t.Errorf("BoolField = %v, want true", cfg.BoolField)
	}
}

func TestBindToStruct_OnUpdateCallback(t *testing.T) {
	type Config struct {
		StringField string `env:"STRING_FIELD"`
	}

	snapshot := map[string]string{
		"STRING_FIELD": "value",
	}

	callbackCalled := false
	onUpdate := func() {
		callbackCalled = true
	}

	var cfg Config
	err := BindToStruct(snapshot, &cfg, onUpdate)
	if err != nil {
		t.Fatalf("BindToStruct() error = %v", err)
	}

	// Callback is not called during binding, only on updates
	// This test verifies the callback parameter is accepted
	if callbackCalled {
		t.Error("Callback should not be called during binding")
	}
}

