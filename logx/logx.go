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
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/eggybyte-technology/egg/core/identity"
	"github.com/eggybyte-technology/egg/core/log"
)

// Format specifies the output format for logs.
type Format string

const (
	// FormatLogfmt outputs logs in logfmt format (key=value pairs).
	FormatLogfmt Format = "logfmt"
	// FormatJSON outputs logs in JSON format.
	FormatJSON Format = "json"
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
	handler *Handler
	attrs   []slog.Attr
}

// Handler is a custom slog.Handler that outputs logfmt with sorted fields.
type Handler struct {
	opts   Options
	mu     sync.Mutex
	writer io.Writer
	attrs  []slog.Attr
	group  string
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

	handler := &Handler{
		opts:   options,
		writer: options.Writer,
	}

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
	attrs := kvToAttrs(kv)
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
	attrs := kvToAttrs(kv)
	if err != nil {
		attrs = append([]slog.Attr{slog.Any("error", err)}, attrs...)
	}
	l.logWithAttrs(slog.LevelError, msg, attrs)
}

// log is the internal logging method.
func (l *Logger) log(level slog.Level, msg string, kv ...any) {
	attrs := kvToAttrs(kv)
	l.logWithAttrs(level, msg, attrs)
}

// logWithAttrs logs with pre-converted attributes.
func (l *Logger) logWithAttrs(level slog.Level, msg string, attrs []slog.Attr) {
	if level < l.handler.opts.Level {
		return
	}

	// Combine logger attrs with call attrs
	allAttrs := append([]slog.Attr{}, l.attrs...)
	allAttrs = append(allAttrs, attrs...)

	l.handler.handle(level, msg, allAttrs)
}

// handle writes the log record.
func (h *Handler) handle(level slog.Level, msg string, attrs []slog.Attr) {
	h.mu.Lock()
	defer h.mu.Unlock()

	var buf strings.Builder

	// Add timestamp if not disabled (usually disabled in containers)
	if !h.opts.DisableTimestamp {
		timestamp := time.Now().Format(time.RFC3339)
		buf.WriteString("time=")
		buf.WriteString(timestamp)
		buf.WriteString(" ")
	}

	// Add level (only field with color)
	levelStr := levelString(level)
	buf.WriteString("level=")
	if h.opts.Color {
		buf.WriteString(colorizeLevel(levelStr))
	} else {
		buf.WriteString(levelStr)
	}

	// Add message (always quoted for consistency)
	buf.WriteString(" msg=")
	buf.WriteString(fmt.Sprintf("%q", msg))

	// Combine handler attrs with record attrs
	allAttrs := append([]slog.Attr{}, h.attrs...)
	allAttrs = append(allAttrs, attrs...)

	// Sort attributes by key for stable output
	sortedAttrs := sortAttrs(allAttrs)

	// Add sorted attributes in key=value format
	for _, attr := range sortedAttrs {
		buf.WriteString(" ")
		buf.WriteString(attr.Key)
		buf.WriteString("=")
		buf.WriteString(formatValue(attr.Key, attr.Value, h.opts))
	}

	buf.WriteString("\n")

	// Write to output
	h.writer.Write([]byte(buf.String()))
}

// Enabled reports whether the handler handles records at the given level.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level
}

// Handle implements slog.Handler.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	attrs := make([]slog.Attr, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})
	h.handle(r.Level, r.Message, attrs)
	return nil
}

// WithAttrs returns a new Handler with the given attributes.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := append([]slog.Attr{}, h.attrs...)
	newAttrs = append(newAttrs, attrs...)

	return &Handler{
		opts:   h.opts,
		writer: h.writer,
		attrs:  newAttrs,
		group:  h.group,
	}
}

// WithGroup returns a new Handler with the given group name.
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		opts:   h.opts,
		writer: h.writer,
		attrs:  h.attrs,
		group:  name,
	}
}

// kvToAttrs converts key-value pairs to slog.Attr slice.
// It also flattens helper pairs created by core/log helpers, which pass
// key-value as []any{"key", value} inside the variadic list.
func kvToAttrs(kv []any) []slog.Attr {
	// First, expand any nested []any pairs to a flat key, value sequence.
	flat := make([]any, 0, len(kv))
	for _, item := range kv {
		switch v := item.(type) {
		case []any:
			// If it's an even-length 2-tuple-like pair, append as-is.
			if len(v) == 2 {
				flat = append(flat, v[0], v[1])
			} else {
				// Fallback: append the slice itself as a single value
				flat = append(flat, v)
			}
		default:
			flat = append(flat, v)
		}
	}

	// Now, consume flat as key,value pairs.
	attrs := make([]slog.Attr, 0, len(flat)/2)
	for i := 0; i < len(flat)-1; i += 2 {
		key := fmt.Sprintf("%v", flat[i])
		value := flat[i+1]
		attrs = append(attrs, slog.Any(key, value))
	}
	return attrs
}

// sortAttrs sorts attributes by key.
func sortAttrs(attrs []slog.Attr) []slog.Attr {
	sorted := make([]slog.Attr, len(attrs))
	copy(sorted, attrs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Key < sorted[j].Key
	})
	return sorted
}

// formatValue formats a slog.Value for logfmt output.
func formatValue(key string, v slog.Value, opts Options) string {
	// Check if this is a sensitive field (by key name)
	for _, field := range opts.SensitiveFields {
		if strings.EqualFold(key, field) {
			return `"***REDACTED***"`
		}
	}

	switch v.Kind() {
	case slog.KindString:
		s := v.String()
		// Apply payload limit
		if opts.PayloadMaxBytes > 0 && len(s) > opts.PayloadMaxBytes {
			truncated := fmt.Sprintf("%s...(truncated, %d bytes)", s[:opts.PayloadMaxBytes], len(s))
			return fmt.Sprintf("%q", truncated)
		}
		// Always quote strings for logfmt consistency
		return fmt.Sprintf("%q", s)
	case slog.KindInt64:
		return fmt.Sprintf("%d", v.Int64())
	case slog.KindUint64:
		return fmt.Sprintf("%d", v.Uint64())
	case slog.KindFloat64:
		f := v.Float64()
		// Format floats cleanly (remove trailing zeros)
		if f == float64(int64(f)) {
			return fmt.Sprintf("%.0f", f)
		}
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", f), "0"), ".")
	case slog.KindBool:
		return fmt.Sprintf("%t", v.Bool())
	case slog.KindDuration:
		// Format duration in milliseconds for consistency
		ms := v.Duration().Milliseconds()
		return fmt.Sprintf("%d", ms)
	case slog.KindTime:
		return fmt.Sprintf("%q", v.Time().Format(time.RFC3339))
	default:
		return fmt.Sprintf("%q", v.String())
	}
}

// quoteIfNeeded quotes a string if it contains spaces or special characters.
func quoteIfNeeded(s string) string {
	if strings.ContainsAny(s, " \t\n\r\"=") {
		return fmt.Sprintf("%q", s)
	}
	return s
}

// levelString returns the string representation of a log level.
func levelString(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "DEBUG"
	case slog.LevelInfo:
		return "INFO"
	case slog.LevelWarn:
		return "WARN"
	case slog.LevelError:
		return "ERROR"
	default:
		return fmt.Sprintf("LEVEL(%d)", level)
	}
}

// colorizeLevel adds ANSI color codes ONLY to the level value.
// This ensures only the level field is colored, not the key.
func colorizeLevel(level string) string {
	const (
		reset   = "\033[0m"
		red     = "\033[31m"
		yellow  = "\033[33m"
		cyan    = "\033[36m"
		magenta = "\033[35m"
	)

	switch level {
	case "DEBUG":
		return magenta + level + reset
	case "INFO":
		return cyan + level + reset
	case "WARN":
		return yellow + level + reset
	case "ERROR":
		return red + level + reset
	default:
		return level
	}
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
