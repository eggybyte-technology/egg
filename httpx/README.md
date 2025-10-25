# egg/httpx

## Overview

`httpx` provides HTTP utilities for request binding, validation, security headers,
and CORS configuration. It simplifies building secure HTTP APIs with minimal boilerplate.

## Key Features

- JSON request binding with validation
- Standard error responses
- Security headers (HSTS, CSP, etc.)
- CORS middleware with flexible configuration
- Input validation using struct tags
- Clean error responses

## Dependencies

Layer: **L2 (Capability Layer)**  
Depends on: `github.com/go-playground/validator/v10`

## Installation

```bash
go get github.com/eggybyte-technology/egg/httpx@latest
```

## Basic Usage

```go
import (
    "net/http"
    "github.com/eggybyte-technology/egg/httpx"
)

type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=2,max=50"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"required,min=18,max=120"`
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
    // Bind and validate request
    var req CreateUserRequest
    if err := httpx.BindAndValidate(r, &req); err != nil {
        httpx.WriteError(w, err, http.StatusBadRequest)
        return
    }
    
    // Business logic...
    user := createUser(req)
    
    // Write JSON response
    httpx.WriteJSON(w, http.StatusCreated, map[string]any{
        "user": user,
    })
}
```

## API Reference

### Request Binding

```go
// BindAndValidate binds JSON request body to target struct and validates it
func BindAndValidate(r *http.Request, target any) error
```

### Response Writing

```go
// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, status int, data any) error

// WriteError writes a standard error response
func WriteError(w http.ResponseWriter, err error, status int) error
```

### Error Response

```go
type ErrorResponse struct {
    Error   string                 `json:"error"`
    Message string                 `json:"message,omitempty"`
    Details map[string]interface{} `json:"details,omitempty"`
}
```

### Handlers

```go
// NotFoundHandler returns a standard 404 JSON response
func NotFoundHandler() http.HandlerFunc

// MethodNotAllowedHandler returns a standard 405 JSON response
func MethodNotAllowedHandler() http.HandlerFunc
```

### Security Middleware

```go
// SecureMiddleware adds security headers to responses
func SecureMiddleware(headers SecurityHeaders) func(http.Handler) http.Handler

type SecurityHeaders struct {
    ContentTypeOptions    bool   // X-Content-Type-Options: nosniff
    FrameOptions          bool   // X-Frame-Options: DENY
    ReferrerPolicy        bool   // Referrer-Policy: no-referrer
    StrictTransportSec    bool   // Strict-Transport-Security (HSTS)
    HSTSMaxAge            int    // Max age for HSTS in seconds
    ContentSecurityPolicy string // Optional CSP header
}

// DefaultSecurityHeaders returns security headers with sensible defaults
func DefaultSecurityHeaders() SecurityHeaders
```

### CORS Middleware

```go
// CORSMiddleware adds CORS headers to responses
func CORSMiddleware(opts CORSOptions) func(http.Handler) http.Handler

type CORSOptions struct {
    AllowedOrigins   []string // Allowed origins
    AllowedMethods   []string // Allowed methods
    AllowedHeaders   []string // Allowed headers
    ExposedHeaders   []string // Exposed headers
    AllowCredentials bool     // Allow credentials
    MaxAge           int      // Preflight cache duration in seconds
}

// DefaultCORSOptions returns CORS options with sensible defaults
func DefaultCORSOptions() CORSOptions
```

## Architecture

The httpx module provides HTTP utilities:

```
httpx/
├── httpx.go             # Public API (~190 lines)
│   ├── BindAndValidate()
│   ├── WriteJSON()
│   ├── WriteError()
│   ├── NotFoundHandler()
│   ├── MethodNotAllowedHandler()
│   ├── SecureMiddleware()
│   └── CORSMiddleware()
└── internal/
    └── middleware.go    # Middleware implementations
        ├── ApplySecurityHeaders()
        └── ApplyCORSHeaders()
```

## Example: Complete API

```go
package main

import (
    "net/http"
    "github.com/eggybyte-technology/egg/httpx"
)

type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=2,max=50"`
    Email string `json:"email" validate:"required,email"`
}

type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    mux := http.NewServeMux()
    
    // Add security headers
    secureHandler := httpx.SecureMiddleware(httpx.DefaultSecurityHeaders())
    
    // Add CORS support
    corsHandler := httpx.CORSMiddleware(httpx.CORSOptions{
        AllowedOrigins: []string{"https://example.com"},
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
        AllowedHeaders: []string{"Content-Type", "Authorization"},
    })
    
    // Register handlers
    mux.HandleFunc("POST /api/users", createUserHandler)
    mux.HandleFunc("GET /api/users/{id}", getUserHandler)
    mux.HandleFunc("/", httpx.NotFoundHandler())
    
    // Apply middleware
    handler := secureHandler(corsHandler(mux))
    
    http.ListenAndServe(":8080", handler)
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := httpx.BindAndValidate(r, &req); err != nil {
        httpx.WriteError(w, err, http.StatusBadRequest)
        return
    }
    
    user := &User{
        ID:    generateID(),
        Name:  req.Name,
        Email: req.Email,
    }
    
    httpx.WriteJSON(w, http.StatusCreated, user)
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    
    user, err := findUser(id)
    if err != nil {
        httpx.WriteError(w, err, http.StatusNotFound)
        return
    }
    
    httpx.WriteJSON(w, http.StatusOK, user)
}
```

## Example: Input Validation

```go
type UpdateProfileRequest struct {
    Name     string `json:"name" validate:"required,min=2,max=50"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"min=0,max=150"`
    Website  string `json:"website" validate:"omitempty,url"`
    Bio      string `json:"bio" validate:"max=500"`
    Tags     []string `json:"tags" validate:"max=10,dive,min=1,max=20"`
}

func updateProfileHandler(w http.ResponseWriter, r *http.Request) {
    var req UpdateProfileRequest
    if err := httpx.BindAndValidate(r, &req); err != nil {
        // Returns detailed validation errors
        httpx.WriteError(w, err, http.StatusBadRequest)
        return
    }
    
    // All fields are validated
    updateProfile(req)
    httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
```

Validation error response:
```json
{
  "error": "Bad Request",
  "message": "validation failed: Key: 'UpdateProfileRequest.Email' Error:Field validation for 'Email' failed on the 'email' tag"
}
```

## Example: Security Headers

```go
// Production security headers
headers := httpx.SecurityHeaders{
    ContentTypeOptions: true,  // X-Content-Type-Options: nosniff
    FrameOptions:       true,  // X-Frame-Options: DENY
    ReferrerPolicy:     true,  // Referrer-Policy: no-referrer
    StrictTransportSec: true,  // Strict-Transport-Security: max-age=31536000
    HSTSMaxAge:         31536000,  // 1 year
    ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline'",
}

middleware := httpx.SecureMiddleware(headers)
handler := middleware(mux)
```

Response headers:
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: no-referrer
Strict-Transport-Security: max-age=31536000
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline'
```

## Example: CORS Configuration

```go
// Development CORS (allow all)
devCORS := httpx.CORSOptions{
    AllowedOrigins:   []string{"*"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowedHeaders:   []string{"*"},
    AllowCredentials: false,
    MaxAge:           3600,
}

// Production CORS (specific origins)
prodCORS := httpx.CORSOptions{
    AllowedOrigins:   []string{"https://app.example.com", "https://www.example.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Request-Id"},
    ExposedHeaders:   []string{"X-Request-Id", "X-RateLimit-Remaining"},
    AllowCredentials: true,
    MaxAge:           7200,
}

middleware := httpx.CORSMiddleware(prodCORS)
handler := middleware(mux)
```

## Example: Error Handling

```go
func handler(w http.ResponseWriter, r *http.Request) {
    user, err := getUser(id)
    
    switch {
    case err == ErrNotFound:
        httpx.WriteError(w, err, http.StatusNotFound)
    case err == ErrUnauthorized:
        httpx.WriteError(w, err, http.StatusUnauthorized)
    case err != nil:
        httpx.WriteError(w, err, http.StatusInternalServerError)
    default:
        httpx.WriteJSON(w, http.StatusOK, user)
    }
}
```

Error responses:
```json
{
  "error": "Not Found",
  "message": "user not found"
}
```

## Validation Tags

Common validation tags supported by `github.com/go-playground/validator/v10`:

| Tag          | Description                           | Example                |
| ------------ | ------------------------------------- | ---------------------- |
| `required`   | Field must be present                 | `validate:"required"`  |
| `min`        | Minimum value/length                  | `validate:"min=2"`     |
| `max`        | Maximum value/length                  | `validate:"max=50"`    |
| `email`      | Valid email format                    | `validate:"email"`     |
| `url`        | Valid URL format                      | `validate:"url"`       |
| `uuid`       | Valid UUID format                     | `validate:"uuid"`      |
| `alpha`      | Alphabetic characters only            | `validate:"alpha"`     |
| `alphanum`   | Alphanumeric characters only          | `validate:"alphanum"`  |
| `numeric`    | Numeric characters only               | `validate:"numeric"`   |
| `omitempty`  | Skip validation if field is empty     | `validate:"omitempty,email"` |
| `dive`       | Validate array/slice elements         | `validate:"dive,min=1"` |

## Best Practices

1. **Always validate input** - Use `BindAndValidate()` for all user input
2. **Use appropriate status codes** - 400 for validation, 404 for not found, etc.
3. **Enable security headers in production** - Protect against common attacks
4. **Configure CORS carefully** - Don't use `*` in production
5. **Return consistent error format** - Use `WriteError()` for all errors
6. **Validate nested structures** - Use `dive` tag for arrays/slices
7. **Handle OPTIONS requests** - CORS middleware handles this automatically

## Testing

```go
func TestCreateUser(t *testing.T) {
    // Create test request
    body := strings.NewReader(`{"name":"John","email":"john@example.com"}`)
    req := httptest.NewRequest("POST", "/api/users", body)
    req.Header.Set("Content-Type", "application/json")
    
    // Create response recorder
    w := httptest.NewRecorder()
    
    // Call handler
    createUserHandler(w, req)
    
    // Assert response
    assert.Equal(t, http.StatusCreated, w.Code)
    
    var user User
    json.NewDecoder(w.Body).Decode(&user)
    assert.Equal(t, "John", user.Name)
}
```

## Stability

**Status**: Stable  
**Layer**: L2 (Capability)  
**API Guarantees**: Backward-compatible changes only

The httpx module is production-ready and follows semantic versioning.

## License

This package is part of the egg framework and is licensed under the MIT License.
See the root LICENSE file for details.
