// Package identity provides user identity and request metadata context management.
//
// Overview:
//   - Responsibility: Store and retrieve user identity and request metadata from context
//   - Key Types: UserInfo for user data, RequestMeta for request metadata
//   - Concurrency Model: All functions are safe for concurrent use
//   - Error Semantics: Functions return boolean to indicate presence of data
//   - Performance Notes: Minimal allocations, context-based storage
//
// Usage:
//
//	ctx := identity.WithUser(ctx, &identity.UserInfo{UserID: "123"})
//	user, ok := identity.UserFrom(ctx)
//	ctx = identity.WithMeta(ctx, &identity.RequestMeta{RequestID: "req-123"})
package identity

import (
	"context"
	"crypto/subtle"

	"go.eggybyte.com/egg/core/errors"
)

// UserInfo contains user identity information.
// This is a container for user data without authentication logic.
type UserInfo struct {
	UserID   string   // Unique user identifier
	UserName string   // Human-readable user name
	Roles    []string // User roles/permissions
}

// RequestMeta contains request metadata information.
type RequestMeta struct {
	RequestID     string // Unique request identifier for tracing
	InternalToken string // Internal service token
	RemoteIP      string // Client IP address
	UserAgent     string // Client user agent string
}

type contextKey string

const (
	userKey contextKey = "user"
	metaKey contextKey = "meta"
)

// WithUser stores user information in the context.
// Returns a new context with the user information attached.
// If u is nil, returns the context unchanged.
func WithUser(ctx context.Context, u *UserInfo) context.Context {
	if u == nil {
		return ctx
	}
	return context.WithValue(ctx, userKey, u)
}

// UserFrom retrieves user information from the context.
// Returns the user info and a boolean indicating if it was found.
func UserFrom(ctx context.Context) (*UserInfo, bool) {
	u, ok := ctx.Value(userKey).(*UserInfo)
	if !ok || u == nil {
		return nil, false
	}
	return u, true
}

// WithMeta stores request metadata in the context.
// Returns a new context with the metadata attached.
// If m is nil, returns the context unchanged.
func WithMeta(ctx context.Context, m *RequestMeta) context.Context {
	if m == nil {
		return ctx
	}
	return context.WithValue(ctx, metaKey, m)
}

// MetaFrom retrieves request metadata from the context.
// Returns the metadata and a boolean indicating if it was found.
func MetaFrom(ctx context.Context) (*RequestMeta, bool) {
	m, ok := ctx.Value(metaKey).(*RequestMeta)
	if !ok || m == nil {
		return nil, false
	}
	return m, true
}

// HasRole checks if the user in the context has the specified role.
// Returns false if no user is found in the context.
func HasRole(ctx context.Context, role string) bool {
	user, ok := UserFrom(ctx)
	if !ok {
		return false
	}

	for _, userRole := range user.Roles {
		if userRole == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the user in the context has any of the specified roles.
// Returns false if no user is found in the context.
func HasAnyRole(ctx context.Context, roles ...string) bool {
	user, ok := UserFrom(ctx)
	if !ok {
		return false
	}

	for _, userRole := range user.Roles {
		for _, role := range roles {
			if userRole == role {
				return true
			}
		}
	}
	return false
}

// IsInternalService checks if the request is from an internal service.
// This is determined by checking if the internal token matches the expected service name.
func IsInternalService(ctx context.Context, serviceName string) bool {
	meta, ok := MetaFrom(ctx)
	if !ok {
		return false
	}

	// Simple check: if internal token is not empty and matches service name
	return meta.InternalToken != "" && meta.InternalToken == serviceName
}

// RequireInternalToken validates the internal token from context against expected token.
// Returns error if token is missing or invalid. Uses constant-time comparison to prevent timing attacks.
//
// Parameters:
//   - ctx: context containing request metadata
//   - expectedToken: the expected internal token value
//
// Returns:
//   - error: nil if token is valid; CodeUnauthenticated if missing or invalid
//
// Security:
//   - Uses crypto/subtle for constant-time string comparison
//   - Prevents timing attack vulnerabilities
func RequireInternalToken(ctx context.Context, expectedToken string) error {
	if expectedToken == "" {
		// If no token is configured, skip validation
		return nil
	}

	meta, ok := MetaFrom(ctx)
	if !ok || meta.InternalToken == "" {
		return errors.New(errors.CodeUnauthenticated, "internal token required")
	}

	// Use constant-time comparison for security
	if !secureCompare(meta.InternalToken, expectedToken) {
		return errors.New(errors.CodeUnauthenticated, "invalid internal token")
	}

	return nil
}

// secureCompare performs constant-time string comparison to prevent timing attacks.
func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
