# Egg Logging Standard

## Overview

This document defines the unified logging standard for the egg framework. All services using egg must follow these guidelines to ensure logs are structured, parseable, and compatible with observability systems like Loki, Promtail, and OpenTelemetry.

## Format Requirements

### ✅ Single-Line Logfmt

All logs MUST be output in single-line logfmt format:

```
level=INFO msg="operation completed" service=user-service version=1.0.0 duration_ms=1.5 user_id="u-123"
```

**Requirements:**
- No multi-line output
- No nested brackets `[key value]`
- No JSON objects in values
- All key-value pairs separated by spaces
- String values with spaces MUST be quoted

### ✅ Field Ordering

Fields MUST be output in the following order:

1. `level` - Log level (DEBUG, INFO, WARN, ERROR)
2. `msg` - Log message (always quoted)
3. All other fields in **alphabetical order by key**

Example:
```
level=INFO msg="user created" email="user@example.com" service=user-service user_id="123" version=1.0.0
```

### ✅ No Timestamps

**Do NOT include timestamps** in log output. Container runtimes (Docker, Kubernetes) automatically add timestamps.

❌ Bad:
```
time=2025-10-24T03:31:54+08:00 level=INFO msg="starting"
```

✅ Good:
```
level=INFO msg="starting"
```

## Standard Fields

### Required Fields

These fields MUST be present in all logs from servicex-based services:

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `level` | string | Log level | `INFO`, `ERROR`, `WARN`, `DEBUG` |
| `msg` | string | Human-readable message | `"request completed"` |
| `service` | string | Service name | `user-service` |
| `version` | string | Service version | `1.0.0` |

### Recommended Fields

These fields SHOULD be included when available:

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `procedure` | string | RPC method or handler | `/user.v1.UserService/CreateUser` |
| `request_id` | string | Unique request identifier | `req-abc123` |
| `trace_id` | string | Distributed trace ID | `a1b2c3d4e5f6` |
| `user_id` | string | Authenticated user ID | `u-123` |
| `duration_ms` | float | Operation duration in milliseconds | `1.5` |

### Error Fields

When `level=ERROR`, these fields MUST be included:

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `error` | string | Error message | `"record not found"` |
| `code` | string | Error code | `NOT_FOUND`, `INTERNAL` |
| `op` | string | Operation that failed | `repository.GetUser` |

Example error log:
```
level=ERROR msg="database query failed" code="NOT_FOUND" error="record not found" op="repository.GetUser" service=user-service
```

## Color Guidelines

### ✅ Level-Only Coloring

**ONLY** the level value should be colored, not the key:

```
level=INFO  # "INFO" in cyan, "level=" in normal text
```

### Color Mapping

| Level | Color | ANSI Code |
|-------|-------|-----------|
| DEBUG | Magenta | `\033[35m` |
| INFO | Cyan | `\033[36m` |
| WARN | Yellow | `\033[33m` |
| ERROR | Red | `\033[31m` |

### Environment Control

Colors MUST be disabled in non-TTY environments (containers, pipes):

- Default: `Color=false` in production
- Enable with: `LOG_COLOR=true` environment variable

## Request Lifecycle Logs

### Start Log

```
level=INFO msg="request started" procedure="/user.v1.UserService/CreateUser" request_id="req-123" service=user-service
```

### Application Logic

```
level=INFO msg="creating user record" email="user@example.com" op="service.CreateUser" service=user-service user_id="123"
```

### Error Handling

```
level=ERROR msg="database query failed" code="NOT_FOUND" error="record not found" op="repository.FindUser" service=user-service
```

### Completion Log

```
level=INFO msg="request completed" duration_ms=1.71 procedure="/user.v1.UserService/CreateUser" request_id="req-123" service=user-service status="OK"
```

## Value Formatting Rules

### Strings

Always quote strings that contain spaces or special characters:

```
msg="user created"          # Quoted (contains space)
service=user-service        # Unquoted (no spaces)
email="test@example.com"    # Quoted (contains special chars)
```

### Numbers

Format numbers without quotes:

```
duration_ms=1.5    # Float
count=5            # Integer
user_id="123"      # String (quoted)
```

### Floats

Remove unnecessary trailing zeros:

```
duration_ms=1.5    # Not 1.500000
latency=2          # Not 2.0
```

### Booleans

Format as lowercase without quotes:

```
enabled=true
success=false
```

### Durations

Always convert to milliseconds:

```
duration_ms=1500   # Not "1.5s"
timeout_ms=5000    # Not "5s"
```

## Anti-Patterns

### ❌ Nested Brackets

```
msg="CreateUser request received" [email test@example.com]="[name Test User]"
```

### ❌ Multiple Lines

```
msg="user created
  email: test@example.com
  name: Test User"
```

### ❌ JSON Values

```
msg="user created" data={"email":"test@example.com","name":"Test User"}
```

### ❌ Random Field Order

```
# Inconsistent ordering between logs
level=INFO user_id="123" msg="action 1" service=app
level=INFO service=app msg="action 2" user_id="123"
```

### ❌ Timestamps in Logs

```
time=2025-10-24T03:31:54+08:00 level=INFO msg="starting"
```

## Loki/Promtail Configuration

### Recommended Label Keys

```yaml
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

### Query Examples

```promql
# All errors from user-service
{service="user-service"} |= `level=ERROR`

# Slow requests (> 1000ms)
{service="user-service"} | logfmt | duration_ms > 1000

# Specific error codes
{service="user-service"} | logfmt | code="NOT_FOUND"

# Requests by procedure
{service="user-service"} | logfmt | procedure="/user.v1.UserService/CreateUser"
```

## Implementation

### Using logx

```go
import (
    "go.eggybyte.com/egg/logx"
    "log/slog"
)

// Create logger
logger := logx.New(
    logx.WithFormat(logx.FormatLogfmt),
    logx.WithLevel(slog.LevelInfo),
    logx.WithColor(false), // Disable in production
)

// Add service context
logger = logger.With(
    "service", "user-service",
    "version", "1.0.0",
)

// Log messages
logger.Info("user created",
    "user_id", "123",
    "email", "test@example.com",
)

// Log errors
logger.Error(err, "database query failed",
    "op", "repository.GetUser",
    "code", "NOT_FOUND",
)
```

### Using servicex

servicex automatically configures logging with proper defaults:

```go
import "go.eggybyte.com/egg/servicex"

servicex.Run(ctx,
    servicex.WithService("user-service", "1.0.0"),
    servicex.WithRegister(func(app *servicex.App) error {
        // Logger already configured with service/version
        app.Logger().Info("service initialized")
        return nil
    }),
)
```

## Validation

All logs MUST match this regex pattern:

```regex
^level=(DEBUG|INFO|WARN|ERROR) msg="[^"]+"( [a-zA-Z0-9_]+=("[^"]*"|[^ ]+))*$
```

### Valid Examples

```
level=INFO msg="starting service"
level=INFO msg="user created" email="test@example.com" service=user-service user_id="123"
level=ERROR msg="query failed" code="NOT_FOUND" error="record not found" op="repo.Get"
```

### Invalid Examples

```
INFO: user created                                    # Wrong format
level=INFO msg=user created email=test@example.com   # Unquoted msg
[INFO] user created                                  # Wrong format
time=2025... level=INFO msg="starting"                # Has timestamp
```

## Environment Variables

Control log behavior via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `INFO` | Minimum log level (DEBUG, INFO, WARN, ERROR) |
| `LOG_FORMAT` | `logfmt` | Output format (logfmt, json) |
| `LOG_COLOR` | `false` | Enable ANSI colors |

## Summary

✅ **DO:**
- Use single-line logfmt format
- Sort fields alphabetically (after level and msg)
- Quote strings with spaces/special chars
- Include service and version in all logs
- Use standard field names
- Color only the level value
- Disable timestamps (container adds them)

❌ **DON'T:**
- Use multi-line logs
- Use nested structures `[key value]`
- Include timestamps
- Use inconsistent field names
- Color entire log lines
- Use JSON in field values

## References

- [logx Package](/logx/README.md)
- [servicex Package](/servicex/README.md)
- [Loki Documentation](https://grafana.com/docs/loki/latest/)
- [Logfmt Specification](https://brandur.org/logfmt)

