package logx

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"go.eggybyte.com/egg/core/identity"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithWriter(&buf), WithColor(false))

	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "level=INFO") {
		t.Errorf("expected level=INFO, got: %s", output)
	}
	if !strings.Contains(output, `msg="test message"`) {
		t.Errorf("expected msg in output, got: %s", output)
	}
	// String values are now quoted
	if !strings.Contains(output, `key="value"`) {
		t.Errorf("expected key=\"value\" in output, got: %s", output)
	}
}

func TestFieldSorting(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithWriter(&buf), WithColor(false))

	logger.Info("test", "zebra", "z", "alpha", "a", "beta", "b")

	output := buf.String()
	// Check that fields are sorted: alpha < beta < zebra (values are quoted)
	alphaPos := strings.Index(output, `alpha="a"`)
	betaPos := strings.Index(output, `beta="b"`)
	zebraPos := strings.Index(output, `zebra="z"`)

	if alphaPos == -1 || betaPos == -1 || zebraPos == -1 {
		t.Fatalf("missing fields in output: %s", output)
	}

	if alphaPos > betaPos || betaPos > zebraPos {
		t.Errorf("fields not sorted correctly: alpha=%d, beta=%d, zebra=%d\nOutput: %s",
			alphaPos, betaPos, zebraPos, output)
	}
}

func TestColorization(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithWriter(&buf), WithColor(true))

	logger.Info("test")

	output := buf.String()
	// Should contain ANSI color codes for INFO level
	if !strings.Contains(output, "\033[") {
		t.Errorf("expected ANSI color codes in output, got: %s", output)
	}
}

func TestSensitiveFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(
		WithWriter(&buf),
		WithColor(false),
		WithSensitiveFields("password", "token"),
	)

	logger.Info("login", "password", "secret123")

	output := buf.String()
	if strings.Contains(output, "secret123") {
		t.Errorf("sensitive field not redacted: %s", output)
	}
	if !strings.Contains(output, "REDACTED") {
		t.Errorf("expected REDACTED in output: %s", output)
	}
}

func TestPayloadLimit(t *testing.T) {
	var buf bytes.Buffer
	logger := New(
		WithWriter(&buf),
		WithColor(false),
		WithPayloadLimit(10),
	)

	longString := strings.Repeat("a", 100)
	logger.Info("test", "data", longString)

	output := buf.String()
	if strings.Contains(output, longString) {
		t.Errorf("long payload not truncated: %s", output)
	}
	if !strings.Contains(output, "truncated") {
		t.Errorf("expected truncation marker in output: %s", output)
	}
}

func TestWithLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithWriter(&buf), WithColor(false))

	childLogger := logger.With("service", "test")
	childLogger.Info("message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, `service="test"`) {
		t.Errorf("expected service=\"test\" in output: %s", output)
	}
	if !strings.Contains(output, `key="value"`) {
		t.Errorf("expected key=\"value\" in output: %s", output)
	}
}

func TestErrorLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithWriter(&buf), WithColor(false))

	logger.Error(nil, "operation failed", "op", "test")

	output := buf.String()
	if !strings.Contains(output, "level=ERROR") {
		t.Errorf("expected level=ERROR in output: %s", output)
	}
	if !strings.Contains(output, `op="test"`) {
		t.Errorf("expected op=\"test\" in output: %s", output)
	}
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name     string
		logLevel slog.Level
		logFunc  func(logger *Logger)
		expected string
	}{
		{
			name:     "debug disabled at info level",
			logLevel: slog.LevelInfo,
			logFunc:  func(l *Logger) { l.Debug("debug msg") },
			expected: "",
		},
		{
			name:     "info enabled at info level",
			logLevel: slog.LevelInfo,
			logFunc:  func(l *Logger) { l.Info("info msg") },
			expected: "level=INFO",
		},
		{
			name:     "warn enabled at info level",
			logLevel: slog.LevelInfo,
			logFunc:  func(l *Logger) { l.Warn("warn msg") },
			expected: "level=WARN",
		},
		{
			name:     "error enabled at info level",
			logLevel: slog.LevelInfo,
			logFunc:  func(l *Logger) { l.Error(nil, "error msg") },
			expected: "level=ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := New(
				WithWriter(&buf),
				WithColor(false),
				WithLevel(tt.logLevel),
			).(*Logger)

			tt.logFunc(logger)

			output := buf.String()
			if tt.expected == "" {
				if output != "" {
					t.Errorf("expected no output, got: %s", output)
				}
			} else {
				if !strings.Contains(output, tt.expected) {
					t.Errorf("expected %q in output, got: %s", tt.expected, output)
				}
			}
		})
	}
}

func TestFromContext(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := New(WithWriter(&buf), WithColor(false))

	ctx := context.Background()
	ctx = identity.WithUser(ctx, &identity.UserInfo{UserID: "u-123"})
	ctx = identity.WithMeta(ctx, &identity.RequestMeta{RequestID: "req-abc"})

	ctxLogger := FromContext(ctx, baseLogger)
	ctxLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, `user_id="u-123"`) {
		t.Errorf("expected user_id in output: %s", output)
	}
	if !strings.Contains(output, `request_id="req-abc"`) {
		t.Errorf("expected request_id in output: %s", output)
	}
}

func BenchmarkLogger(b *testing.B) {
	logger := New(WithWriter(&bytes.Buffer{}), WithColor(false))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark", "iteration", i)
	}
}

func BenchmarkLoggerWithSorting(b *testing.B) {
	logger := New(WithWriter(&bytes.Buffer{}), WithColor(false))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark",
			"z_field", "z",
			"m_field", "m",
			"a_field", "a",
		)
	}
}
