# 🛠️ Utils Package

The `utils` package provides common utility functions for the EggyByte framework.

## Overview

This package contains reusable utility functions that are commonly needed across the framework. It's designed to be zero-dependency and highly performant.

## Features

- **Zero dependencies** - No external dependencies, pure Go
- **High performance** - Optimized for speed and minimal allocations
- **Type safety** - Strongly typed utility functions
- **Comprehensive testing** - Well-tested utility functions
- **Documentation** - Clear documentation with examples

## Quick Start

```go
import "github.com/eggybyte-technology/egg/core/utils"

// String utilities
isEmpty := utils.IsEmpty("")
isValid := utils.IsValidEmail("user@example.com")

// Time utilities
duration := utils.ParseDuration("1h30m")
timestamp := utils.FormatTimestamp(time.Now())

// Validation utilities
isValid := utils.IsValidUUID("550e8400-e29b-41d4-a716-446655440000")
```

## API Reference

### String Utilities

```go
// IsEmpty checks if a string is empty or contains only whitespace
func IsEmpty(s string) bool

// IsNotEmpty checks if a string is not empty and contains non-whitespace characters
func IsNotEmpty(s string) bool

// IsValidEmail validates email format
func IsValidEmail(email string) bool

// IsValidUUID validates UUID format
func IsValidUUID(uuid string) bool

// Truncate truncates a string to the specified length
func Truncate(s string, maxLen int) string

// Sanitize removes potentially dangerous characters from a string
func Sanitize(s string) string
```

### Time Utilities

```go
// ParseDuration parses a duration string with common formats
func ParseDuration(s string) (time.Duration, error)

// FormatTimestamp formats a timestamp for logging
func FormatTimestamp(t time.Time) string

// IsValidTimezone checks if a timezone is valid
func IsValidTimezone(tz string) bool

// GetTimezoneOffset returns the timezone offset in seconds
func GetTimezoneOffset(tz string) (int, error)
```

### Validation Utilities

```go
// IsValidURL validates URL format
func IsValidURL(url string) bool

// IsValidIP validates IP address format
func IsValidIP(ip string) bool

// IsValidPort validates port number
func IsValidPort(port int) bool

// IsValidHostname validates hostname format
func IsValidHostname(hostname string) bool
```

### Conversion Utilities

```go
// StringToInt converts string to int with error handling
func StringToInt(s string) (int, error)

// StringToFloat64 converts string to float64 with error handling
func StringToFloat64(s string) (float64, error)

// IntToString converts int to string
func IntToString(i int) string

// Float64ToString converts float64 to string
func Float64ToString(f float64) string
```

## Usage Examples

### String Validation

```go
func validateUserInput(input string) error {
    if utils.IsEmpty(input) {
        return errors.New("VALIDATION_ERROR", "input cannot be empty")
    }
    
    if !utils.IsValidEmail(input) {
        return errors.New("VALIDATION_ERROR", "invalid email format")
    }
    
    return nil
}

func sanitizeUserInput(input string) string {
    return utils.Sanitize(input)
}
```

### Time Handling

```go
func parseConfigDuration(durationStr string) (time.Duration, error) {
    duration, err := utils.ParseDuration(durationStr)
    if err != nil {
        return 0, errors.Wrap(err, "CONFIG_ERROR", "invalid duration format")
    }
    
    return duration, nil
}

func logWithTimestamp(logger log.Logger, msg string) {
    timestamp := utils.FormatTimestamp(time.Now())
    logger.Info(msg, log.Str("timestamp", timestamp))
}
```

### URL and Network Validation

```go
func validateServiceConfig(config *ServiceConfig) error {
    if !utils.IsValidURL(config.Endpoint) {
        return errors.New("VALIDATION_ERROR", "invalid endpoint URL")
    }
    
    if !utils.IsValidPort(config.Port) {
        return errors.New("VALIDATION_ERROR", "invalid port number")
    }
    
    if !utils.IsValidHostname(config.Hostname) {
        return errors.New("VALIDATION_ERROR", "invalid hostname")
    }
    
    return nil
}
```

### Data Conversion

```go
func parseConfigValue(value string, targetType string) (interface{}, error) {
    switch targetType {
    case "int":
        return utils.StringToInt(value)
    case "float64":
        return utils.StringToFloat64(value)
    case "string":
        return value, nil
    default:
        return nil, errors.New("CONFIG_ERROR", "unsupported type")
    }
}
```

### UUID Validation

```go
func validateUserID(userID string) error {
    if !utils.IsValidUUID(userID) {
        return errors.New("VALIDATION_ERROR", "invalid user ID format")
    }
    
    return nil
}

func generateUserID() string {
    return uuid.New().String()
}
```

## Implementation Examples

### String Utilities Implementation

```go
func IsEmpty(s string) bool {
    return len(strings.TrimSpace(s)) == 0
}

func IsNotEmpty(s string) bool {
    return !IsEmpty(s)
}

func IsValidEmail(email string) bool {
    if IsEmpty(email) {
        return false
    }
    
    // Basic email validation regex
    emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    return emailRegex.MatchString(email)
}

func IsValidUUID(uuid string) bool {
    if IsEmpty(uuid) {
        return false
    }
    
    // UUID validation regex
    uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
    return uuidRegex.MatchString(strings.ToLower(uuid))
}

func Truncate(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    
    if maxLen <= 3 {
        return s[:maxLen]
    }
    
    return s[:maxLen-3] + "..."
}

func Sanitize(s string) string {
    // Remove potentially dangerous characters
    dangerous := []string{"<", ">", "\"", "'", "&", "\n", "\r", "\t"}
    result := s
    
    for _, char := range dangerous {
        result = strings.ReplaceAll(result, char, "")
    }
    
    return strings.TrimSpace(result)
}
```

### Time Utilities Implementation

```go
func ParseDuration(s string) (time.Duration, error) {
    if IsEmpty(s) {
        return 0, errors.New("empty duration string")
    }
    
    // Try standard duration parsing first
    if duration, err := time.ParseDuration(s); err == nil {
        return duration, nil
    }
    
    // Try common formats
    formats := map[string]string{
        "1h":     "1h0m0s",
        "30m":    "0h30m0s",
        "1d":     "24h0m0s",
        "1w":     "168h0m0s",
    }
    
    if standardFormat, exists := formats[s]; exists {
        return time.ParseDuration(standardFormat)
    }
    
    return 0, errors.New("invalid duration format")
}

func FormatTimestamp(t time.Time) string {
    return t.Format("2006-01-02T15:04:05.000Z07:00")
}

func IsValidTimezone(tz string) bool {
    if IsEmpty(tz) {
        return false
    }
    
    _, err := time.LoadLocation(tz)
    return err == nil
}

func GetTimezoneOffset(tz string) (int, error) {
    if IsEmpty(tz) {
        return 0, errors.New("empty timezone")
    }
    
    location, err := time.LoadLocation(tz)
    if err != nil {
        return 0, err
    }
    
    now := time.Now()
    _, offset := now.In(location).Zone()
    return offset, nil
}
```

## Testing

```go
func TestStringUtilities(t *testing.T) {
    // Test IsEmpty
    assert.True(t, utils.IsEmpty(""))
    assert.True(t, utils.IsEmpty("   "))
    assert.False(t, utils.IsEmpty("hello"))
    
    // Test IsValidEmail
    assert.True(t, utils.IsValidEmail("user@example.com"))
    assert.False(t, utils.IsValidEmail("invalid-email"))
    assert.False(t, utils.IsValidEmail(""))
    
    // Test IsValidUUID
    assert.True(t, utils.IsValidUUID("550e8400-e29b-41d4-a716-446655440000"))
    assert.False(t, utils.IsValidUUID("invalid-uuid"))
    assert.False(t, utils.IsValidUUID(""))
    
    // Test Truncate
    assert.Equal(t, "hello", utils.Truncate("hello", 10))
    assert.Equal(t, "he...", utils.Truncate("hello world", 5))
    
    // Test Sanitize
    assert.Equal(t, "hello", utils.Sanitize("hello<script>"))
    assert.Equal(t, "test", utils.Sanitize("test\n\r\t"))
}

func TestTimeUtilities(t *testing.T) {
    // Test ParseDuration
    duration, err := utils.ParseDuration("1h")
    assert.NoError(t, err)
    assert.Equal(t, time.Hour, duration)
    
    duration, err = utils.ParseDuration("30m")
    assert.NoError(t, err)
    assert.Equal(t, 30*time.Minute, duration)
    
    // Test FormatTimestamp
    timestamp := utils.FormatTimestamp(time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC))
    assert.Contains(t, timestamp, "2023-01-01T12:00:00")
    
    // Test IsValidTimezone
    assert.True(t, utils.IsValidTimezone("UTC"))
    assert.True(t, utils.IsValidTimezone("America/New_York"))
    assert.False(t, utils.IsValidTimezone("Invalid/Timezone"))
}

func TestValidationUtilities(t *testing.T) {
    // Test IsValidURL
    assert.True(t, utils.IsValidURL("https://example.com"))
    assert.True(t, utils.IsValidURL("http://localhost:8080"))
    assert.False(t, utils.IsValidURL("invalid-url"))
    
    // Test IsValidIP
    assert.True(t, utils.IsValidIP("192.168.1.1"))
    assert.True(t, utils.IsValidIP("::1"))
    assert.False(t, utils.IsValidIP("invalid-ip"))
    
    // Test IsValidPort
    assert.True(t, utils.IsValidPort(8080))
    assert.True(t, utils.IsValidPort(1))
    assert.False(t, utils.IsValidPort(0))
    assert.False(t, utils.IsValidPort(65536))
    
    // Test IsValidHostname
    assert.True(t, utils.IsValidHostname("example.com"))
    assert.True(t, utils.IsValidHostname("localhost"))
    assert.False(t, utils.IsValidHostname(""))
}

func TestConversionUtilities(t *testing.T) {
    // Test StringToInt
    value, err := utils.StringToInt("123")
    assert.NoError(t, err)
    assert.Equal(t, 123, value)
    
    _, err = utils.StringToInt("invalid")
    assert.Error(t, err)
    
    // Test StringToFloat64
    value, err := utils.StringToFloat64("123.45")
    assert.NoError(t, err)
    assert.Equal(t, 123.45, value)
    
    _, err = utils.StringToFloat64("invalid")
    assert.Error(t, err)
    
    // Test IntToString
    assert.Equal(t, "123", utils.IntToString(123))
    
    // Test Float64ToString
    assert.Equal(t, "123.45", utils.Float64ToString(123.45))
}
```

## Best Practices

### 1. Use Validation Utilities

```go
func validateConfig(config *Config) error {
    if !utils.IsValidURL(config.Endpoint) {
        return errors.New("VALIDATION_ERROR", "invalid endpoint URL")
    }
    
    if !utils.IsValidPort(config.Port) {
        return errors.New("VALIDATION_ERROR", "invalid port number")
    }
    
    return nil
}
```

### 2. Sanitize User Input

```go
func processUserInput(input string) string {
    // Always sanitize user input
    return utils.Sanitize(input)
}
```

### 3. Use Type-Safe Conversions

```go
func parseConfigValue(value string) (int, error) {
    // Use type-safe conversion with error handling
    return utils.StringToInt(value)
}
```

### 4. Validate Before Processing

```go
func processUser(user *User) error {
    // Validate before processing
    if utils.IsEmpty(user.Email) {
        return errors.New("VALIDATION_ERROR", "email is required")
    }
    
    if !utils.IsValidEmail(user.Email) {
        return errors.New("VALIDATION_ERROR", "invalid email format")
    }
    
    // Process user
    return nil
}
```

## Thread Safety

All functions in this package are safe for concurrent use. They are stateless and don't maintain any internal state.

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
