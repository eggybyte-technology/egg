# egg/logx

Production-grade structured logging for the egg microservice framework.

## Overview

`logx` provides a lightweight, structured logging implementation that integrates seamlessly with egg services. It outputs clean, single-line logfmt format that is fully compatible with Loki, Promtail, and other observability systems.

## Key Features

- **Pure logfmt format**: Single-line, parseable, Loki-compatible output
- **Stable field ordering**: Fields sorted alphabetically for consistent diffs
- **Context-aware**: Automatic injection of `request_id`, `user_id`, `trace_id` from context
- **Security**: Sensitive field redaction (passwords, tokens, etc.)
- **No timestamps**: Optimized for container environments (Docker/K8s add timestamps)
- **Smart coloring**: Only level field is colored, controlled by environment
- **Type-safe**: Built on Go's standard `log/slog`
- **Zero dependencies**: Only uses standard library

## Log Format Standard

All logs follow this format:

```
level=INFO msg="operation completed" field1="value1" field2=123 field3="value3"
```

### Format Rules

✅ **DO:**
- Single-line output only
- `level=` and `msg=` first, then fields alphabetically sorted
- Quote strings that contain spaces or special characters
- Use standard field names (`service`, `version`, `procedure`, `duration_ms`)
- Disable timestamps (containers add them)

❌ **DON'T:**
- Use multi-line logs
- Use nested brackets `[key value]`
- Include timestamps in output
- Use inconsistent field names

See [docs/LOGGING.md](/docs/LOGGING.md) for complete standards.

## Installation

```bash
go get github.com/eggybyte-technology/egg/logx@latest
```

## Basic Usage

```go
import (
    "log/slog"
    "github.com/eggybyte-technology/egg/logx"
)

func main() {
    // Create logger with default settings
    logger := logx.New(
        logx.WithFormat(logx.FormatLogfmt),
        logx.WithLevel(slog.LevelInfo),
        logx.WithColor(false), // Disable colors in production
    )

    // Add service context
    logger = logger.With(
        "service", "user-service",
        "version", "1.0.0",
    )

    // Log messages
    logger.Info("service started")
    // Output: level=INFO msg="service started" service="user-service" version="1.0.0"

    logger.Info("user created",
        "user_id", "u-123",
        "email", "test@example.com",
    )
    // Output: level=INFO msg="user created" email="test@example.com" service="user-service" user_id="u-123" version="1.0.0"
}
```

## Log Levels

```go
logger.Debug("debug message", "op", "initialization")
// Output: level=DEBUG msg="debug message" op="initialization"

logger.Info("informational message", "status", "ok")
// Output: level=INFO msg="informational message" status="ok"

logger.Warn("warning message", "threshold", 1000)
// Output: level=WARN msg="warning message" threshold=1000

logger.Error(err, "operation failed", "op", "database.Query", "code", "INTERNAL")
// Output: level=ERROR msg="operation failed" code="INTERNAL" error="sql: no rows" op="database.Query"
```

## Context-Aware Logging

Automatically inject request metadata from context:

```go
import (
    "context"
    "github.com/eggybyte-technology/egg/core/identity"
    "github.com/eggybyte-technology/egg/logx"
)

func HandleRequest(ctx context.Context) {
    // Context contains user and request metadata
    ctx = identity.WithUser(ctx, &identity.UserInfo{UserID: "u-123"})
    ctx = identity.WithMeta(ctx, &identity.RequestMeta{RequestID: "req-abc"})

    // Create context-aware logger
    logger := logx.FromContext(ctx, baseLogger)

    logger.Info("processing request")
    // Output: level=INFO msg="processing request" request_id="req-abc" user_id="u-123"
}
```

## Color Support

Enable colors for development (automatically disabled in non-TTY):

```go
// With colors (for local development)
logger := logx.New(
    logx.WithColor(true),
)

logger.Debug("debug")   // Magenta
logger.Info("info")     // Cyan
logger.Warn("warning")  // Yellow
logger.Error(nil, "error") // Red
```

**Note**: Only the level value is colored, not the entire line.

## Sensitive Field Redaction

Automatically redact sensitive information:

```go
logger := logx.New(
    logx.WithSensitiveFields("password", "token", "secret", "api_key"),
)

logger.Info("login attempt",
    "username", "john",
    "password", "secret123",
)
// Output: level=INFO msg="login attempt" password="***REDACTED***" username="john"
```

## Payload Limiting

Truncate large payloads to prevent log bloat:

```go
logger := logx.New(
    logx.WithPayloadLimit(100), // Limit to 100 bytes
)

longString := strings.Repeat("a", 1000)
logger.Info("processing", "data", longString)
// Output: level=INFO msg="processing" data="aaaa...(truncated, 1000 bytes)"
```

## Configuration Options

### WithFormat

```go
logx.WithFormat(logx.FormatLogfmt) // Default: key=value format
logx.WithFormat(logx.FormatJSON)   // JSON format
```

### WithLevel

```go
logx.WithLevel(slog.LevelDebug) // Show DEBUG and above
logx.WithLevel(slog.LevelInfo)  // Default: Show INFO and above
logx.WithLevel(slog.LevelWarn)  // Show WARN and above
logx.WithLevel(slog.LevelError) // Show ERROR only
```

### WithColor

```go
logx.WithColor(true)  // Enable ANSI colors
logx.WithColor(false) // Disable colors (default, recommended for production)
```

### WithWriter

```go
logx.WithWriter(os.Stdout)  // Write to stdout
logx.WithWriter(os.Stderr)  // Write to stderr (default)
logx.WithWriter(customWriter) // Custom io.Writer
```

### WithSensitiveFields

```go
logx.WithSensitiveFields("password", "token", "secret", "api_key")
```

### WithPayloadLimit

```go
logx.WithPayloadLimit(1024) // Limit field values to 1024 bytes
```

## Environment Variables

Control logging behavior via environment:

```bash
# Set log level
export LOG_LEVEL=DEBUG  # DEBUG, INFO, WARN, ERROR

# Enable colors
export LOG_COLOR=true   # true, false

# Set format
export LOG_FORMAT=logfmt # logfmt, json
```

## Integration with servicex

When using `servicex`, logging is automatically configured:

```go
import "github.com/eggybyte-technology/egg/servicex"

servicex.Run(ctx,
    servicex.WithService("user-service", "1.0.0"),
    servicex.WithRegister(func(app *servicex.App) error {
        // Logger is pre-configured with service and version
        app.Logger().Info("service initialized")
        // Output: level=INFO msg="service initialized" service="user-service" version="1.0.0"
        return nil
    }),
)
```

## Standard Fields

Use these standard field names for consistency:

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `service` | string | Service name | `user-service` |
| `version` | string | Service version | `1.0.0` |
| `procedure` | string | RPC method or handler | `/user.v1.UserService/CreateUser` |
| `op` | string | Internal operation | `repository.GetUser` |
| `request_id` | string | Unique request identifier | `req-abc123` |
| `trace_id` | string | Distributed trace ID | `a1b2c3d4` |
| `user_id` | string | Authenticated user ID | `u-123` |
| `duration_ms` | float | Operation duration | `1.5` |
| `error` | string | Error message | `record not found` |
| `code` | string | Error code | `NOT_FOUND` |

## Error Logging

Always include these fields with errors:

```go
logger.Error(err, "database query failed",
    "op", "repository.GetUser",      // Operation that failed
    "code", "NOT_FOUND",               // Error code
    "user_id", "u-123",                // Context
)
// Output: level=ERROR msg="database query failed" code="NOT_FOUND" error="sql: no rows" op="repository.GetUser" user_id="u-123"
```

## Request Lifecycle Logging

Log request lifecycle consistently:

```go
// Request start
logger.Info("request started",
    "procedure", "/user.v1.UserService/CreateUser",
    "request_id", "req-123",
)

// Application logic
logger.Info("creating user record",
    "email", "user@example.com",
    "op", "service.CreateUser",
)

// Request completion
logger.Info("request completed",
    "procedure", "/user.v1.UserService/CreateUser",
    "request_id", "req-123",
    "duration_ms", 1.5,
    "status", "OK",
)
```

## Loki/Promtail Integration

Logs are automatically parseable by Loki:

```yaml
# promtail-config.yaml
scrape_configs:
  - job_name: egg-services
    static_configs:
      - targets: [localhost:3100]
    pipeline_stages:
      - logfmt:
          mapping:
            level:
            service:
            procedure:
            code:
      - labels:
          level:
          service:
          procedure:
          code:
```

Query examples:

```promql
# All errors from user-service
{service="user-service"} |= `level=ERROR`

# Slow requests
{service="user-service"} | logfmt | duration_ms > 1000

# Specific error codes
{service="user-service"} | logfmt | code="NOT_FOUND"
```

## Performance

Benchmarks on MacBook Pro (M1):

```
BenchmarkLogger-10                  1000000      1043 ns/op      512 B/op       8 allocs/op
BenchmarkLoggerWithSorting-10        500000      2156 ns/op      768 B/op      12 allocs/op
```

## Testing

```go
import (
    "bytes"
    "testing"
    "github.com/eggybyte-technology/egg/logx"
)

func TestLogging(t *testing.T) {
    var buf bytes.Buffer
    logger := logx.New(
        logx.WithWriter(&buf),
        logx.WithColor(false),
    )

    logger.Info("test message", "key", "value")

    output := buf.String()
    if !strings.Contains(output, `msg="test message"`) {
        t.Errorf("expected msg in output: %s", output)
    }
}
```

## Migration Guide

### From fmt.Printf

```go
// Before
fmt.Printf("User created: %s (%s)\n", userID, email)

// After
logger.Info("user created", "user_id", userID, "email", email)
```

### From log.Printf

```go
// Before
log.Printf("[INFO] Request completed in %dms", duration)

// After
logger.Info("request completed", "duration_ms", duration)
```

### From zap/zerolog

```go
// Before (zap)
logger.Info("user created",
    zap.String("user_id", userID),
    zap.String("email", email),
)

// After (logx)
logger.Info("user created",
    "user_id", userID,
    "email", email,
)
```

## Dependencies

- Layer: L1 (depends only on `core/log` interface)
- External: None (only standard library)
- Stability: Stable since v0.1.0

## Related Packages

- [core/log](/core/log/README.md) - Core logging interface
- [servicex](/servicex/README.md) - Service initialization with logging
- [connectx](/connectx/README.md) - RPC interceptors with logging

## License

This package is part of the EggyByte framework and is licensed under the MIT License. See the root LICENSE file for details.
