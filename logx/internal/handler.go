// Package internal provides internal implementation details for logx.
package internal

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

// Options configures the logger behavior.
type Options struct {
	Format           string     // Output format: logfmt or json
	Level            slog.Level // Minimum log level
	Color            bool       // Enable colorization for level field only
	Writer           io.Writer  // Output writer (default: os.Stderr)
	PayloadMaxBytes  int        // Maximum bytes to log for large payloads (0 = unlimited)
	SensitiveFields  []string   // Field names to mask (e.g., "password", "token")
	DisableTimestamp bool       // Disable timestamp in output
	DisableCaller    bool       // Disable caller information
}

// Handler is a custom slog.Handler that outputs logfmt with sorted fields.
type Handler struct {
	opts   Options
	mu     sync.Mutex
	writer io.Writer
	attrs  []slog.Attr
	group  string
}

// NewHandler creates a new Handler with the given options.
func NewHandler(opts Options, writer io.Writer) *Handler {
	return &Handler{
		opts:   opts,
		writer: writer,
	}
}

// handle writes the log record (internal method).
func (h *Handler) handle(level slog.Level, msg string, attrs []slog.Attr) {
	// Check if level is enabled
	if level < h.opts.Level {
		return
	}

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
	levelStr := LevelString(level)
	buf.WriteString("level=")
	if h.opts.Color {
		buf.WriteString(ColorizeLevel(levelStr))
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
	sortedAttrs := SortAttrs(allAttrs)

	// Add sorted attributes in key=value format
	for _, attr := range sortedAttrs {
		buf.WriteString(" ")
		buf.WriteString(attr.Key)
		buf.WriteString("=")
		buf.WriteString(FormatValue(attr.Key, attr.Value, h.opts))
	}

	buf.WriteString("\n")

	// Write to output
	h.writer.Write([]byte(buf.String()))
}

// LogRecord writes a log record (public method for logx package).
func (h *Handler) LogRecord(level slog.Level, msg string, attrs []slog.Attr) {
	h.handle(level, msg, attrs)
}

// Enabled reports whether the handler handles records at the given level.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level
}

// Handle implements slog.Handler (renamed from HandleRecord).
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

// KVToAttrs converts key-value pairs to slog.Attr slice.
func KVToAttrs(kv []any) []slog.Attr {
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

// SortAttrs sorts attributes by key.
func SortAttrs(attrs []slog.Attr) []slog.Attr {
	sorted := make([]slog.Attr, len(attrs))
	copy(sorted, attrs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Key < sorted[j].Key
	})
	return sorted
}

// FormatValue formats a slog.Value for logfmt output.
func FormatValue(key string, v slog.Value, opts Options) string {
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

// LevelString returns the string representation of a log level.
func LevelString(level slog.Level) string {
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

// ColorizeLevel adds ANSI color codes ONLY to the level value.
func ColorizeLevel(level string) string {
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
