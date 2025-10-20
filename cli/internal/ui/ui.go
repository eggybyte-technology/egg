// Package ui provides unified output formatting for the egg CLI.
//
// Overview:
//   - Responsibility: Standardized logging, progress indication, and user interaction
//   - Key Types: Output formatters, progress indicators
//   - Concurrency Model: Thread-safe output operations
//   - Error Semantics: User-friendly error messages with suggestions
//   - Performance Notes: Buffered output, minimal allocations
//
// Usage:
//
//	ui.Info("Operation completed successfully")
//	ui.Error("Failed to create service: %v", err)
package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var (
	verbose        bool
	nonInteractive bool
	jsonOutput     bool
	mu             sync.RWMutex
)

// OutputLevel represents the severity level of a message.
type OutputLevel string

const (
	LevelDebug   OutputLevel = "debug"
	LevelInfo    OutputLevel = "info"
	LevelWarning OutputLevel = "warning"
	LevelError   OutputLevel = "error"
	LevelSuccess OutputLevel = "success"
)

// Message represents a structured output message.
//
// Parameters:
//   - Level: Message severity level
//   - Text: Human-readable message content
//   - Data: Optional structured data for JSON output
//   - Timestamp: When the message was created
//
// Returns:
//   - None (data structure)
//
// Concurrency:
//   - Safe for concurrent access
//
// Performance:
//   - Minimal memory allocation
type Message struct {
	Level     OutputLevel `json:"level"`
	Text      string      `json:"text"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// SetVerbose enables or disables verbose output.
//
// Parameters:
//   - enabled: Whether to show debug messages
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(1) operation
func SetVerbose(enabled bool) {
	mu.Lock()
	defer mu.Unlock()
	verbose = enabled
}

// SetNonInteractive disables interactive prompts.
//
// Parameters:
//   - enabled: Whether to disable interactive prompts
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(1) operation
func SetNonInteractive(enabled bool) {
	mu.Lock()
	defer mu.Unlock()
	nonInteractive = enabled
}

// SetJSONOutput enables JSON-formatted output.
//
// Parameters:
//   - enabled: Whether to output in JSON format
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(1) operation
func SetJSONOutput(enabled bool) {
	mu.Lock()
	defer mu.Unlock()
	jsonOutput = enabled
}

// output writes a message to the appropriate output stream.
//
// Parameters:
//   - level: Message severity level
//   - format: Printf-style format string
//   - args: Format arguments
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - Buffered output, minimal allocations
func output(level OutputLevel, format string, args ...interface{}) {
	mu.RLock()
	useJSON := jsonOutput
	useVerbose := verbose
	mu.RUnlock()

	// Skip debug messages if not verbose
	if level == LevelDebug && !useVerbose {
		return
	}

	text := fmt.Sprintf(format, args...)
	message := Message{
		Level:     level,
		Text:      text,
		Timestamp: time.Now(),
	}

	if useJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(message); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to encode JSON output: %v\n", err)
		}
		return
	}

	// Choose output stream based on level
	var writer io.Writer = os.Stdout
	if level == LevelError {
		writer = os.Stderr
	}

	// Format with color and prefix
	var prefix string
	switch level {
	case LevelDebug:
		prefix = "🔍 DEBUG:"
	case LevelInfo:
		prefix = "ℹ️  INFO:"
	case LevelWarning:
		prefix = "⚠️  WARN:"
	case LevelError:
		prefix = "❌ ERROR:"
	case LevelSuccess:
		prefix = "✅ SUCCESS:"
	}

	fmt.Fprintf(writer, "%s %s\n", prefix, text)
}

// Debug outputs a debug message.
//
// Parameters:
//   - format: Printf-style format string
//   - args: Format arguments
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - Only shown when verbose mode is enabled
func Debug(format string, args ...interface{}) {
	output(LevelDebug, format, args...)
}

// Info outputs an informational message.
//
// Parameters:
//   - format: Printf-style format string
//   - args: Format arguments
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - Always shown unless JSON mode is enabled
func Info(format string, args ...interface{}) {
	output(LevelInfo, format, args...)
}

// Warning outputs a warning message.
//
// Parameters:
//   - format: Printf-style format string
//   - args: Format arguments
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - Always shown unless JSON mode is enabled
func Warning(format string, args ...interface{}) {
	output(LevelWarning, format, args...)
}

// Error outputs an error message.
//
// Parameters:
//   - format: Printf-style format string
//   - args: Format arguments
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - Always shown, goes to stderr
func Error(format string, args ...interface{}) {
	output(LevelError, format, args...)
}

// Success outputs a success message.
//
// Parameters:
//   - format: Printf-style format string
//   - args: Format arguments
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - Always shown unless JSON mode is enabled
func Success(format string, args ...interface{}) {
	output(LevelSuccess, format, args...)
}

// Step outputs a step indicator with message.
//
// Parameters:
//   - step: Step number
//   - total: Total number of steps
//   - format: Printf-style format string
//   - args: Format arguments
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - Minimal formatting overhead
func Step(step, total int, format string, args ...interface{}) {
	mu.RLock()
	useJSON := jsonOutput
	mu.RUnlock()

	if useJSON {
		Info(format, args...)
		return
	}

	text := fmt.Sprintf(format, args...)
	fmt.Printf("  [%d/%d] %s\n", step, total, text)
}

// Confirm prompts the user for confirmation.
//
// Parameters:
//   - format: Printf-style format string
//   - args: Format arguments
//
// Returns:
//   - bool: True if user confirmed, false otherwise
//
// Concurrency:
//   - Single-threaded (blocks on user input)
//
// Performance:
//   - Blocks until user responds
func Confirm(format string, args ...interface{}) bool {
	mu.RLock()
	nonInt := nonInteractive
	mu.RUnlock()

	if nonInt {
		return true // Auto-confirm in non-interactive mode
	}

	text := fmt.Sprintf(format, args...)
	fmt.Printf("❓ %s [y/N]: ", text)

	var response string
	fmt.Scanln(&response)
	return response == "y" || response == "Y" || response == "yes"
}

// Progress represents a progress indicator.
type Progress struct {
	total   int
	current int
	title   string
	mu      sync.Mutex
}

// NewProgress creates a new progress indicator.
//
// Parameters:
//   - title: Progress title
//   - total: Total number of items
//
// Returns:
//   - *Progress: Progress indicator instance
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Minimal initialization overhead
func NewProgress(title string, total int) *Progress {
	return &Progress{
		total:   total,
		current: 0,
		title:   title,
	}
}

// Update increments the progress and updates display.
//
// Parameters:
//   - None (uses internal state)
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(1) operation
func (p *Progress) Update() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current++

	mu.RLock()
	useJSON := jsonOutput
	mu.RUnlock()

	if useJSON {
		return // Don't show progress in JSON mode
	}

	percentage := float64(p.current) / float64(p.total) * 100
	fmt.Printf("\r🔄 %s: %d/%d (%.1f%%)", p.title, p.current, p.total, percentage)

	if p.current >= p.total {
		fmt.Println() // New line when complete
	}
}

// Complete marks the progress as complete.
//
// Parameters:
//   - None
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(1) operation
func (p *Progress) Complete() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = p.total

	mu.RLock()
	useJSON := jsonOutput
	mu.RUnlock()

	if !useJSON {
		fmt.Printf("\r✅ %s: Complete\n", p.title)
	}
}
