# egg/logx

## Overview

`logx` provides structured logging based on `log/slog` with logfmt/JSON output formats,
field sorting, colorization, and sensitive field masking. It's designed for production
use with container-friendly output and minimal performance overhead.

## Key Features

- Multiple output formats: Logfmt, JSON, and Console (human-readable)
- Automatic field sorting for consistency
- Optional colorization for development
- Sensitive field masking (passwords, tokens)
- Payload size limiting
- Context-aware logging with trace/request IDs
- Zero external dependencies beyond stdlib

## Dependencies

Layer: **L1 (Foundation Layer)**  
Depends on: `core/log`, `core/identity`, `log/slog` (stdlib)

## Installation

```bash
go get go.eggybyte.com/egg/logx@latest
```

## Basic Usage

```go
import (
    "log/slog"
    "go.eggybyte.com/egg/logx"
)

func main() {
    // Create logger with logfmt format
    logger := logx.New(
        logx.WithFormat(logx.FormatLogfmt),
        logx.WithLevel(slog.LevelInfo),
        logx.WithColor(true),
    )
    
    // Log messages
    logger.Info("user created", "user_id", "u-123", "email", "user@example.com")
    logger.Warn("slow request", "duration_ms", 1500, "path", "/api/users")
    logger.Error(err, "database connection failed", "retry_count", 3)
}
```

Output (logfmt):
```
level=INFO msg="user created" email=user@example.com user_id=u-123
level=WARN msg="slow request" duration_ms=1500 path=/api/users
level=ERROR msg="database connection failed" error="connection timeout" retry_count=3
```

### Console Format (Human-Readable)

For development environments, use the console format for easier reading:

```go
logger := logx.New(
    logx.WithFormat(logx.FormatConsole),
    logx.WithLevel(slog.LevelInfo),
    logx.WithColor(true),
)

logger.Info("user created", "user_id", "u-123", "email", "user@example.com")
logger.Warn("slow request", "duration_ms", 1500, "path", "/api/users")
```

Output (console with colors):
```
INFO    2024-01-15 10:30:00  user created
        email: user@example.com
        user_id: u-123
WARN    2024-01-15 10:30:01  slow request
        duration_ms: 1500
        path: /api/users
```

The console format provides:
- Aligned log levels with colors
- Human-readable timestamps
- Indented key-value pairs
- No quotes around strings
- Natural duration formatting (e.g., "100ms" instead of "100")

## Log Level Configuration

### Using Environment Variable (Recommended)

When using `servicex`, log level is automatically configured from the `LOG_LEVEL` environment variable:

```bash
# Debug - Shows all logs including request/response bodies
LOG_LEVEL=debug go run main.go

# Info - Standard operational logs (default)
LOG_LEVEL=info go run main.go

# Warn - Only warnings and errors
LOG_LEVEL=warn go run main.go

# Error - Only errors
LOG_LEVEL=error go run main.go
```

**In Docker Compose:**

```yaml
environment:
  LOG_LEVEL: debug  # debug, info, warn, error
```

### Programmatic Configuration

You can also set the log level programmatically:

```go
import (
    "log/slog"
    "go.eggybyte.com/egg/logx"
)

// Create logger with specific level
logger := logx.New(
    logx.WithFormat(logx.FormatConsole),
    logx.WithLevel(slog.LevelDebug),
    logx.WithColor(true),
)
```

### Integration with servicex

When using `servicex`, you have two options:

**Option 1: Let servicex create the logger (recommended)**

```go
servicex.Run(ctx,
    servicex.WithService("my-service", "1.0.0"),
    servicex.WithAppConfig(cfg),
    // servicex creates logger based on LOG_LEVEL environment variable
)
```

**Option 2: Provide custom logger**

```go
logger := logx.New(
    logx.WithFormat(logx.FormatConsole),
    logx.WithLevel(slog.LevelDebug),
    logx.WithColor(true),
)

servicex.Run(ctx,
    servicex.WithLogger(logger), // Custom logger takes precedence
    servicex.WithAppConfig(cfg),
)
```

**Priority:**
1. Custom logger via `WithLogger()` → uses its configured level
2. `LOG_LEVEL` environment variable → parsed and applied
3. Default → `info` level

## Configuration Options

| Option                | Type          | Description                                |
| --------------------- | ------------- | ------------------------------------------ |
| `WithFormat(format)`  | `Format`      | Output format: logfmt, json, or console    |
| `WithLevel(level)`    | `slog.Level`  | Minimum log level (Debug, Info, Warn, Error) |
| `WithColor(enabled)`  | `bool`        | Enable colorization (for levels and console format) |
| `WithWriter(w)`       | `io.Writer`   | Output writer (default: os.Stderr)         |
| `WithPayloadLimit(n)` | `int`         | Maximum bytes for large payloads           |
| `WithSensitiveFields()`| `[]string`   | Field names to mask (e.g., "password")     |

## API Reference

### Logger Interface

```go
type Logger interface {
    // With returns a new Logger with additional key-value pairs
    With(kv ...any) Logger
    
    // Debug logs a debug message
    Debug(msg string, kv ...any)
    
    // Info logs an informational message
    Info(msg string, kv ...any)
    
    // Warn logs a warning message
    Warn(msg string, kv ...any)
    
    // Error logs an error message
    Error(err error, msg string, kv ...any)
}
```

### Constructor

```go
// New creates a new Logger with the given options
func New(opts ...Option) log.Logger
```

### Context Helper

```go
// FromContext creates a logger with context-injected fields
func FromContext(ctx context.Context, base log.Logger) log.Logger
```

## Architecture

The logx module follows a clean architecture pattern:

```
logx/
├── logx.go              # Public API (~210 lines)
│   ├── Logger           # Logger implementation
│   ├── Options          # Configuration structs
│   ├── New()            # Constructor
│   └── FromContext()    # Context helper
└── internal/
    └── handler.go       # slog.Handler implementation (~260 lines)
        ├── Handle()     # Log record processing
        ├── formatLogfmt()   # Logfmt formatting
        ├── formatJSON()     # JSON formatting
        └── sortAndFilter()  # Field sorting
```

**Design Highlights:**
- Public interface implements `core/log.Logger`
- Complex formatting logic in internal handler
- Efficient field sorting and filtering
- Minimal allocations in hot path

## Example: Basic Logging

```go
package main

import (
    "log/slog"
    "go.eggybyte.com/egg/logx"
)

func main() {
    logger := logx.New(
        logx.WithFormat(logx.FormatLogfmt),
        logx.WithLevel(slog.LevelInfo),
        logx.WithColor(true),
    )
    
    logger.Info("server started", "port", 8080)
    logger.Debug("debug message")  // Not printed (level = Info)
    logger.Warn("high memory usage", "percent", 85)
    
    err := connectDatabase()
    if err != nil {
        logger.Error(err, "failed to connect to database", "retry", 3)
    }
}
```

## Example: Context-Aware Logging

```go
import (
    "context"
    "go.eggybyte.com/egg/core/identity"
    "go.eggybyte.com/egg/logx"
)

func handleRequest(ctx context.Context, baseLogger log.Logger) {
    // Extract request metadata from context
    logger := logx.FromContext(ctx, baseLogger)
    
    // Logger now includes request_id and user_id from context
    logger.Info("processing request")
    // Output: level=INFO msg="processing request" request_id=req-123 user_id=u-456
    
    logger.Info("request completed", "duration_ms", 150)
    // Output: level=INFO msg="request completed" duration_ms=150 request_id=req-123 user_id=u-456
}

// In your handler
func MyHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Inject request metadata
    ctx = identity.WithMeta(ctx, identity.Meta{
        RequestID: "req-123",
    })
    
    // Inject user info
    ctx = identity.WithUser(ctx, identity.User{
        UserID: "u-456",
    })
    
    handleRequest(ctx, logger)
}
```

## Example: Structured Fields

```go
func logUserActivity(logger log.Logger) {
    // Simple fields
    logger.Info("user login",
        "user_id", "u-123",
        "ip", "192.168.1.1",
        "success", true,
    )
    
    // Numeric fields
    logger.Info("api request",
        "endpoint", "/api/users",
        "status_code", 200,
        "duration_ms", 45,
        "response_bytes", 1024,
    )
    
    // Error fields
    err := errors.New("connection timeout")
    logger.Error(err, "operation failed",
        "operation", "save_user",
        "retry_count", 3,
    )
}
```

Output (logfmt):
```
level=INFO msg="user login" ip=192.168.1.1 success=true user_id=u-123
level=INFO msg="api request" duration_ms=45 endpoint=/api/users response_bytes=1024 status_code=200
level=ERROR msg="operation failed" error="connection timeout" operation=save_user retry_count=3
```

## Example: Sensitive Field Masking

```go
logger := logx.New(
    logx.WithFormat(logx.FormatLogfmt),
    logx.WithSensitiveFields("password", "token", "secret", "api_key"),
)

// These fields will be masked
logger.Info("user auth",
    "username", "john",
    "password", "secret123",        // Masked: password=***
    "api_key", "sk_live_123456",    // Masked: api_key=***
)

// Output: level=INFO msg="user auth" api_key=*** password=*** username=john
```

## Example: Logger Chaining

```go
// Base logger
baseLogger := logx.New(
    logx.WithFormat(logx.FormatLogfmt),
    logx.WithLevel(slog.LevelInfo),
)

// Add service context
serviceLogger := baseLogger.With(
    "service", "user-service",
    "version", "1.0.0",
)

// Add request context
requestLogger := serviceLogger.With(
    "request_id", "req-123",
    "method", "POST",
    "path", "/api/users",
)

// All logs include service and request context
requestLogger.Info("creating user", "user_id", "u-456")
// Output: level=INFO msg="creating user" method=POST path=/api/users request_id=req-123 service=user-service user_id=u-456 version=1.0.0
```

## Example: JSON Format

```go
logger := logx.New(
    logx.WithFormat(logx.FormatJSON),
    logx.WithLevel(slog.LevelInfo),
)

logger.Info("user created",
    "user_id", "u-123",
    "email", "user@example.com",
    "created_at", time.Now(),
)
```

Output (JSON):
```json
{"level":"INFO","msg":"user created","created_at":"2024-01-15T10:30:00Z","email":"user@example.com","user_id":"u-123"}
```

## Field Sorting

logx automatically sorts fields alphabetically for consistent output:

```go
logger.Info("event",
    "zebra", 1,
    "apple", 2,
    "banana", 3,
)
// Output: level=INFO msg=event apple=2 banana=3 zebra=1
```

**Special Fields Order:**
1. `level` (always first)
2. `msg` (always second)
3. Other fields (sorted alphabetically)
4. `error` (if present, sorted with others)

## Payload Limiting

Limit large field values to prevent log bloat:

```go
logger := logx.New(
    logx.WithFormat(logx.FormatLogfmt),
    logx.WithPayloadLimit(100),  // Limit to 100 bytes
)

largeData := strings.Repeat("x", 200)
logger.Info("data received", "payload", largeData)
// Output: level=INFO msg="data received" payload=xxx...(truncated)
```

## Color Support

Enable colorization for better readability in development:

```go
logger := logx.New(
    logx.WithFormat(logx.FormatLogfmt),
    logx.WithColor(true),
)

logger.Debug("debug message")  // Gray
logger.Info("info message")    // Blue
logger.Warn("warn message")    // Yellow
logger.Error(err, "error message")  // Red
```

**Note**: Colors are applied only to the level field. Disable in production.

## Integration with servicex

logx is automatically used by servicex:

```go
import "go.eggybyte.com/egg/servicex"

func main() {
    err := servicex.Run(ctx,
        servicex.WithConfig(cfg),
        servicex.WithDebugLogs(true),  // Sets level to Debug
        servicex.WithRegister(register),
    )
}

func register(app *servicex.App) error {
    logger := app.Logger()  // Get configured logger
    logger.Info("service registered")
    return nil
}
```

## Performance Considerations

- **Field Sorting**: Minimal overhead (~few microseconds per log)
- **Allocations**: Optimized to reduce allocations in hot path
- **Formatting**: Logfmt is faster than JSON
- **Buffering**: Output is buffered by default

Benchmark results (approximate):
```
BenchmarkLogfmt-8    500000    2500 ns/op    256 B/op    8 allocs/op
BenchmarkJSON-8      300000    4000 ns/op    512 B/op   12 allocs/op
```

## Best Practices

1. **Use logfmt in production** - More readable, faster, easier to parse
2. **Enable colors in development only** - Better terminal readability
3. **Mask sensitive fields** - Never log passwords, tokens, secrets
4. **Use structured fields** - Avoid string concatenation in messages
5. **Keep field names consistent** - Use snake_case (e.g., `user_id`, `request_id`)
6. **Log at appropriate levels** - Debug for development, Info for normal ops
7. **Include context** - Use `FromContext()` or `With()` for request/user context

## Stability

**Status**: Stable  
**Layer**: L1 (Foundation)  
**API Guarantees**: Backward-compatible changes only

The logx module is production-ready and follows semantic versioning.

## License

This package is part of the egg framework and is licensed under the MIT License.
See the root LICENSE file for details.
