// Package logx provides a structured logging implementation based on slog.
//
// Overview:
//   - Responsibility: Unified logging with logfmt/JSON output, field sorting, and colorization
//   - Key Types: Logger implementation, Handler for slog, Options for configuration
//   - Concurrency Model: All loggers are safe for concurrent use
//   - Error Semantics: No errors returned; logging failures are silently handled
//   - Performance Notes: Optimized for production with field sorting and optional payload limits
//
// Usage:
//
//	logger := logx.New(logx.WithFormat("logfmt"), logx.WithColor(true))
//	logger.Info("user created", logx.Str("user_id", "u-123"))
package logx

import (
	"context"
	"io"
	"log/slog"
	"os"

	"go.eggybyte.com/egg/core/identity"
	"go.eggybyte.com/egg/core/log"
	"go.eggybyte.com/egg/logx/internal"
)

// Format specifies the output format for logs.
type Format string

const (
	// FormatLogfmt outputs logs in logfmt format (key=value pairs).
	FormatLogfmt Format = "logfmt"
	// FormatJSON outputs logs in JSON format.
	FormatJSON Format = "json"
	// FormatConsole outputs logs in a human-readable console format with colors and indentation.
	FormatConsole Format = "console"
)

// Options configures the logger behavior.
type Options struct {
	Format           Format     // Output format: logfmt or json
	Level            slog.Level // Minimum log level
	Color            bool       // Enable colorization for level field only
	Writer           io.Writer  // Output writer (default: os.Stderr)
	PayloadMaxBytes  int        // Maximum bytes to log for large payloads (0 = unlimited)
	SensitiveFields  []string   // Field names to mask (e.g., "password", "token")
	DisableTimestamp bool       // Disable timestamp in output
	DisableCaller    bool       // Disable caller information
}

// Logger implements the core/log.Logger interface using slog.
type Logger struct {
	handler *internal.Handler
	attrs   []slog.Attr
}

// New creates a new Logger with the given options.
func New(opts ...Option) log.Logger {
	options := Options{
		Format:           FormatLogfmt,
		Level:            slog.LevelInfo,
		Color:            false,
		Writer:           os.Stderr,
		DisableTimestamp: true, // Container already adds timestamp
	}

	for _, opt := range opts {
		opt(&options)
	}

	if options.Writer == nil {
		options.Writer = os.Stderr
	}

	handler := internal.NewHandler(internal.Options{
		Format:           string(options.Format),
		Level:            options.Level,
		Color:            options.Color,
		Writer:           options.Writer,
		PayloadMaxBytes:  options.PayloadMaxBytes,
		SensitiveFields:  options.SensitiveFields,
		DisableTimestamp: options.DisableTimestamp,
		DisableCaller:    options.DisableCaller,
	}, options.Writer)

	return &Logger{
		handler: handler,
	}
}

// Option configures logger behavior.
type Option func(*Options)

// WithFormat sets the output format.
func WithFormat(format Format) Option {
	return func(o *Options) {
		o.Format = format
	}
}

// WithLevel sets the minimum log level.
func WithLevel(level slog.Level) Option {
	return func(o *Options) {
		o.Level = level
	}
}

// WithColor enables colorization for the level field only.
func WithColor(enabled bool) Option {
	return func(o *Options) {
		o.Color = enabled
	}
}

// WithWriter sets the output writer.
func WithWriter(w io.Writer) Option {
	return func(o *Options) {
		o.Writer = w
	}
}

// WithPayloadLimit sets the maximum bytes to log for large payloads.
func WithPayloadLimit(maxBytes int) Option {
	return func(o *Options) {
		o.PayloadMaxBytes = maxBytes
	}
}

// WithSensitiveFields sets field names to mask in logs.
func WithSensitiveFields(fields ...string) Option {
	return func(o *Options) {
		o.SensitiveFields = fields
	}
}

// With returns a new Logger with the given key-value pairs attached.
func (l *Logger) With(kv ...any) log.Logger {
	attrs := internal.KVToAttrs(kv)
	newAttrs := append([]slog.Attr{}, l.attrs...)
	newAttrs = append(newAttrs, attrs...)

	return &Logger{
		handler: l.handler,
		attrs:   newAttrs,
	}
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, kv ...any) {
	l.log(slog.LevelDebug, msg, kv...)
}

// Info logs an informational message.
func (l *Logger) Info(msg string, kv ...any) {
	l.log(slog.LevelInfo, msg, kv...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, kv ...any) {
	l.log(slog.LevelWarn, msg, kv...)
}

// Error logs an error message.
func (l *Logger) Error(err error, msg string, kv ...any) {
	attrs := internal.KVToAttrs(kv)
	if err != nil {
		attrs = append([]slog.Attr{slog.Any("error", err)}, attrs...)
	}
	l.logWithAttrs(slog.LevelError, msg, attrs)
}

// log is the internal logging method.
func (l *Logger) log(level slog.Level, msg string, kv ...any) {
	attrs := internal.KVToAttrs(kv)
	l.logWithAttrs(level, msg, attrs)
}

// logWithAttrs logs with pre-converted attributes.
func (l *Logger) logWithAttrs(level slog.Level, msg string, attrs []slog.Attr) {
	// Combine logger attrs with call attrs
	allAttrs := append([]slog.Attr{}, l.attrs...)
	allAttrs = append(allAttrs, attrs...)

	l.handler.LogRecord(level, msg, allAttrs)
}

// FromContext creates a logger with context-injected fields (trace_id, request_id, user_id).
func FromContext(ctx context.Context, base log.Logger) log.Logger {
	var attrs []any

	// Extract request metadata
	if meta, ok := identity.MetaFrom(ctx); ok {
		if meta.RequestID != "" {
			attrs = append(attrs, "request_id", meta.RequestID)
		}
	}

	// Extract user info
	if user, ok := identity.UserFrom(ctx); ok {
		if user.UserID != "" {
			attrs = append(attrs, "user_id", user.UserID)
		}
	}

	if len(attrs) > 0 {
		return base.With(attrs...)
	}
	return base
}

// ParseLevel converts a log level string to slog.Level.
//
// Supported values (case-insensitive):
//   - "debug", "DEBUG" -> slog.LevelDebug (-4)
//   - "info", "INFO" -> slog.LevelInfo (0)
//   - "warn", "warning", "WARN", "WARNING" -> slog.LevelWarn (4)
//   - "error", "ERROR" -> slog.LevelError (8)
//
// Parameters:
//   - level: log level string from configuration
//
// Returns:
//   - slog.Level: parsed log level (defaults to Info if unrecognized)
//
// Examples:
//
//	level := logx.ParseLevel("debug")  // returns slog.LevelDebug
//	level := logx.ParseLevel("INFO")   // returns slog.LevelInfo
//	level := logx.ParseLevel("invalid") // returns slog.LevelInfo (default)
//
// Concurrency:
//
//	Safe for concurrent use (pure function, no shared state).
func ParseLevel(level string) slog.Level {
	switch level {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "info", "INFO":
		return slog.LevelInfo
	case "warn", "warning", "WARN", "WARNING":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo // Default to Info for unrecognized values
	}
}

// NewFromEnv creates a new Logger with log level read from LOG_LEVEL environment variable.
//
// This is a convenience function for microservices that follows the egg framework's
// configuration pattern. It automatically:
//   - Reads LOG_LEVEL from environment (defaults to "info" if not set)
//   - Uses console format for human-readable output
//   - Enables colorization for development
//
// Environment Variables:
//   - LOG_LEVEL: Log level (debug, info, warn, error); default: "info"
//
// Returns:
//   - log.Logger: configured logger instance ready for use
//
// Examples:
//
//	// In main.go:
//	logger := logx.NewFromEnv()
//	logger.Info("service starting")
//
//	// With LOG_LEVEL=debug:
//	// export LOG_LEVEL=debug
//	// ./my-service
//
// Usage in servicex:
//
//	err := servicex.Run(ctx,
//	    servicex.WithService("my-service", "1.0.0"),
//	    servicex.WithLogger(logx.NewFromEnv()),  // Automatically reads LOG_LEVEL
//	    // ... other options
//	)
//
// Concurrency:
//
//	Safe to call from multiple goroutines (reads environment once).
func NewFromEnv() log.Logger {
	logLevelStr := os.Getenv("LOG_LEVEL")
	if logLevelStr == "" {
		logLevelStr = "info" // default
	}

	logLevel := ParseLevel(logLevelStr)

	return New(
		WithFormat(FormatConsole),
		WithLevel(logLevel),
		WithColor(true),
	)
}
