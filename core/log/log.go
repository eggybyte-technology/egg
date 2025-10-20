// Package log provides a minimal logging interface compatible with slog concepts.
//
// Overview:
//   - Responsibility: Define a stable logging interface for the egg framework
//   - Key Types: Logger interface with structured key-value logging
//   - Concurrency Model: Logger implementations must be safe for concurrent use
//   - Error Semantics: Error method accepts error as first parameter for structured logging
//   - Performance Notes: Interface designed for zero-allocation key-value pairs
//
// Usage:
//
//	logger := yourImplementation{}
//	logger.Info("user login", log.Str("user_id", "123"), log.Int("attempt", 1))
package log

import "time"

// Logger defines a structured logging interface compatible with slog concepts.
// Implementations must be safe for concurrent use.
type Logger interface {
	// With returns a new Logger with the given key-value pairs attached.
	// The returned Logger should share the same underlying implementation
	// but with additional context.
	With(kv ...any) Logger

	// Debug logs a debug message with optional key-value pairs.
	Debug(msg string, kv ...any)

	// Info logs an informational message with optional key-value pairs.
	Info(msg string, kv ...any)

	// Warn logs a warning message with optional key-value pairs.
	Warn(msg string, kv ...any)

	// Error logs an error message with the error and optional key-value pairs.
	// The error should be the first parameter for structured error handling.
	Error(err error, msg string, kv ...any)
}

// Str creates a string key-value pair for structured logging.
// This is a convenience function for creating key-value pairs.
func Str(k, v string) any {
	return []any{k, v}
}

// Int creates an integer key-value pair for structured logging.
// This is a convenience function for creating key-value pairs.
func Int(k string, v int) any {
	return []any{k, v}
}

// Dur creates a duration key-value pair for structured logging.
// This is a convenience function for creating key-value pairs.
func Dur(k string, v time.Duration) any {
	return []any{k, v}
}
