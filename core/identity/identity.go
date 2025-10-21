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
func WithUser(ctx context.Context, u *UserInfo) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// UserFrom retrieves user information from the context.
// Returns the user info and a boolean indicating if it was found.
func UserFrom(ctx context.Context) (*UserInfo, bool) {
	u, ok := ctx.Value(userKey).(*UserInfo)
	return u, ok
}

// WithMeta stores request metadata in the context.
// Returns a new context with the metadata attached.
func WithMeta(ctx context.Context, m *RequestMeta) context.Context {
	return context.WithValue(ctx, metaKey, m)
}

// MetaFrom retrieves request metadata from the context.
// Returns the metadata and a boolean indicating if it was found.
func MetaFrom(ctx context.Context) (*RequestMeta, bool) {
	m, ok := ctx.Value(metaKey).(*RequestMeta)
	return m, ok
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
