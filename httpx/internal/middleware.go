// Package internal provides internal implementation details for httpx.
package internal

import (
	"fmt"
	"net/http"
	"strings"
)

// SecurityHeaders adds security headers to HTTP response.
type SecurityHeaders struct {
	ContentTypeOptions    bool
	FrameOptions          bool
	ReferrerPolicy        bool
	StrictTransportSec    bool
	HSTSMaxAge            int
	ContentSecurityPolicy string
}

// ApplySecurityHeaders applies security headers to the response writer.
func ApplySecurityHeaders(w http.ResponseWriter, headers SecurityHeaders) {
	if headers.ContentTypeOptions {
		w.Header().Set("X-Content-Type-Options", "nosniff")
	}

	if headers.FrameOptions {
		w.Header().Set("X-Frame-Options", "DENY")
	}

	if headers.ReferrerPolicy {
		w.Header().Set("Referrer-Policy", "no-referrer")
	}

	if headers.StrictTransportSec {
		hstsValue := fmt.Sprintf("max-age=%d; includeSubDomains", headers.HSTSMaxAge)
		w.Header().Set("Strict-Transport-Security", hstsValue)
	}

	if headers.ContentSecurityPolicy != "" {
		w.Header().Set("Content-Security-Policy", headers.ContentSecurityPolicy)
	}
}

// CORSOptions configures CORS behavior.
type CORSOptions struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// ApplyCORSHeaders applies CORS headers to the response writer.
func ApplyCORSHeaders(w http.ResponseWriter, r *http.Request, opts CORSOptions) bool {
	origin := r.Header.Get("Origin")

	// Check if origin is allowed
	allowed := false
	for _, allowedOrigin := range opts.AllowedOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			allowed = true
			break
		}
	}

	if !allowed {
		return false
	}

	if len(opts.AllowedOrigins) == 1 && opts.AllowedOrigins[0] == "*" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}

	if len(opts.AllowedMethods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(opts.AllowedMethods, ", "))
	}

	if len(opts.AllowedHeaders) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(opts.AllowedHeaders, ", "))
	}

	if len(opts.ExposedHeaders) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(opts.ExposedHeaders, ", "))
	}

	if opts.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	if opts.MaxAge > 0 {
		w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", opts.MaxAge))
	}

	return true
}

