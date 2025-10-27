# ⚠️ Errors Package

The `errors` package provides structured error handling with error codes for the EggyByte framework.

## Overview

This package extends Go's standard error handling with structured error codes, making it easier to handle errors programmatically and provide better error messages to clients.

## Features

- **Structured error codes** - Categorized error types for programmatic handling
- **Zero dependencies** - No external dependencies, pure Go
- **Integration ready** - Works seamlessly with Connect and HTTP handlers
- **Context preservation** - Maintains error context through the call chain
- **Type safety** - Strongly typed error creation and handling

## Quick Start

```go
import "go.eggybyte.com/egg/core/errors"

// Create structured errors
err := errors.New("VALIDATION_ERROR", "invalid email format")

// Check error types
if errors.Is(err, "VALIDATION_ERROR") {
    // Handle validation error
}

// Wrap existing errors
wrappedErr := errors.Wrap(err, "DATABASE_ERROR", "failed to save user")
```

## API Reference

### Functions

#### Error Creation

```go
// New creates a new structured error with code and message
func New(code, message string) error

// Newf creates a new structured error with formatted message
func Newf(code, format string, args ...interface{}) error

// Wrap wraps an existing error with a new code and message
func Wrap(err error, code, message string) error

// Wrapf wraps an existing error with a new code and formatted message
func Wrapf(err error, code, format string, args ...interface{}) error
```

#### Error Inspection

```go
// Is checks if an error has the specified code
func Is(err error, code string) bool

// Code extracts the error code from an error
func Code(err error) string

// Message extracts the error message from an error
func Message(err error) string
```

## Error Codes

The package defines common error codes for consistent error handling:

### System Errors
- `INTERNAL_ERROR` - Internal server error
- `SERVICE_UNAVAILABLE` - Service temporarily unavailable
- `TIMEOUT` - Operation timeout

### Validation Errors
- `VALIDATION_ERROR` - Input validation failed
- `INVALID_FORMAT` - Invalid data format
- `MISSING_REQUIRED` - Required field missing

### Authentication & Authorization
- `UNAUTHENTICATED` - User not authenticated
- `PERMISSION_DENIED` - Insufficient permissions
- `TOKEN_EXPIRED` - Authentication token expired

### Business Logic
- `NOT_FOUND` - Resource not found
- `ALREADY_EXISTS` - Resource already exists
- `CONFLICT` - Business rule conflict
- `QUOTA_EXCEEDED` - Resource quota exceeded

## Usage Examples

### Basic Error Handling

```go
func validateUser(user *User) error {
    if user.Email == "" {
        return errors.New("MISSING_REQUIRED", "email is required")
    }
    
    if !isValidEmail(user.Email) {
        return errors.New("INVALID_FORMAT", "invalid email format")
    }
    
    return nil
}

func createUser(user *User) error {
    if err := validateUser(user); err != nil {
        return err // Pass through validation errors
    }
    
    if err := saveUser(user); err != nil {
        return errors.Wrap(err, "DATABASE_ERROR", "failed to save user")
    }
    
    return nil
}
```

### Error Handling in Services

```go
func (s *UserService) GetUser(ctx context.Context, userID string) (*User, error) {
    if userID == "" {
        return nil, errors.New("MISSING_REQUIRED", "user ID is required")
    }
    
    user, err := s.repository.GetUser(ctx, userID)
    if err != nil {
        if errors.Is(err, "NOT_FOUND") {
            return nil, errors.New("NOT_FOUND", "user not found")
        }
        return nil, errors.Wrap(err, "DATABASE_ERROR", "failed to get user")
    }
    
    return user, nil
}
```

### Connect Integration

```go
func (s *UserService) GetUser(ctx context.Context, req *connect.Request[GetUserRequest]) (*connect.Response[GetUserResponse], error) {
    user, err := s.GetUser(ctx, req.Msg.UserId)
    if err != nil {
        // Convert structured errors to Connect errors
        switch errors.Code(err) {
        case "NOT_FOUND":
            return nil, connect.NewError(connect.CodeNotFound, err)
        case "PERMISSION_DENIED":
            return nil, connect.NewError(connect.CodePermissionDenied, err)
        case "VALIDATION_ERROR":
            return nil, connect.NewError(connect.CodeInvalidArgument, err)
        default:
            return nil, connect.NewError(connect.CodeInternal, err)
        }
    }
    
    return connect.NewResponse(&GetUserResponse{User: user}), nil
}
```

### HTTP Handler Integration

```go
func userHandler(w http.ResponseWriter, r *http.Request) {
    user, err := getUserFromRequest(r)
    if err != nil {
        writeErrorResponse(w, err)
        return
    }
    
    // Process user
}

func writeErrorResponse(w http.ResponseWriter, err error) {
    code := errors.Code(err)
    message := errors.Message(err)
    
    var statusCode int
    switch code {
    case "NOT_FOUND":
        statusCode = http.StatusNotFound
    case "PERMISSION_DENIED":
        statusCode = http.StatusForbidden
    case "VALIDATION_ERROR":
        statusCode = http.StatusBadRequest
    default:
        statusCode = http.StatusInternalServerError
    }
    
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(map[string]string{
        "error":   code,
        "message": message,
    })
}
```

## Error Wrapping Patterns

### Database Operations

```go
func (r *UserRepository) SaveUser(ctx context.Context, user *User) error {
    if err := r.db.Save(user).Error; err != nil {
        return errors.Wrap(err, "DATABASE_ERROR", "failed to save user")
    }
    return nil
}
```

### External Service Calls

```go
func (c *EmailClient) SendEmail(ctx context.Context, email *Email) error {
    resp, err := c.client.Post("/send", email)
    if err != nil {
        return errors.Wrap(err, "EXTERNAL_SERVICE_ERROR", "failed to send email")
    }
    
    if resp.StatusCode >= 400 {
        return errors.New("EXTERNAL_SERVICE_ERROR", "email service returned error")
    }
    
    return nil
}
```

### Business Logic

```go
func (s *OrderService) ProcessOrder(ctx context.Context, order *Order) error {
    // Check inventory
    if !s.inventory.HasStock(order.Items) {
        return errors.New("INSUFFICIENT_STOCK", "not enough inventory")
    }
    
    // Process payment
    if err := s.payment.ProcessPayment(order.Payment); err != nil {
        return errors.Wrap(err, "PAYMENT_ERROR", "failed to process payment")
    }
    
    // Create order
    if err := s.repository.CreateOrder(ctx, order); err != nil {
        return errors.Wrap(err, "DATABASE_ERROR", "failed to create order")
    }
    
    return nil
}
```

## Testing

```go
func TestErrorHandling(t *testing.T) {
    // Test error creation
    err := errors.New("VALIDATION_ERROR", "invalid input")
    assert.Equal(t, "VALIDATION_ERROR", errors.Code(err))
    assert.Equal(t, "invalid input", errors.Message(err))
    
    // Test error checking
    assert.True(t, errors.Is(err, "VALIDATION_ERROR"))
    assert.False(t, errors.Is(err, "NOT_FOUND"))
    
    // Test error wrapping
    originalErr := errors.New("ORIGINAL_ERROR", "original message")
    wrappedErr := errors.Wrap(originalErr, "WRAPPER_ERROR", "wrapper message")
    
    assert.True(t, errors.Is(wrappedErr, "WRAPPER_ERROR"))
    assert.Contains(t, wrappedErr.Error(), "wrapper message")
}
```

## Best Practices

### 1. Use Specific Error Codes

```go
// Good: Specific error code
return errors.New("INVALID_EMAIL_FORMAT", "email must be valid")

// Avoid: Generic error code
return errors.New("ERROR", "something went wrong")
```

### 2. Preserve Error Context

```go
func processUser(user *User) error {
    if err := validateUser(user); err != nil {
        return err // Don't wrap validation errors
    }
    
    if err := saveUser(user); err != nil {
        return errors.Wrap(err, "DATABASE_ERROR", "failed to save user")
    }
    
    return nil
}
```

### 3. Handle Errors Appropriately

```go
func handleError(err error) {
    switch errors.Code(err) {
    case "VALIDATION_ERROR":
        // Return 400 Bad Request
    case "NOT_FOUND":
        // Return 404 Not Found
    case "PERMISSION_DENIED":
        // Return 403 Forbidden
    default:
        // Return 500 Internal Server Error
    }
}
```

## Thread Safety

All functions in this package are safe for concurrent use.

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
