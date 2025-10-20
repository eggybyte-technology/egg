// Package connectx provides tests for Connect interceptors and identity injection.
package connectx

import (
	"testing"

	"github.com/eggybyte-technology/egg/core/log"
	"github.com/eggybyte-technology/egg/obsx"
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

func TestDefaultHeaderMapping(t *testing.T) {
	mapping := DefaultHeaderMapping()

	if mapping.RequestID != "X-Request-Id" {
		t.Errorf("RequestID = %v, want X-Request-Id", mapping.RequestID)
	}

	if mapping.InternalToken != "X-Internal-Token" {
		t.Errorf("InternalToken = %v, want X-Internal-Token", mapping.InternalToken)
	}

	if mapping.UserID != "X-User-Id" {
		t.Errorf("UserID = %v, want X-User-Id", mapping.UserID)
	}

	if mapping.UserName != "X-User-Name" {
		t.Errorf("UserName = %v, want X-User-Name", mapping.UserName)
	}

	if mapping.Tenant != "X-User-Tenant" {
		t.Errorf("Tenant = %v, want X-User-Tenant", mapping.Tenant)
	}

	if mapping.Roles != "X-User-Roles" {
		t.Errorf("Roles = %v, want X-User-Roles", mapping.Roles)
	}

	if mapping.RealIP != "X-Real-IP" {
		t.Errorf("RealIP = %v, want X-Real-IP", mapping.RealIP)
	}

	if mapping.ForwardedFor != "X-Forwarded-For" {
		t.Errorf("ForwardedFor = %v, want X-Forwarded-For", mapping.ForwardedFor)
	}

	if mapping.UserAgent != "User-Agent" {
		t.Errorf("UserAgent = %v, want User-Agent", mapping.UserAgent)
	}
}

func TestDefaultInterceptors(t *testing.T) {
	logger := &testLogger{}

	tests := []struct {
		name string
		opts Options
	}{
		{
			name: "basic options",
			opts: Options{
				Logger:            logger,
				SlowRequestMillis: 1000,
			},
		},
		{
			name: "with custom headers",
			opts: Options{
				Logger: logger,
				Headers: HeaderMapping{
					RequestID:     "X-Custom-Request-Id",
					InternalToken: "X-Custom-Token",
					UserID:        "X-Custom-User-Id",
					UserName:      "X-Custom-User-Name",
					Tenant:        "X-Custom-Tenant",
					Roles:         "X-Custom-Roles",
					RealIP:        "X-Custom-Real-IP",
					ForwardedFor:  "X-Custom-Forwarded-For",
					UserAgent:     "X-Custom-User-Agent",
				},
				SlowRequestMillis: 500,
			},
		},
		{
			name: "with payload accounting",
			opts: Options{
				Logger:            logger,
				PayloadAccounting: true,
				SlowRequestMillis: 2000,
			},
		},
		{
			name: "with request/response body logging",
			opts: Options{
				Logger:            logger,
				WithRequestBody:   true,
				WithResponseBody:  true,
				SlowRequestMillis: 1500,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptors := DefaultInterceptors(tt.opts)

			if len(interceptors) == 0 {
				t.Error("Expected non-empty interceptors slice")
			}

			// Verify we have multiple interceptors
			if len(interceptors) < 3 {
				t.Errorf("Expected at least 3 interceptors, got %d", len(interceptors))
			}
		})
	}
}

func TestDefaultInterceptors_WithOtel(t *testing.T) {
	logger := &testLogger{}

	// Create a minimal otel provider for testing
	// Note: We don't actually initialize it fully in tests
	var otel *obsx.Provider = nil

	opts := Options{
		Logger:            logger,
		Otel:              otel,
		SlowRequestMillis: 1000,
	}

	interceptors := DefaultInterceptors(opts)

	if len(interceptors) == 0 {
		t.Error("Expected non-empty interceptors slice")
	}
}

func TestDefaultInterceptors_DefaultSlowThreshold(t *testing.T) {
	logger := &testLogger{}

	opts := Options{
		Logger: logger,
		// Don't set SlowRequestMillis to test default
	}

	interceptors := DefaultInterceptors(opts)

	if len(interceptors) == 0 {
		t.Error("Expected non-empty interceptors slice")
	}
}

func TestDefaultInterceptors_EmptyHeaders(t *testing.T) {
	logger := &testLogger{}

	opts := Options{
		Logger: logger,
		// Don't set Headers to test default
	}

	interceptors := DefaultInterceptors(opts)

	if len(interceptors) == 0 {
		t.Error("Expected non-empty interceptors slice")
	}
}

func TestOptions(t *testing.T) {
	logger := &testLogger{}

	opts := Options{
		Logger:            logger,
		Headers:           DefaultHeaderMapping(),
		WithRequestBody:   true,
		WithResponseBody:  true,
		SlowRequestMillis: 1000,
		PayloadAccounting: true,
	}

	if opts.Logger == nil {
		t.Error("Logger should not be nil")
	}

	if opts.Headers.RequestID != "X-Request-Id" {
		t.Error("Headers should be set")
	}

	if !opts.WithRequestBody {
		t.Error("WithRequestBody should be true")
	}

	if !opts.WithResponseBody {
		t.Error("WithResponseBody should be true")
	}

	if opts.SlowRequestMillis != 1000 {
		t.Errorf("SlowRequestMillis = %v, want 1000", opts.SlowRequestMillis)
	}

	if !opts.PayloadAccounting {
		t.Error("PayloadAccounting should be true")
	}
}
