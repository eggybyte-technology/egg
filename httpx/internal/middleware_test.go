// Package internal provides tests for httpx internal middleware.
package internal

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApplySecurityHeaders_AllEnabled(t *testing.T) {
	w := httptest.NewRecorder()

	headers := SecurityHeaders{
		ContentTypeOptions:    true,
		FrameOptions:          true,
		ReferrerPolicy:        true,
		StrictTransportSec:    true,
		HSTSMaxAge:            31536000,
		ContentSecurityPolicy: "default-src 'self'",
	}

	ApplySecurityHeaders(w, headers)

	result := w.Header()

	if result.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", result.Get("X-Content-Type-Options"), "nosniff")
	}
	if result.Get("X-Frame-Options") != "DENY" {
		t.Errorf("X-Frame-Options = %q, want %q", result.Get("X-Frame-Options"), "DENY")
	}
	if result.Get("Referrer-Policy") != "no-referrer" {
		t.Errorf("Referrer-Policy = %q, want %q", result.Get("Referrer-Policy"), "no-referrer")
	}
	hstsValue := result.Get("Strict-Transport-Security")
	expectedHSTS := "max-age=31536000; includeSubDomains"
	if hstsValue != expectedHSTS {
		t.Errorf("Strict-Transport-Security = %q, want %q", hstsValue, expectedHSTS)
	}
	if result.Get("Content-Security-Policy") != "default-src 'self'" {
		t.Errorf("Content-Security-Policy = %q, want %q", result.Get("Content-Security-Policy"), "default-src 'self'")
	}
}

func TestApplySecurityHeaders_NoneEnabled(t *testing.T) {
	w := httptest.NewRecorder()

	headers := SecurityHeaders{
		ContentTypeOptions:    false,
		FrameOptions:          false,
		ReferrerPolicy:        false,
		StrictTransportSec:    false,
		ContentSecurityPolicy: "",
	}

	ApplySecurityHeaders(w, headers)

	result := w.Header()

	if result.Get("X-Content-Type-Options") != "" {
		t.Error("X-Content-Type-Options should not be set")
	}
	if result.Get("X-Frame-Options") != "" {
		t.Error("X-Frame-Options should not be set")
	}
	if result.Get("Referrer-Policy") != "" {
		t.Error("Referrer-Policy should not be set")
	}
	if result.Get("Strict-Transport-Security") != "" {
		t.Error("Strict-Transport-Security should not be set")
	}
	if result.Get("Content-Security-Policy") != "" {
		t.Error("Content-Security-Policy should not be set")
	}
}

func TestApplySecurityHeaders_Partial(t *testing.T) {
	w := httptest.NewRecorder()

	headers := SecurityHeaders{
		ContentTypeOptions: true,
		FrameOptions:      false,
		HSTSMaxAge:        3600,
	}

	ApplySecurityHeaders(w, headers)

	result := w.Header()

	if result.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("X-Content-Type-Options should be set")
	}
	if result.Get("X-Frame-Options") != "" {
		t.Error("X-Frame-Options should not be set")
	}
}

func TestApplyCORSHeaders_AllowedOrigin(t *testing.T) {
	tests := []struct {
		name          string
		origin        string
		allowedOrigins []string
		wantAllowed   bool
		wantHeader    string
	}{
		{
			name:          "exact match",
			origin:        "https://example.com",
			allowedOrigins: []string{"https://example.com"},
			wantAllowed:   true,
			wantHeader:    "https://example.com",
		},
		{
			name:          "wildcard",
			origin:        "https://example.com",
			allowedOrigins: []string{"*"},
			wantAllowed:   true,
			wantHeader:    "*",
		},
		{
			name:          "not allowed",
			origin:        "https://evil.com",
			allowedOrigins: []string{"https://example.com"},
			wantAllowed:   false,
			wantHeader:    "",
		},
		{
			name:          "multiple origins",
			origin:        "https://example.com",
			allowedOrigins: []string{"https://example.com", "https://other.com"},
			wantAllowed:   true,
			wantHeader:    "https://example.com",
		},
		{
			name:          "no origin header",
			origin:        "",
			allowedOrigins: []string{"https://example.com"},
			wantAllowed:   false,
			wantHeader:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			opts := CORSOptions{
				AllowedOrigins: tt.allowedOrigins,
			}

			allowed := ApplyCORSHeaders(w, req, opts)

			if allowed != tt.wantAllowed {
				t.Errorf("ApplyCORSHeaders() allowed = %v, want %v", allowed, tt.wantAllowed)
			}

			if tt.wantAllowed {
				headerValue := w.Header().Get("Access-Control-Allow-Origin")
				if headerValue != tt.wantHeader {
					t.Errorf("Access-Control-Allow-Origin = %q, want %q", headerValue, tt.wantHeader)
				}
			}
		})
	}
}

func TestApplyCORSHeaders_AllowedMethods(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")

	opts := CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST", "PUT"},
	}

	ApplyCORSHeaders(w, req, opts)

	methods := w.Header().Get("Access-Control-Allow-Methods")
	if methods != "GET, POST, PUT" {
		t.Errorf("Access-Control-Allow-Methods = %q, want %q", methods, "GET, POST, PUT")
	}
}

func TestApplyCORSHeaders_AllowedHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")

	opts := CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}

	ApplyCORSHeaders(w, req, opts)

	headers := w.Header().Get("Access-Control-Allow-Headers")
	if headers != "Content-Type, Authorization" {
		t.Errorf("Access-Control-Allow-Headers = %q, want %q", headers, "Content-Type, Authorization")
	}
}

func TestApplyCORSHeaders_ExposedHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")

	opts := CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
		ExposedHeaders: []string{"X-Custom-Header"},
	}

	ApplyCORSHeaders(w, req, opts)

	headers := w.Header().Get("Access-Control-Expose-Headers")
	if headers != "X-Custom-Header" {
		t.Errorf("Access-Control-Expose-Headers = %q, want %q", headers, "X-Custom-Header")
	}
}

func TestApplyCORSHeaders_AllowCredentials(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")

	opts := CORSOptions{
		AllowedOrigins:   []string{"https://example.com"},
		AllowCredentials: true,
	}

	ApplyCORSHeaders(w, req, opts)

	credential := w.Header().Get("Access-Control-Allow-Credentials")
	if credential != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want %q", credential, "true")
	}
}

func TestApplyCORSHeaders_MaxAge(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")

	opts := CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
		MaxAge:         3600,
	}

	ApplyCORSHeaders(w, req, opts)

	maxAge := w.Header().Get("Access-Control-Max-Age")
	if maxAge != "3600" {
		t.Errorf("Access-Control-Max-Age = %q, want %q", maxAge, "3600")
	}
}

func TestApplyCORSHeaders_ZeroMaxAge(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")

	opts := CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
		MaxAge:         0,
	}

	ApplyCORSHeaders(w, req, opts)

	maxAge := w.Header().Get("Access-Control-Max-Age")
	if maxAge != "" {
		t.Errorf("Access-Control-Max-Age = %q, want empty", maxAge)
	}
}

func TestApplyCORSHeaders_EmptyMethods(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")

	opts := CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{},
	}

	ApplyCORSHeaders(w, req, opts)

	methods := w.Header().Get("Access-Control-Allow-Methods")
	if methods != "" {
		t.Errorf("Access-Control-Allow-Methods = %q, want empty", methods)
	}
}

func TestApplyCORSHeaders_EmptyHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")

	opts := CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
		AllowedHeaders: []string{},
	}

	ApplyCORSHeaders(w, req, opts)

	headers := w.Header().Get("Access-Control-Allow-Headers")
	if headers != "" {
		t.Errorf("Access-Control-Allow-Headers = %q, want empty", headers)
	}
}

func TestApplyCORSHeaders_EmptyExposedHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")

	opts := CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
		ExposedHeaders: []string{},
	}

	ApplyCORSHeaders(w, req, opts)

	headers := w.Header().Get("Access-Control-Expose-Headers")
	if headers != "" {
		t.Errorf("Access-Control-Expose-Headers = %q, want empty", headers)
	}
}

func TestApplyCORSHeaders_AllOptions(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")

	opts := CORSOptions{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type"},
		ExposedHeaders:   []string{"X-Custom"},
		AllowCredentials: true,
		MaxAge:           3600,
	}

	allowed := ApplyCORSHeaders(w, req, opts)
	if !allowed {
		t.Error("ApplyCORSHeaders() should return true for allowed origin")
	}

	result := w.Header()
	if result.Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Error("Access-Control-Allow-Origin should be set")
	}
	if result.Get("Access-Control-Allow-Methods") != "GET, POST" {
		t.Error("Access-Control-Allow-Methods should be set")
	}
	if result.Get("Access-Control-Allow-Headers") != "Content-Type" {
		t.Error("Access-Control-Allow-Headers should be set")
	}
	if result.Get("Access-Control-Expose-Headers") != "X-Custom" {
		t.Error("Access-Control-Expose-Headers should be set")
	}
	if result.Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("Access-Control-Allow-Credentials should be set")
	}
	if result.Get("Access-Control-Max-Age") != "3600" {
		t.Error("Access-Control-Max-Age should be set")
	}
}

