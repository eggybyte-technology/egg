package httpx

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type TestRequest struct {
	Name  string `json:"name" validate:"required,min=2"`
	Email string `json:"email" validate:"required,email"`
}

func TestBindAndValidate(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		expectError bool
	}{
		{
			name:        "valid request",
			body:        `{"name":"John Doe","email":"john@example.com"}`,
			expectError: false,
		},
		{
			name:        "missing name",
			body:        `{"email":"john@example.com"}`,
			expectError: true,
		},
		{
			name:        "invalid email",
			body:        `{"name":"John","email":"invalid"}`,
			expectError: true,
		},
		{
			name:        "name too short",
			body:        `{"name":"J","email":"john@example.com"}`,
			expectError: true,
		},
		{
			name:        "invalid JSON",
			body:        `{"name":`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			var target TestRequest
			err := BindAndValidate(req, &target)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"status": "ok"}

	err := WriteJSON(w, http.StatusOK, data)
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected application/json content type, got %s", contentType)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", response["status"])
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	err := bytes.ErrTooLarge

	WriteError(w, err, http.StatusBadRequest)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if response.Error == "" {
		t.Error("expected error field to be set")
	}
}

func TestNotFoundHandler(t *testing.T) {
	handler := NotFoundHandler()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error != "Not Found" {
		t.Errorf("expected 'Not Found', got %s", response.Error)
	}
}

func TestMethodNotAllowedHandler(t *testing.T) {
	handler := MethodNotAllowedHandler()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/resource", nil)

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error != "Method Not Allowed" {
		t.Errorf("expected 'Method Not Allowed', got %s", response.Error)
	}
}

func TestSecureMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	headers := DefaultSecurityHeaders()
	middleware := SecureMiddleware(headers)
	wrappedHandler := middleware(handler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	wrappedHandler.ServeHTTP(w, req)

	// Check security headers
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("expected X-Content-Type-Options header")
	}

	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("expected X-Frame-Options header")
	}

	if w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Error("expected Referrer-Policy header")
	}

	// HSTS should NOT be set by default
	if w.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS should not be set by default")
	}
}

func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	opts := CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
		MaxAge:         3600,
	}

	middleware := CORSMiddleware(opts)
	wrappedHandler := middleware(handler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")

	wrappedHandler.ServeHTTP(w, req)

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}

	if !strings.Contains(w.Header().Get("Access-Control-Allow-Methods"), "GET") {
		t.Error("expected GET in Access-Control-Allow-Methods")
	}
}

func TestCORSPreflightRequest(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called for preflight")
	})

	opts := DefaultCORSOptions()
	middleware := CORSMiddleware(opts)
	wrappedHandler := middleware(handler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for preflight, got %d", w.Code)
	}
}
