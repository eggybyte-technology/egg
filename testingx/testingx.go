// Package testingx provides testing utilities for egg framework.
//
// Overview:
//   - Responsibility: Testing helpers, mocks, and fixtures
//   - Key Types: MockLogger, test helpers for identity and errors
//   - Concurrency Model: Thread-safe where needed
//   - Error Semantics: Test failures via testing.T
//   - Performance Notes: Optimized for test execution
//
// Usage:
//
//	logger := testingx.NewMockLogger(t)
//	ctx := testingx.NewContextWithIdentity(t, &identity.UserInfo{UserID: "u-123"})
package testingx

import (
	"bytes"
	"context"
	"sync"
	"testing"

	"go.eggybyte.com/egg/core/errors"
	"go.eggybyte.com/egg/core/identity"
	"go.eggybyte.com/egg/core/log"
)

// MockLogger is a mock logger for testing.
type MockLogger struct {
	t       *testing.T
	mu      sync.Mutex
	entries []LogEntry
}

// LogEntry represents a single log entry.
type LogEntry struct {
	Level   string
	Message string
	Fields  []any
	Error   error
}

// NewMockLogger creates a new mock logger.
func NewMockLogger(t *testing.T) *MockLogger {
	return &MockLogger{
		t:       t,
		entries: make([]LogEntry, 0),
	}
}

// With returns a new logger with the given fields.
func (m *MockLogger) With(kv ...any) log.Logger {
	return m // Simplified: just return self
}

// Debug logs a debug message.
func (m *MockLogger) Debug(msg string, kv ...any) {
	m.log("DEBUG", msg, nil, kv)
}

// Info logs an info message.
func (m *MockLogger) Info(msg string, kv ...any) {
	m.log("INFO", msg, nil, kv)
}

// Warn logs a warning message.
func (m *MockLogger) Warn(msg string, kv ...any) {
	m.log("WARN", msg, nil, kv)
}

// Error logs an error message.
func (m *MockLogger) Error(err error, msg string, kv ...any) {
	m.log("ERROR", msg, err, kv)
}

// log stores a log entry.
func (m *MockLogger) log(level, msg string, err error, kv []any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, LogEntry{
		Level:   level,
		Message: msg,
		Fields:  kv,
		Error:   err,
	})
}

// Entries returns all log entries.
func (m *MockLogger) Entries() []LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	entries := make([]LogEntry, len(m.entries))
	copy(entries, m.entries)
	return entries
}

// AssertLogged asserts that a message was logged.
func (m *MockLogger) AssertLogged(level, msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, entry := range m.entries {
		if entry.Level == level && entry.Message == msg {
			return
		}
	}
	m.t.Errorf("Expected log message not found: level=%s msg=%q", level, msg)
}

// Clear clears all log entries.
func (m *MockLogger) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = nil
}

// NewContextWithIdentity creates a context with identity for testing.
func NewContextWithIdentity(t *testing.T, user *identity.UserInfo) context.Context {
	t.Helper()
	ctx := context.Background()
	if user != nil {
		ctx = identity.WithUser(ctx, user)
	}
	return ctx
}

// NewContextWithMeta creates a context with request metadata for testing.
func NewContextWithMeta(t *testing.T, meta *identity.RequestMeta) context.Context {
	t.Helper()
	ctx := context.Background()
	if meta != nil {
		ctx = identity.WithMeta(ctx, meta)
	}
	return ctx
}

// AssertError asserts that an error has the expected code.
func AssertError(t *testing.T, err error, expectedCode errors.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("Expected error with code %s, got nil", expectedCode)
	}

	code := errors.CodeOf(err)
	if code != expectedCode {
		t.Errorf("Expected error code %s, got %s", expectedCode, code)
	}
}

// AssertNoError asserts that no error occurred.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

// CaptureLogger creates a logger that captures output to a buffer.
type CaptureLogger struct {
	buffer bytes.Buffer
	mu     sync.Mutex
}

// NewCaptureLogger creates a new capture logger.
func NewCaptureLogger() *CaptureLogger {
	return &CaptureLogger{}
}

// With returns a new logger with the given fields.
func (c *CaptureLogger) With(kv ...any) log.Logger {
	return c
}

// Debug logs a debug message.
func (c *CaptureLogger) Debug(msg string, kv ...any) {
	c.write("DEBUG", msg, nil, kv)
}

// Info logs an info message.
func (c *CaptureLogger) Info(msg string, kv ...any) {
	c.write("INFO", msg, nil, kv)
}

// Warn logs a warning message.
func (c *CaptureLogger) Warn(msg string, kv ...any) {
	c.write("WARN", msg, nil, kv)
}

// Error logs an error message.
func (c *CaptureLogger) Error(err error, msg string, kv ...any) {
	c.write("ERROR", msg, err, kv)
}

// write writes to the buffer.
func (c *CaptureLogger) write(level, msg string, err error, kv []any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buffer.WriteString(level)
	c.buffer.WriteString(": ")
	c.buffer.WriteString(msg)
	if err != nil {
		c.buffer.WriteString(" error=")
		c.buffer.WriteString(err.Error())
	}
	c.buffer.WriteString("\n")
}

// String returns the captured output.
func (c *CaptureLogger) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buffer.String()
}

// Clear clears the buffer.
func (c *CaptureLogger) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buffer.Reset()
}
