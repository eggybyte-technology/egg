// Package internal provides tests for health check endpoints.
package internal

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.eggybyte.com/egg/logx"
)

// TestSetupHealthEndpoints tests health endpoint registration.
func TestSetupHealthEndpoints(t *testing.T) {
	logger := logx.New()
	mux := http.NewServeMux()

	SetupHealthEndpoints(mux, logger)

	// Verify endpoints are registered by making test requests
	endpoints := []string{"/health", "/ready", "/live"}
	for _, endpoint := range endpoints {
		t.Run("endpoint_"+endpoint, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("GET %s returned %d, want %d", endpoint, w.Code, http.StatusOK)
			}
		})
	}
}

// TestHealthEndpoint tests the /health endpoint.
func TestHealthEndpoint(t *testing.T) {
	logger := logx.New()
	mux := http.NewServeMux()
	SetupHealthEndpoints(mux, logger)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET request",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"healthy"}`,
		},
		{
			name:           "HEAD request",
			method:         http.MethodHead,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "POST request (not allowed)",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
		{
			name:           "PUT request (not allowed)",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
		{
			name:           "DELETE request (not allowed)",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s /health returned %d, want %d", tt.method, w.Code, tt.expectedStatus)
			}

			if tt.expectedBody != "" && !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("%s /health body = %q, want to contain %q", tt.method, w.Body.String(), tt.expectedBody)
			}
		})
	}
}

// TestReadyEndpoint tests the /ready endpoint.
func TestReadyEndpoint(t *testing.T) {
	logger := logx.New()
	mux := http.NewServeMux()
	SetupHealthEndpoints(mux, logger)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET request",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ready"}`,
		},
		{
			name:           "HEAD request",
			method:         http.MethodHead,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "POST request (not allowed)",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/ready", nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s /ready returned %d, want %d", tt.method, w.Code, tt.expectedStatus)
			}

			if tt.expectedBody != "" && !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("%s /ready body = %q, want to contain %q", tt.method, w.Body.String(), tt.expectedBody)
			}
		})
	}
}

// TestLiveEndpoint tests the /live endpoint.
func TestLiveEndpoint(t *testing.T) {
	logger := logx.New()
	mux := http.NewServeMux()
	SetupHealthEndpoints(mux, logger)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET request",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"alive"}`,
		},
		{
			name:           "HEAD request",
			method:         http.MethodHead,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "POST request (not allowed)",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/live", nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s /live returned %d, want %d", tt.method, w.Code, tt.expectedStatus)
			}

			if tt.expectedBody != "" && !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("%s /live body = %q, want to contain %q", tt.method, w.Body.String(), tt.expectedBody)
			}
		})
	}
}

// TestMultipleEndpointsCoexist tests that all endpoints work together.
func TestMultipleEndpointsCoexist(t *testing.T) {
	logger := logx.New()
	mux := http.NewServeMux()
	SetupHealthEndpoints(mux, logger)

	endpoints := []struct {
		path string
		want string
	}{
		{"/health", "healthy"},
		{"/ready", "ready"},
		{"/live", "alive"},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, endpoint.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("GET %s returned %d, want %d", endpoint.path, w.Code, http.StatusOK)
			}

			if !strings.Contains(w.Body.String(), endpoint.want) {
				t.Errorf("GET %s body = %q, want to contain %q", endpoint.path, w.Body.String(), endpoint.want)
			}
		})
	}
}

// TestHealthEndpointsResponseFormat tests JSON response format.
func TestHealthEndpointsResponseFormat(t *testing.T) {
	logger := logx.New()
	mux := http.NewServeMux()
	SetupHealthEndpoints(mux, logger)

	tests := []struct {
		path         string
		expectedJSON string
	}{
		{"/health", `{"status":"healthy"}`},
		{"/ready", `{"status":"ready"}`},
		{"/live", `{"status":"alive"}`},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			body := strings.TrimSpace(w.Body.String())
			if body != tt.expectedJSON {
				t.Errorf("GET %s body = %q, want %q", tt.path, body, tt.expectedJSON)
			}
		})
	}
}


