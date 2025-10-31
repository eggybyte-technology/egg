// Package internal provides tests for logx internal implementation.
package internal

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewHandler(t *testing.T) {
	opts := Options{
		Format: "logfmt",
		Level:  slog.LevelInfo,
	}
	buf := &bytes.Buffer{}

	handler := NewHandler(opts, buf)
	if handler == nil {
		t.Fatal("NewHandler should return non-nil handler")
	}
	if handler.writer != buf {
		t.Error("Handler writer should be set")
	}
	if handler.opts.Format != "logfmt" {
		t.Errorf("Format = %q, want %q", handler.opts.Format, "logfmt")
	}
}

func TestHandler_Enabled(t *testing.T) {
	tests := []struct {
		name  string
		level slog.Level
		minLevel slog.Level
		want  bool
	}{
		{"debug below info", slog.LevelDebug, slog.LevelInfo, false},
		{"info at info", slog.LevelInfo, slog.LevelInfo, true},
		{"warn above info", slog.LevelWarn, slog.LevelInfo, true},
		{"error above info", slog.LevelError, slog.LevelInfo, true},
		{"debug at debug", slog.LevelDebug, slog.LevelDebug, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(Options{Level: tt.minLevel}, &bytes.Buffer{})
			got := handler.Enabled(context.Background(), tt.level)
			if got != tt.want {
				t.Errorf("Enabled(%v) = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}

func TestHandler_Handle(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewHandler(Options{
		Format:           "logfmt",
		Level:            slog.LevelInfo,
		DisableTimestamp: true,
	}, buf)

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	record.AddAttrs(slog.String("key", "value"))

	err := handler.Handle(context.Background(), record)
	if err != nil {
		t.Errorf("Handle() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Output should contain message, got: %q", output)
	}
	if !strings.Contains(output, "key=\"value\"") {
		t.Errorf("Output should contain key=\"value\", got: %q", output)
	}
}

func TestHandler_LogRecord(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewHandler(Options{
		Format:           "logfmt",
		Level:            slog.LevelInfo,
		DisableTimestamp: true,
	}, buf)

	attrs := []slog.Attr{
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	}

	handler.LogRecord(slog.LevelInfo, "test message", attrs)

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Output should contain message, got: %q", output)
	}
	if !strings.Contains(output, "key1=\"value1\"") {
		t.Errorf("Output should contain key1=\"value1\", got: %q", output)
	}
}

func TestHandler_WithAttrs(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewHandler(Options{
		Format:           "logfmt",
		Level:            slog.LevelInfo,
		DisableTimestamp: true,
	}, buf)

	attrs := []slog.Attr{
		slog.String("base", "value"),
	}

	newHandler := handler.WithAttrs(attrs).(*Handler)
	if newHandler == handler {
		t.Error("WithAttrs should return a new handler")
	}
	if len(newHandler.attrs) != 1 {
		t.Errorf("Expected 1 attr, got %d", len(newHandler.attrs))
	}

	// Test that attrs are included in log output
	newHandler.LogRecord(slog.LevelInfo, "test", nil)
	output := buf.String()
	if !strings.Contains(output, "base=\"value\"") {
		t.Errorf("Output should contain base=\"value\", got: %q", output)
	}
}

func TestHandler_WithGroup(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewHandler(Options{
		Format:           "logfmt",
		Level:            slog.LevelInfo,
		DisableTimestamp: true,
	}, buf)

	newHandler := handler.WithGroup("group").(*Handler)
	if newHandler.group != "group" {
		t.Errorf("Group = %q, want %q", newHandler.group, "group")
	}
	if newHandler == handler {
		t.Error("WithGroup should return a new handler")
	}
}

func TestHandler_LevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewHandler(Options{
		Format:           "logfmt",
		Level:            slog.LevelWarn,
		DisableTimestamp: true,
	}, buf)

	handler.LogRecord(slog.LevelDebug, "debug message", nil)
	handler.LogRecord(slog.LevelInfo, "info message", nil)
	handler.LogRecord(slog.LevelWarn, "warn message", nil)
	handler.LogRecord(slog.LevelError, "error message", nil)

	output := buf.String()
	if strings.Contains(output, "debug message") {
		t.Error("Debug message should be filtered out")
	}
	if strings.Contains(output, "info message") {
		t.Error("Info message should be filtered out")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message should be included")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error message should be included")
	}
}

func TestFormatLogfmt(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewHandler(Options{
		Format:           "logfmt",
		Level:            slog.LevelInfo,
		DisableTimestamp: true,
		Color:            false,
	}, buf)

	attrs := []slog.Attr{
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
		slog.Bool("key3", true),
	}

	handler.LogRecord(slog.LevelInfo, "test message", attrs)

	output := buf.String()
	if !strings.Contains(output, "level=INFO") {
		t.Errorf("Output should contain level=INFO, got: %q", output)
	}
	if !strings.Contains(output, "msg=\"test message\"") {
		t.Errorf("Output should contain msg, got: %q", output)
	}
	if !strings.Contains(output, "key1=\"value1\"") {
		t.Errorf("Output should contain key1, got: %q", output)
	}
	if !strings.Contains(output, "key2=42") {
		t.Errorf("Output should contain key2, got: %q", output)
	}
	if !strings.Contains(output, "key3=true") {
		t.Errorf("Output should contain key3, got: %q", output)
	}
}

func TestFormatConsole(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewHandler(Options{
		Format:           "console",
		Level:            slog.LevelInfo,
		DisableTimestamp: true,
		Color:            false,
	}, buf)

	attrs := []slog.Attr{
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	}

	handler.LogRecord(slog.LevelInfo, "test message", attrs)

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("Output should contain INFO, got: %q", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("Output should contain message, got: %q", output)
	}
	if !strings.Contains(output, "key1: value1") {
		t.Errorf("Output should contain key1: value1, got: %q", output)
	}
}

func TestFormatLogfmt_WithTimestamp(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewHandler(Options{
		Format:           "logfmt",
		Level:            slog.LevelInfo,
		DisableTimestamp: false,
	}, buf)

	handler.LogRecord(slog.LevelInfo, "test message", nil)

	output := buf.String()
	if !strings.Contains(output, "time=") {
		t.Error("Output should contain timestamp when DisableTimestamp is false")
	}
}

func TestFormatLogfmt_WithColor(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewHandler(Options{
		Format:           "logfmt",
		Level:            slog.LevelInfo,
		DisableTimestamp: true,
		Color:            true,
	}, buf)

	handler.LogRecord(slog.LevelInfo, "test message", nil)

	output := buf.String()
	// Check for ANSI color codes
	if !strings.Contains(output, "\033[") {
		t.Error("Output should contain ANSI color codes when Color is enabled")
	}
}

func TestSortAttrs(t *testing.T) {
	attrs := []slog.Attr{
		slog.String("zebra", "value"),
		slog.String("apple", "value"),
		slog.String("banana", "value"),
	}

	sorted := SortAttrs(attrs)

	if len(sorted) != 3 {
		t.Fatalf("Expected 3 attrs, got %d", len(sorted))
	}
	if sorted[0].Key != "apple" {
		t.Errorf("First key = %q, want %q", sorted[0].Key, "apple")
	}
	if sorted[1].Key != "banana" {
		t.Errorf("Second key = %q, want %q", sorted[1].Key, "banana")
	}
	if sorted[2].Key != "zebra" {
		t.Errorf("Third key = %q, want %q", sorted[2].Key, "zebra")
	}
}

func TestKVToAttrs(t *testing.T) {
	tests := []struct {
		name string
		kv   []any
		want int
	}{
		{
			name: "simple pairs",
			kv:   []any{"key1", "value1", "key2", "value2"},
			want: 2,
		},
		{
			name: "nested slice pairs",
			kv:   []any{[]any{"key1", "value1"}},
			want: 1,
		},
		{
			name: "mixed pairs",
			kv:   []any{"key1", "value1", []any{"key2", "value2"}},
			want: 2,
		},
		{
			name: "odd length",
			kv:   []any{"key1", "value1", "key2"},
			want: 1, // Last key without value is ignored
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := KVToAttrs(tt.kv)
			if len(attrs) != tt.want {
				t.Errorf("KVToAttrs() len = %d, want %d", len(attrs), tt.want)
			}
		})
	}
}

func TestFormatValue_String(t *testing.T) {
	opts := Options{}
	value := slog.StringValue("test string")

	result := FormatValue("key", value, opts)
	if result != `"test string"` {
		t.Errorf("FormatValue(string) = %q, want %q", result, `"test string"`)
	}
}

func TestFormatValue_Int(t *testing.T) {
	opts := Options{}
	value := slog.Int64Value(42)

	result := FormatValue("key", value, opts)
	if result != "42" {
		t.Errorf("FormatValue(int) = %q, want %q", result, "42")
	}
}

func TestFormatValue_Uint(t *testing.T) {
	opts := Options{}
	value := slog.Uint64Value(100)

	result := FormatValue("key", value, opts)
	if result != "100" {
		t.Errorf("FormatValue(uint) = %q, want %q", result, "100")
	}
}

func TestFormatValue_Float(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  string
	}{
		{"integer float", 42.0, "42"},
		{"decimal float", 3.14, "3.14"},
		{"many decimals", 3.141592653589793, "3.141593"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{}
			value := slog.Float64Value(tt.value)
			result := FormatValue("key", value, opts)
			if !strings.Contains(result, tt.want) {
				t.Errorf("FormatValue(float) = %q, want to contain %q", result, tt.want)
			}
		})
	}
}

func TestFormatValue_Bool(t *testing.T) {
	tests := []struct {
		name  string
		value bool
		want  string
	}{
		{"true", true, "true"},
		{"false", false, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{}
			value := slog.BoolValue(tt.value)
			result := FormatValue("key", value, opts)
			if result != tt.want {
				t.Errorf("FormatValue(bool) = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestFormatValue_Duration(t *testing.T) {
	opts := Options{}
	value := slog.DurationValue(5 * time.Second)

	result := FormatValue("key", value, opts)
	if result != "5000" {
		t.Errorf("FormatValue(duration) = %q, want %q", result, "5000")
	}
}

func TestFormatValue_Time(t *testing.T) {
	opts := Options{}
	now := time.Now()
	value := slog.TimeValue(now)

	result := FormatValue("key", value, opts)
	if !strings.Contains(result, now.Format(time.RFC3339)) {
		t.Errorf("FormatValue(time) = %q, want to contain RFC3339 format", result)
	}
}

func TestFormatValue_SensitiveField(t *testing.T) {
	opts := Options{
		SensitiveFields: []string{"password", "token"},
	}

	tests := []struct {
		key   string
		value slog.Value
	}{
		{"password", slog.StringValue("secret123")},
		{"PASSWORD", slog.StringValue("secret123")}, // Case insensitive
		{"token", slog.StringValue("abc123")},
		{"api_key", slog.StringValue("not sensitive")},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := FormatValue(tt.key, tt.value, opts)
			if strings.Contains(strings.ToLower(tt.key), "password") || strings.Contains(strings.ToLower(tt.key), "token") {
				if result != `"***REDACTED***"` {
					t.Errorf("FormatValue(sensitive) = %q, want %q", result, `"***REDACTED***"`)
				}
			}
		})
	}
}

func TestFormatValue_PayloadLimit(t *testing.T) {
	opts := Options{
		PayloadMaxBytes: 10,
	}
	longString := strings.Repeat("a", 100)
	value := slog.StringValue(longString)

	result := FormatValue("key", value, opts)
	if !strings.Contains(result, "truncated") {
		t.Errorf("FormatValue(truncated) = %q, want to contain 'truncated'", result)
	}
	if !strings.Contains(result, "100") {
		t.Errorf("FormatValue(truncated) = %q, want to contain total bytes", result)
	}
}

func TestFormatConsoleValue_String(t *testing.T) {
	opts := Options{}
	value := slog.StringValue("test string")

	result := FormatConsoleValue("key", value, opts)
	if result != "test string" {
		t.Errorf("FormatConsoleValue(string) = %q, want %q", result, "test string")
	}
}

func TestFormatConsoleValue_Duration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"milliseconds", 500 * time.Millisecond, "500ms"},
		{"seconds", 5 * time.Second, "5s"},
		{"minutes", 2 * time.Minute, "2m0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{}
			value := slog.DurationValue(tt.duration)
			result := FormatConsoleValue("key", value, opts)
			if !strings.Contains(result, tt.want) {
				t.Errorf("FormatConsoleValue(duration) = %q, want to contain %q", result, tt.want)
			}
		})
	}
}

func TestFormatConsoleValue_SensitiveField(t *testing.T) {
	opts := Options{
		SensitiveFields: []string{"password"},
	}

	value := slog.StringValue("secret123")
	result := FormatConsoleValue("password", value, opts)

	if result != "***REDACTED***" {
		t.Errorf("FormatConsoleValue(sensitive) = %q, want %q", result, "***REDACTED***")
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level slog.Level
		want  string
	}{
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelInfo, "INFO"},
		{slog.LevelWarn, "WARN"},
		{slog.LevelError, "ERROR"},
		{slog.Level(100), "LEVEL(100)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			result := LevelString(tt.level)
			if result != tt.want {
				t.Errorf("LevelString(%v) = %q, want %q", tt.level, result, tt.want)
			}
		})
	}
}

func TestColorizeLevel(t *testing.T) {
	tests := []struct {
		level string
		want  string
	}{
		{"DEBUG", "\033[35mDEBUG\033[0m"},
		{"INFO", "\033[36mINFO\033[0m"},
		{"WARN", "\033[33mWARN\033[0m"},
		{"ERROR", "\033[31mERROR\033[0m"},
		{"UNKNOWN", "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			result := ColorizeLevel(tt.level)
			if result != tt.want {
				t.Errorf("ColorizeLevel(%q) = %q, want %q", tt.level, result, tt.want)
			}
		})
	}
}

func TestHandler_Concurrency(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewHandler(Options{
		Format:           "logfmt",
		Level:            slog.LevelInfo,
		DisableTimestamp: true,
	}, buf)

	const numGoroutines = 50
	const numLogsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numLogsPerGoroutine; j++ {
				handler.LogRecord(slog.LevelInfo, "message", []slog.Attr{
					slog.Int("id", id),
					slog.Int("iter", j),
				})
			}
		}(i)
	}

	wg.Wait()

	output := buf.String()
	lines := strings.Count(output, "\n")
	expectedLines := numGoroutines * numLogsPerGoroutine
	if lines != expectedLines {
		t.Errorf("Expected %d log lines, got %d", expectedLines, lines)
	}
}

func TestHandler_AttributeSorting(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewHandler(Options{
		Format:           "logfmt",
		Level:            slog.LevelInfo,
		DisableTimestamp: true,
	}, buf)

	attrs := []slog.Attr{
		slog.String("zebra", "value"),
		slog.String("apple", "value"),
		slog.String("banana", "value"),
	}

	handler.LogRecord(slog.LevelInfo, "test", attrs)

	output := buf.String()
	// Check that attributes are sorted
	appleIdx := strings.Index(output, "apple=")
	bananaIdx := strings.Index(output, "banana=")
	zebraIdx := strings.Index(output, "zebra=")

	if appleIdx == -1 || bananaIdx == -1 || zebraIdx == -1 {
		t.Fatal("All attributes should be present")
	}

	if appleIdx >= bananaIdx || bananaIdx >= zebraIdx {
		t.Error("Attributes should be sorted alphabetically")
	}
}

func TestHandler_WithAttrsChaining(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewHandler(Options{
		Format:           "logfmt",
		Level:            slog.LevelInfo,
		DisableTimestamp: true,
	}, buf)

	h1 := handler.WithAttrs([]slog.Attr{slog.String("key1", "value1")})
	h2 := h1.WithAttrs([]slog.Attr{slog.String("key2", "value2")})

	h2.(*Handler).LogRecord(slog.LevelInfo, "test", nil)

	output := buf.String()
	if !strings.Contains(output, "key1=\"value1\"") {
		t.Error("Output should contain key1=\"value1\" from first WithAttrs")
	}
	if !strings.Contains(output, "key2=\"value2\"") {
		t.Error("Output should contain key2=\"value2\" from second WithAttrs")
	}
}

func TestKVToAttrs_InvalidPairs(t *testing.T) {
	// Test with odd number of elements (last key without value)
	kv := []any{"key1", "value1", "key2"}
	attrs := KVToAttrs(kv)

	if len(attrs) != 1 {
		t.Errorf("Expected 1 attr (key2 ignored), got %d", len(attrs))
	}
	if attrs[0].Key != "key1" {
		t.Errorf("Expected key1, got %q", attrs[0].Key)
	}
}

func TestFormatValue_AllTypes(t *testing.T) {
	opts := Options{}
	now := time.Now()

	tests := []struct {
		name  string
		value slog.Value
		check func(string) bool
	}{
		{"string", slog.StringValue("test"), func(s string) bool { return strings.Contains(s, "test") }},
		{"int64", slog.Int64Value(42), func(s string) bool { return s == "42" }},
		{"uint64", slog.Uint64Value(100), func(s string) bool { return s == "100" }},
		{"float64", slog.Float64Value(3.14), func(s string) bool { return strings.Contains(s, "3.14") }},
		{"bool true", slog.BoolValue(true), func(s string) bool { return s == "true" }},
		{"bool false", slog.BoolValue(false), func(s string) bool { return s == "false" }},
		{"duration", slog.DurationValue(5 * time.Second), func(s string) bool { return s == "5000" }},
		{"time", slog.TimeValue(now), func(s string) bool { return strings.Contains(s, now.Format(time.RFC3339)) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatValue("key", tt.value, opts)
			if !tt.check(result) {
				t.Errorf("FormatValue(%s) = %q, check failed", tt.name, result)
			}
		})
	}
}

func TestFormatConsoleValue_AllTypes(t *testing.T) {
	opts := Options{}
	now := time.Now()

	tests := []struct {
		name  string
		value slog.Value
		check func(string) bool
	}{
		{"string", slog.StringValue("test"), func(s string) bool { return s == "test" }},
		{"int64", slog.Int64Value(42), func(s string) bool { return s == "42" }},
		{"uint64", slog.Uint64Value(100), func(s string) bool { return s == "100" }},
		{"float64", slog.Float64Value(3.14), func(s string) bool { return strings.Contains(s, "3.14") }},
		{"bool true", slog.BoolValue(true), func(s string) bool { return s == "true" }},
		{"bool false", slog.BoolValue(false), func(s string) bool { return s == "false" }},
		{"duration ms", slog.DurationValue(500 * time.Millisecond), func(s string) bool { return strings.Contains(s, "500ms") }},
		{"time", slog.TimeValue(now), func(s string) bool { return strings.Contains(s, now.Format("2006-01-02 15:04:05")) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatConsoleValue("key", tt.value, opts)
			if !tt.check(result) {
				t.Errorf("FormatConsoleValue(%s) = %q, check failed", tt.name, result)
			}
		})
	}
}

func TestFormatConsole_WithAttributes(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewHandler(Options{
		Format:           "console",
		Level:            slog.LevelInfo,
		DisableTimestamp: true,
		Color:            false,
	}, buf)

	attrs := []slog.Attr{
		slog.String("key1", "value1"),
		slog.String("key2", "value2"),
		slog.String("key3", "value3"),
	}

	handler.LogRecord(slog.LevelInfo, "test message", attrs)

	output := buf.String()
	// Check that attributes are on separate lines with indentation
	lines := strings.Split(output, "\n")
	hasIndentedAttr := false
	for _, line := range lines {
		if strings.HasPrefix(line, "        ") {
			hasIndentedAttr = true
			break
		}
	}
	if !hasIndentedAttr {
		t.Error("Console format should have indented attributes")
	}
}

func TestFormatConsole_NoAttributes(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewHandler(Options{
		Format:           "console",
		Level:            slog.LevelInfo,
		DisableTimestamp: true,
		Color:            false,
	}, buf)

	handler.LogRecord(slog.LevelInfo, "test message", nil)

	output := buf.String()
	// Should end with single newline
	if !strings.HasSuffix(output, "\n") {
		t.Error("Output should end with newline")
	}
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(lines))
	}
}

func TestHandler_NoTimestamp(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewHandler(Options{
		Format:           "logfmt",
		Level:            slog.LevelInfo,
		DisableTimestamp: true,
	}, buf)

	handler.LogRecord(slog.LevelInfo, "test", nil)

	output := buf.String()
	if strings.Contains(output, "time=") {
		t.Error("Output should not contain timestamp when DisableTimestamp is true")
	}
}

func TestHandler_WithTimestamp(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewHandler(Options{
		Format:           "logfmt",
		Level:            slog.LevelInfo,
		DisableTimestamp: false,
	}, buf)

	handler.LogRecord(slog.LevelInfo, "test", nil)

	output := buf.String()
	if !strings.Contains(output, "time=") {
		t.Error("Output should contain timestamp when DisableTimestamp is false")
	}
}

