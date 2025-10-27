// Package httpx provides HTTP utilities for binding, validation, and security.
//
// Overview:
//   - Responsibility: HTTP request/response helpers, security headers, error responses
//   - Key Types: BindOptions for configuration, standard error responses
//   - Concurrency Model: All functions are safe for concurrent use
//   - Error Semantics: Functions return errors for binding and validation failures
//   - Performance Notes: Efficient JSON parsing and validation

// Usage:
//
//	var req UserRequest
//	if err := httpx.BindAndValidate(r, &req); err != nil {
//	  httpx.WriteError(w, err, http.StatusBadRequest)
//	  return
//	}
package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"go.eggybyte.com/egg/httpx/internal"
)

// ErrorResponse represents a standard JSON error response.
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// BindAndValidate binds JSON request body to target struct and validates it.
// The target struct should have `json` and `validate` tags.
func BindAndValidate(r *http.Request, target any) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}

	// Decode JSON
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate
	validate := validator.New()
	if err := validate.Struct(target); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// WriteJSON writes a JSON response.
func WriteJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if data == nil {
		return nil
	}

	encoder := json.NewEncoder(w)
	return encoder.Encode(data)
}

// WriteError writes a standard error response.
func WriteError(w http.ResponseWriter, err error, status int) error {
	response := ErrorResponse{
		Error:   http.StatusText(status),
		Message: err.Error(),
	}

	return WriteJSON(w, status, response)
}

// NotFoundHandler returns a standard 404 JSON response.
func NotFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := ErrorResponse{
			Error:   "Not Found",
			Message: fmt.Sprintf("Path %s not found", r.URL.Path),
		}
		WriteJSON(w, http.StatusNotFound, response)
	}
}

// MethodNotAllowedHandler returns a standard 405 JSON response.
func MethodNotAllowedHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := ErrorResponse{
			Error:   "Method Not Allowed",
			Message: fmt.Sprintf("Method %s not allowed for %s", r.Method, r.URL.Path),
		}
		WriteJSON(w, http.StatusMethodNotAllowed, response)
	}
}

// SecurityHeaders adds security headers to HTTP response.
// These are sensible defaults for production environments.
type SecurityHeaders struct {
	ContentTypeOptions    bool   // X-Content-Type-Options: nosniff
	FrameOptions          bool   // X-Frame-Options: DENY
	ReferrerPolicy        bool   // Referrer-Policy: no-referrer
	StrictTransportSec    bool   // Strict-Transport-Security (HSTS)
	HSTSMaxAge            int    // Max age for HSTS in seconds (default: 31536000 = 1 year)
	ContentSecurityPolicy string // Optional CSP header
}

// DefaultSecurityHeaders returns security headers with sensible defaults.
func DefaultSecurityHeaders() SecurityHeaders {
	return SecurityHeaders{
		ContentTypeOptions: true,
		FrameOptions:       true,
		ReferrerPolicy:     true,
		StrictTransportSec: false, // Disable by default (should be set at load balancer/ingress)
		HSTSMaxAge:         31536000,
	}
}

// SecureMiddleware adds security headers to responses.
func SecureMiddleware(headers SecurityHeaders) func(http.Handler) http.Handler {
	internalHeaders := internal.SecurityHeaders{
		ContentTypeOptions:    headers.ContentTypeOptions,
		FrameOptions:          headers.FrameOptions,
		ReferrerPolicy:        headers.ReferrerPolicy,
		StrictTransportSec:    headers.StrictTransportSec,
		HSTSMaxAge:            headers.HSTSMaxAge,
		ContentSecurityPolicy: headers.ContentSecurityPolicy,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			internal.ApplySecurityHeaders(w, internalHeaders)
			next.ServeHTTP(w, r)
		})
	}
}

// CORSOptions configures CORS behavior.
type CORSOptions struct {
	AllowedOrigins   []string // Allowed origins (e.g., ["https://example.com"])
	AllowedMethods   []string // Allowed methods (default: GET, POST, PUT, DELETE, OPTIONS)
	AllowedHeaders   []string // Allowed headers (default: Content-Type, Authorization)
	ExposedHeaders   []string // Exposed headers
	AllowCredentials bool     // Allow credentials
	MaxAge           int      // Preflight cache duration in seconds
}

// DefaultCORSOptions returns CORS options with sensible defaults.
func DefaultCORSOptions() CORSOptions {
	return CORSOptions{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		MaxAge:         3600,
	}
}

// CORSMiddleware adds CORS headers to responses.
func CORSMiddleware(opts CORSOptions) func(http.Handler) http.Handler {
	internalOpts := internal.CORSOptions{
		AllowedOrigins:   opts.AllowedOrigins,
		AllowedMethods:   opts.AllowedMethods,
		AllowedHeaders:   opts.AllowedHeaders,
		ExposedHeaders:   opts.ExposedHeaders,
		AllowCredentials: opts.AllowCredentials,
		MaxAge:           opts.MaxAge,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			internal.ApplyCORSHeaders(w, r, internalOpts)

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
