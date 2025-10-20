package log

import (
	"testing"
	"time"
)

func TestStr(t *testing.T) {
	kv := Str("key", "value")
	if kv == nil {
		t.Fatal("Str should return non-nil value")
	}

	// Verify it's a slice with two elements
	if slice, ok := kv.([]any); !ok {
		t.Fatal("Str should return []any")
	} else if len(slice) != 2 {
		t.Fatalf("Str should return slice with 2 elements, got %d", len(slice))
	} else if slice[0] != "key" || slice[1] != "value" {
		t.Fatalf("Str should return [\"key\", \"value\"], got %v", slice)
	}
}

func TestInt(t *testing.T) {
	kv := Int("count", 42)
	if kv == nil {
		t.Fatal("Int should return non-nil value")
	}

	// Verify it's a slice with two elements
	if slice, ok := kv.([]any); !ok {
		t.Fatal("Int should return []any")
	} else if len(slice) != 2 {
		t.Fatalf("Int should return slice with 2 elements, got %d", len(slice))
	} else if slice[0] != "count" || slice[1] != 42 {
		t.Fatalf("Int should return [\"count\", 42], got %v", slice)
	}
}

func TestDur(t *testing.T) {
	duration := 5 * time.Second
	kv := Dur("latency", duration)
	if kv == nil {
		t.Fatal("Dur should return non-nil value")
	}

	// Verify it's a slice with two elements
	if slice, ok := kv.([]any); !ok {
		t.Fatal("Dur should return []any")
	} else if len(slice) != 2 {
		t.Fatalf("Dur should return slice with 2 elements, got %d", len(slice))
	} else if slice[0] != "latency" || slice[1] != duration {
		t.Fatalf("Dur should return [\"latency\", %v], got %v", duration, slice)
	}
}

// MockLogger is a test implementation of the Logger interface
type MockLogger struct {
	Messages []string
	Fields   [][]any
}

func (m *MockLogger) With(kv ...any) Logger {
	return m
}

func (m *MockLogger) Debug(msg string, kv ...any) {
	m.Messages = append(m.Messages, msg)
	m.Fields = append(m.Fields, kv)
}

func (m *MockLogger) Info(msg string, kv ...any) {
	m.Messages = append(m.Messages, msg)
	m.Fields = append(m.Fields, kv)
}

func (m *MockLogger) Warn(msg string, kv ...any) {
	m.Messages = append(m.Messages, msg)
	m.Fields = append(m.Fields, kv)
}

func (m *MockLogger) Error(err error, msg string, kv ...any) {
	m.Messages = append(m.Messages, msg)
	m.Fields = append(m.Fields, kv)
}

func TestLoggerInterface(t *testing.T) {
	logger := &MockLogger{}

	// Test all methods
	logger.Debug("debug message", Str("key", "value"))
	logger.Info("info message", Int("count", 1))
	logger.Warn("warn message", Dur("latency", time.Second))
	logger.Error(nil, "error message")

	if len(logger.Messages) != 4 {
		t.Fatalf("Expected 4 messages, got %d", len(logger.Messages))
	}

	expectedMessages := []string{"debug message", "info message", "warn message", "error message"}
	for i, expected := range expectedMessages {
		if logger.Messages[i] != expected {
			t.Errorf("Expected message %q, got %q", expected, logger.Messages[i])
		}
	}
}
