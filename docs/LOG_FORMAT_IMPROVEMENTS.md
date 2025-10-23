# Egg Log Format Improvements

## Summary

This document summarizes the improvements made to the egg logging system to comply with the unified logging standard: single-line logfmt format that is Loki-compatible, field-sorted, and professionally structured.

## Changes Made

### 1. logx Package Improvements

#### Core Format Changes

**File**: `logx/logx.go`

- **Timestamps**: Disabled by default (containers add timestamps automatically)
  ```go
  DisableTimestamp: true, // Container already adds timestamp
  ```

- **Message Quoting**: All messages are now consistently quoted
  ```go
  buf.WriteString(fmt.Sprintf("%q", msg))  // Always quoted
  ```

- **Value Formatting**: Strings are always quoted for logfmt consistency
  ```go
  case slog.KindString:
      return fmt.Sprintf("%q", s)  // Always quote strings
  ```

- **Float Formatting**: Clean formatting without trailing zeros
  ```go
  case slog.KindFloat64:
      f := v.Float64()
      if f == float64(int64(f)) {
          return fmt.Sprintf("%.0f", f)  // 1.0 → 1
      }
      return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", f), "0"), ".")
  ```

- **Duration Formatting**: Convert to milliseconds for consistency
  ```go
  case slog.KindDuration:
      ms := v.Duration().Milliseconds()
      return fmt.Sprintf("%d", ms)
  ```

#### Color Improvements

- **Level-Only Coloring**: Only the level value is colored, not the entire line
  ```go
  buf.WriteString("level=")
  if h.opts.Color {
      buf.WriteString(colorizeLevel(levelStr))
  } else {
      buf.WriteString(levelStr)
  }
  ```

- **Updated Color Scheme**:
  - DEBUG: Magenta (`\033[35m`)
  - INFO: Cyan (`\033[36m`)
  - WARN: Yellow (`\033[33m`)
  - ERROR: Red (`\033[31m`)

#### Test Updates

**File**: `logx/logx_test.go`

Updated all tests to expect quoted string values:
- `key=value` → `key="value"`
- `service=test` → `service="test"`
- `user_id=u-123` → `user_id="u-123"`

### 2. configx Package Improvements

#### Log Statement Simplification

**File**: `configx/configx.go`

Simplified configuration loaded log to avoid bracket notation:

**Before**:
```go
m.logger.Info("configuration loaded", log.Int("keys", len(merged)), log.Str("SERVICE_NAME", merged["SERVICE_NAME"]), log.Str("HTTP_PORT", merged["HTTP_PORT"]))
```

**After**:
```go
m.logger.Info("configuration loaded", "keys", len(merged))
```

**Output**:
```
level=INFO msg="configuration loaded" keys=45
```

#### Added Port Getter Methods

**File**: `configx/configx.go`

Added getter methods for port configuration:
```go
func (c *BaseConfig) GetHTTPPort() string { return c.HTTPPort }
func (c *BaseConfig) GetHealthPort() string { return c.HealthPort }
func (c *BaseConfig) GetMetricsPort() string { return c.MetricsPort }
```

### 3. servicex Package Improvements

#### Service Context Injection

**File**: `servicex/servicex.go`

Automatically inject service and version fields into all logs:

**Before**:
```go
cfg.logger.Info("starting service",
    "service_name", cfg.serviceName,
    "version", cfg.serviceVersion,
)
```

**After**:
```go
// Inject service and version fields into all logs
cfg.logger = cfg.logger.With(
    "service", cfg.serviceName,
    "version", cfg.serviceVersion,
)

cfg.logger.Info("starting service")
```

**Output**:
```
level=INFO msg="starting service" service="user-service" version="1.0.0"
```

#### Port Separation

Added support for separate HTTP, health check, and metrics ports:

```go
type serviceConfig struct {
    // ...
    httpPort    string
    healthPort  string
    metricsPort string
    // ...
}
```

Automatically extract ports from `BaseConfig`:

```go
if baseGetter, ok := cfg.config.(interface {
    GetHTTPPort() string
    GetHealthPort() string
    GetMetricsPort() string
}); ok {
    if port := baseGetter.GetHTTPPort(); port != "" {
        cfg.httpPort = port
    }
    // ...
}
```

### 4. Documentation

#### New Documents

1. **docs/LOGGING.md**: Comprehensive logging standard
   - Format requirements
   - Standard fields
   - Color guidelines
   - Request lifecycle logging
   - Loki/Promtail integration
   - Validation rules

2. **logx/README.md**: Updated with new format
   - Log format standard
   - Color support details
   - Standard field names
   - Error logging patterns
   - Request lifecycle examples
   - Loki integration examples

#### Test Scripts

1. **scripts/test-log-format.sh**: Automated format validation
   - Checks single-line format
   - Verifies level field presence
   - Validates message quoting
   - Confirms no timestamps
   - Checks for service/version fields
   - Detects bracket notation

2. **scripts/demo-log-colors.go**: Interactive demonstration
   - Shows all log levels with colors
   - Demonstrates field sorting
   - Shows with/without colors
   - Highlights key features

## Before and After Examples

### Starting Service

**Before**:
```
time=2025-10-24T03:31:54+08:00 level=INFO msg="starting service" service_name=user-service version=0.1.0
```

**After**:
```
level=INFO msg="starting service" service="user-service" version="0.1.0"
```

### Configuration Loaded

**Before**:
```
level=INFO msg="configuration loaded" [keys 45]="[SERVICE_NAME ]"
```

**After**:
```
level=INFO msg="configuration loaded" keys=45 service="user-service" version="0.1.0"
```

### User Created

**Before**:
```
level=INFO msg="CreateUser request received" [email test@example.com]="[name Test User]"
```

**After**:
```
level=INFO msg="user created successfully" email="test@example.com" name="Test User" service="user-service" user_id="u-123" version="0.1.0"
```

### Error Logging

**Before**:
```
level=ERROR msg="database query failed"
```

**After**:
```
level=ERROR msg="database query failed" code="NOT_FOUND" error="record not found" op="repository.GetUser" service="user-service" version="0.1.0"
```

### Request Completion

**Before**:
```
level=INFO msg="request completed" duration=1.713
```

**After**:
```
level=INFO msg="request completed" duration_ms=1.713 procedure="/user.v1.UserService/CreateUser" request_id="req-abc" service="user-service" status="OK" version="0.1.0"
```

## Validation Results

All format requirements pass:

```
✓ All logs are single-line
✓ All logs have level field
✓ All messages are properly quoted
✓ No timestamps in logs (good for containers)
✓ Service field present
✓ Version field present
✓ No bracket notation found
```

## Benefits

### 1. Observability

- **Loki/Promtail**: Natively parseable without custom regex
- **Field Consistency**: Same field names across all services
- **Queryability**: Easy filtering by service, procedure, code, etc.

### 2. Debugging

- **Stable Output**: Sorted fields make diffs meaningful
- **No Noise**: No timestamps duplicated from container runtime
- **Clear Structure**: Key-value pairs are immediately understandable

### 3. Performance

- **Efficient Parsing**: Logfmt is faster to parse than JSON
- **Compact Output**: No redundant information
- **Minimal Overhead**: Single-line output reduces I/O

### 4. Developer Experience

- **Consistent Format**: Same format everywhere
- **Color Coding**: Level-only coloring for clear visual hierarchy
- **Type Safety**: Built on standard library types

## Migration Checklist

For existing services:

- [ ] Update to latest `logx` version
- [ ] Remove timestamp configuration (now default off)
- [ ] Use standard field names (`service`, `version`, `procedure`, etc.)
- [ ] Ensure error logs include `op`, `code`, `error` fields
- [ ] Remove bracket notation `[key value]`
- [ ] Update Loki/Promtail config to use logfmt parser
- [ ] Test with `scripts/test-log-format.sh`

## Related Files

- `/logx/logx.go` - Core logging implementation
- `/logx/logx_test.go` - Test suite
- `/logx/README.md` - Package documentation
- `/configx/configx.go` - Configuration logging
- `/servicex/servicex.go` - Service logging setup
- `/docs/LOGGING.md` - Complete logging standard
- `/scripts/test-log-format.sh` - Validation script
- `/scripts/demo-log-colors.go` - Interactive demo

## References

- [Logfmt Specification](https://brandur.org/logfmt)
- [Grafana Loki Documentation](https://grafana.com/docs/loki/latest/)
- [Go slog Package](https://pkg.go.dev/log/slog)
- [egg Logging Standard](/docs/LOGGING.md)

