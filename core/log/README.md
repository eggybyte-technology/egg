# 📝 Log Package

The `log` package provides a structured logging interface for the EggyByte framework.

## Overview

This package defines a minimal, structured logging interface that can be implemented by various logging backends. It's designed to be zero-dependency and highly performant.

## Features

- **Zero dependencies** - No external dependencies, pure Go
- **Structured logging** - Key-value pair support for structured data
- **Context support** - Context-aware logging with request tracing
- **Performance optimized** - Minimal allocations and overhead
- **Pluggable backends** - Works with any logging implementation

## Quick Start

```go
import "github.com/eggybyte-technology/egg/core/log"

// Create a logger instance
logger := &SimpleLogger{}

// Basic logging
logger.Info("User logged in", log.Str("user_id", "123"))
logger.Error(err, "Failed to process request", log.Str("request_id", "req-456"))

// Contextual logging
ctxLogger := logger.With(log.Str("service", "user-service"))
ctxLogger.Debug("Processing request", log.Str("method", "GET"))
```

## API Reference

### Logger Interface

```go
type Logger interface {
    // With returns a new logger with additional context
    With(kv ...any) Logger
    
    // Debug logs a debug message with key-value pairs
    Debug(msg string, kv ...any)
    
    // Info logs an info message with key-value pairs
    Info(msg string, kv ...any)
    
    // Warn logs a warning message with key-value pairs
    Warn(msg string, kv ...any)
    
    // Error logs an error message with key-value pairs
    Error(err error, msg string, kv ...any)
}
```

### Key-Value Helpers

```go
// String key-value pair
func Str(key, value string) any

// Int key-value pair
func Int(key string, value int) any

// Bool key-value pair
func Bool(key string, value bool) any

// Duration key-value pair
func Dur(key string, value time.Duration) any

// Time key-value pair
func Time(key string, value time.Time) any

// Any key-value pair
func Any(key string, value any) any
```

## Usage Examples

### Basic Logging

```go
func main() {
    logger := &SimpleLogger{}
    
    // Different log levels
    logger.Debug("Debug message", log.Str("component", "main"))
    logger.Info("Application started", log.Str("version", "1.0.0"))
    logger.Warn("Deprecated feature used", log.Str("feature", "old-api"))
    logger.Error(errors.New("connection failed"), "Database connection failed")
}
```

### Structured Logging

```go
func handleRequest(logger log.Logger, req *http.Request) {
    logger.Info("Request received",
        log.Str("method", req.Method),
        log.Str("path", req.URL.Path),
        log.Str("user_agent", req.UserAgent()),
        log.Str("remote_addr", req.RemoteAddr),
    )
    
    // Process request
    if err := processRequest(req); err != nil {
        logger.Error(err, "Request processing failed",
            log.Str("method", req.Method),
            log.Str("path", req.URL.Path),
        )
    }
}
```

### Contextual Logging

```go
func processUser(logger log.Logger, userID string) {
    // Create contextual logger
    userLogger := logger.With(
        log.Str("user_id", userID),
        log.Str("operation", "process_user"),
    )
    
    userLogger.Info("Starting user processing")
    
    // Process user
    if err := validateUser(userID); err != nil {
        userLogger.Error(err, "User validation failed")
        return
    }
    
    userLogger.Info("User processing completed",
        log.Dur("duration", time.Since(start)),
    )
}
```

### Service Integration

```go
type UserService struct {
    logger log.Logger
    repo   UserRepository
}

func (s *UserService) GetUser(ctx context.Context, userID string) (*User, error) {
    s.logger.Info("Getting user", log.Str("user_id", userID))
    
    user, err := s.repo.GetUser(ctx, userID)
    if err != nil {
        s.logger.Error(err, "Failed to get user", log.Str("user_id", userID))
        return nil, err
    }
    
    s.logger.Info("User retrieved successfully",
        log.Str("user_id", userID),
        log.Str("user_name", user.Name),
    )
    
    return user, nil
}
```

### Connect Service Integration

```go
func (s *UserService) GetUser(ctx context.Context, req *connect.Request[GetUserRequest]) (*connect.Response[GetUserResponse], error) {
    s.logger.Info("GetUser called",
        log.Str("user_id", req.Msg.UserId),
        log.Str("request_id", getRequestID(ctx)),
    )
    
    user, err := s.GetUser(ctx, req.Msg.UserId)
    if err != nil {
        s.logger.Error(err, "GetUser failed",
            log.Str("user_id", req.Msg.UserId),
        )
        return nil, err
    }
    
    s.logger.Info("GetUser completed",
        log.Str("user_id", req.Msg.UserId),
    )
    
    return connect.NewResponse(&GetUserResponse{User: user}), nil
}
```

## Implementation Examples

### Simple Logger Implementation

```go
type SimpleLogger struct {
    fields map[string]any
}

func (l *SimpleLogger) With(kv ...any) log.Logger {
    newFields := make(map[string]any)
    for k, v := range l.fields {
        newFields[k] = v
    }
    
    // Parse key-value pairs
    for i := 0; i < len(kv); i += 2 {
        if i+1 < len(kv) {
            if key, ok := kv[i].(string); ok {
                newFields[key] = kv[i+1]
            }
        }
    }
    
    return &SimpleLogger{fields: newFields}
}

func (l *SimpleLogger) Debug(msg string, kv ...any) {
    l.log("DEBUG", msg, kv...)
}

func (l *SimpleLogger) Info(msg string, kv ...any) {
    l.log("INFO", msg, kv...)
}

func (l *SimpleLogger) Warn(msg string, kv ...any) {
    l.log("WARN", msg, kv...)
}

func (l *SimpleLogger) Error(err error, msg string, kv ...any) {
    if err != nil {
        kv = append(kv, log.Str("error", err.Error()))
    }
    l.log("ERROR", msg, kv...)
}

func (l *SimpleLogger) log(level, msg string, kv ...any) {
    // Build log entry
    entry := map[string]any{
        "level":   level,
        "message": msg,
        "time":    time.Now().Format(time.RFC3339),
    }
    
    // Add fields
    for k, v := range l.fields {
        entry[k] = v
    }
    
    // Add key-value pairs
    for i := 0; i < len(kv); i += 2 {
        if i+1 < len(kv) {
            if key, ok := kv[i].(string); ok {
                entry[key] = kv[i+1]
            }
        }
    }
    
    // Output log entry (implement your preferred output method)
    fmt.Printf("%+v\n", entry)
}
```

### JSON Logger Implementation

```go
type JSONLogger struct {
    fields map[string]any
    writer io.Writer
}

func (l *JSONLogger) With(kv ...any) log.Logger {
    // Similar to SimpleLogger but output JSON
    // Implementation details...
}

func (l *JSONLogger) log(level, msg string, kv ...any) {
    entry := map[string]any{
        "level":   level,
        "message": msg,
        "time":    time.Now().Format(time.RFC3339),
    }
    
    // Add fields and key-value pairs
    // ...
    
    // Output as JSON
    json.NewEncoder(l.writer).Encode(entry)
}
```

## Testing

```go
func TestLogger(t *testing.T) {
    logger := &TestLogger{}
    
    // Test basic logging
    logger.Info("test message", log.Str("key", "value"))
    assert.Contains(t, logger.entries, "test message")
    
    // Test contextual logging
    ctxLogger := logger.With(log.Str("service", "test"))
    ctxLogger.Debug("debug message")
    
    // Test error logging
    err := errors.New("test error")
    logger.Error(err, "error message")
    assert.Contains(t, logger.entries, "error message")
}

type TestLogger struct {
    entries []string
}

func (l *TestLogger) With(kv ...any) log.Logger {
    return l
}

func (l *TestLogger) Debug(msg string, kv ...any) {
    l.entries = append(l.entries, msg)
}

func (l *TestLogger) Info(msg string, kv ...any) {
    l.entries = append(l.entries, msg)
}

func (l *TestLogger) Warn(msg string, kv ...any) {
    l.entries = append(l.entries, msg)
}

func (l *TestLogger) Error(err error, msg string, kv ...any) {
    l.entries = append(l.entries, msg)
}
```

## Best Practices

### 1. Use Structured Logging

```go
// Good: Structured logging
logger.Info("User created",
    log.Str("user_id", user.ID),
    log.Str("email", user.Email),
    log.Time("created_at", user.CreatedAt),
)

// Avoid: String concatenation
logger.Info(fmt.Sprintf("User created: %s (%s)", user.ID, user.Email))
```

### 2. Include Context

```go
func processOrder(logger log.Logger, order *Order) {
    orderLogger := logger.With(
        log.Str("order_id", order.ID),
        log.Str("user_id", order.UserID),
    )
    
    orderLogger.Info("Processing order")
    // ... rest of the function
}
```

### 3. Log Errors with Context

```go
func saveUser(logger log.Logger, user *User) error {
    if err := db.Save(user).Error; err != nil {
        logger.Error(err, "Failed to save user",
            log.Str("user_id", user.ID),
            log.Str("email", user.Email),
        )
        return err
    }
    return nil
}
```

### 4. Use Appropriate Log Levels

```go
// Debug: Detailed information for debugging
logger.Debug("Processing request", log.Str("request_id", reqID))

// Info: General information about program execution
logger.Info("User logged in", log.Str("user_id", userID))

// Warn: Something unexpected happened but the program can continue
logger.Warn("Deprecated API used", log.Str("endpoint", "/old-api"))

// Error: An error occurred but the program can continue
logger.Error(err, "Database connection failed")
```

## Thread Safety

The Logger interface is designed to be thread-safe. Implementations should ensure thread safety if they maintain internal state.

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
