# 🔐 Identity Package

The `identity` package provides user identity and request metadata context management for the EggyByte framework.

## Overview

This package offers a lightweight, context-based approach to managing user identity and request metadata throughout the request lifecycle. It's designed to be zero-dependency and highly performant.

## Features

- **Context-based storage** - Uses Go's context for thread-safe identity management
- **Zero dependencies** - No external dependencies, pure Go
- **Type-safe** - Strongly typed user information and metadata
- **Permission helpers** - Built-in role checking utilities
- **Request metadata** - Comprehensive request context information

## Quick Start

```go
import "go.eggybyte.com/egg/core/identity"

// Store user information in context
ctx := identity.WithUser(ctx, &identity.UserInfo{
    UserID:   "user-123",
    UserName: "john.doe",
    Roles:    []string{"admin", "user"},
})

// Retrieve user information
if user, ok := identity.UserFrom(ctx); ok {
    fmt.Printf("User: %s (%s)\n", user.UserName, user.UserID)
}

// Check permissions
if identity.HasRole(ctx, "admin") {
    // User has admin role
}
```

## API Reference

### Types

#### UserInfo

```go
type UserInfo struct {
    UserID   string   // Unique user identifier
    UserName string   // Human-readable user name
    Roles    []string // User roles/permissions
}
```

#### RequestMeta

```go
type RequestMeta struct {
    RequestID     string // Unique request identifier for tracing
    InternalToken string // Internal service token
    RemoteIP      string // Client IP address
    UserAgent     string // Client user agent string
}
```

### Functions

#### User Management

```go
// WithUser stores user information in the context
func WithUser(ctx context.Context, u *UserInfo) context.Context

// UserFrom retrieves user information from the context
func UserFrom(ctx context.Context) (*UserInfo, bool)
```

#### Request Metadata

```go
// WithMeta stores request metadata in the context
func WithMeta(ctx context.Context, m *RequestMeta) context.Context

// MetaFrom retrieves request metadata from the context
func MetaFrom(ctx context.Context) (*RequestMeta, bool)
```

#### Permission Checks

```go
// HasRole checks if the user has the specified role
func HasRole(ctx context.Context, role string) bool

// HasAnyRole checks if the user has any of the specified roles
func HasAnyRole(ctx context.Context, roles ...string) bool

// IsInternalService checks if the request is from an internal service
func IsInternalService(ctx context.Context, serviceName string) bool
```

## Usage Examples

### Basic User Management

```go
func handleRequest(ctx context.Context) {
    // Add user to context
    user := &identity.UserInfo{
        UserID:   "user-123",
        UserName: "john.doe",
        Roles:    []string{"admin", "user"},
    }
    ctx = identity.WithUser(ctx, user)
    
    // Use in business logic
    processRequest(ctx)
}

func processRequest(ctx context.Context) {
    if user, ok := identity.UserFrom(ctx); ok {
        log.Printf("Processing request for user: %s", user.UserName)
        
        // Check permissions
        if identity.HasRole(ctx, "admin") {
            // Admin-only logic
        }
    }
}
```

### Request Metadata

```go
func handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Add request metadata
    meta := &identity.RequestMeta{
        RequestID:     generateRequestID(),
        InternalToken: r.Header.Get("X-Internal-Token"),
        RemoteIP:      getClientIP(r),
        UserAgent:     r.UserAgent(),
    }
    ctx = identity.WithMeta(ctx, meta)
    
    // Process request with metadata
    processWithMetadata(ctx)
}

func processWithMetadata(ctx context.Context) {
    if meta, ok := identity.MetaFrom(ctx); ok {
        log.Printf("Request ID: %s, IP: %s", meta.RequestID, meta.RemoteIP)
        
        // Check if internal service
        if identity.IsInternalService(ctx, "user-service") {
            // Internal service logic
        }
    }
}
```

### Permission-Based Access Control

```go
func adminOnlyHandler(ctx context.Context) error {
    // Check for admin role
    if !identity.HasRole(ctx, "admin") {
        return errors.New("PERMISSION_DENIED", "admin role required")
    }
    
    // Admin logic here
    return nil
}

func multiRoleHandler(ctx context.Context) error {
    // Check for any of multiple roles
    if !identity.HasAnyRole(ctx, "admin", "moderator", "editor") {
        return errors.New("PERMISSION_DENIED", "insufficient permissions")
    }
    
    // Authorized logic here
    return nil
}
```

## Integration with Connect

The identity package integrates seamlessly with Connect interceptors:

```go
// In your Connect service
func (s *MyService) MyMethod(ctx context.Context, req *connect.Request[MyRequest]) (*connect.Response[MyResponse], error) {
    // User information is automatically injected by Connect interceptors
    if user, ok := identity.UserFrom(ctx); ok {
        log.Printf("User %s called MyMethod", user.UserName)
    }
    
    // Check permissions
    if !identity.HasRole(ctx, "user") {
        return nil, connect.NewError(connect.CodePermissionDenied, errors.New("PERMISSION_DENIED", "user role required"))
    }
    
    // Business logic
    return connect.NewResponse(&MyResponse{}), nil
}
```

## Best Practices

### 1. Context Propagation

Always propagate the context through your call chain:

```go
func serviceA(ctx context.Context) error {
    // Add user to context
    ctx = identity.WithUser(ctx, &identity.UserInfo{...})
    
    // Pass context to service B
    return serviceB(ctx)
}

func serviceB(ctx context.Context) error {
    // User information is available here
    if user, ok := identity.UserFrom(ctx); ok {
        // Use user information
    }
    return nil
}
```

### 2. Error Handling

Use structured errors for permission denials:

```go
func checkPermission(ctx context.Context) error {
    if !identity.HasRole(ctx, "admin") {
        return errors.New("PERMISSION_DENIED", "admin role required")
    }
    return nil
}
```

### 3. Performance Considerations

- The package is designed for high performance with minimal allocations
- Context operations are O(1) and thread-safe
- Avoid storing large amounts of data in the context

## Testing

```go
func TestUserContext(t *testing.T) {
    ctx := context.Background()
    
    // Test user storage and retrieval
    user := &identity.UserInfo{
        UserID:   "test-user",
        UserName: "test",
        Roles:    []string{"admin"},
    }
    
    ctx = identity.WithUser(ctx, user)
    
    retrievedUser, ok := identity.UserFrom(ctx)
    assert.True(t, ok)
    assert.Equal(t, user.UserID, retrievedUser.UserID)
    assert.True(t, identity.HasRole(ctx, "admin"))
}
```

## Thread Safety

All functions in this package are safe for concurrent use. The context-based approach ensures thread safety without additional synchronization.

## Dependencies

This package has **zero dependencies** and only uses Go's standard library.

## Version Compatibility

- **Go 1.21+** required
- **API Stability**: Stable (L1 module)
- **Breaking Changes**: None planned

## Contributing

Contributions are welcome! Please see the main project [Contributing Guide](../../CONTRIBUTING.md) for details.

## License

This package is part of the EggyByte framework and is licensed under the MIT License.
