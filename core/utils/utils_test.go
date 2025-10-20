package utils

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{1, 1, 1},
		{-1, 1, -1},
		{0, 0, 0},
	}

	for _, tt := range tests {
		result := Min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("Min(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 2},
		{2, 1, 2},
		{1, 1, 1},
		{-1, 1, 1},
		{0, 0, 0},
	}

	for _, tt := range tests {
		result := Max(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("Max(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestContains(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	tests := []struct {
		item     string
		expected bool
	}{
		{"apple", true},
		{"banana", true},
		{"cherry", true},
		{"orange", false},
		{"", false},
	}

	for _, tt := range tests {
		result := Contains(slice, tt.item)
		if result != tt.expected {
			t.Errorf("Contains(%v, %q) = %v, expected %v", slice, tt.item, result, tt.expected)
		}
	}
}

func TestUnique(t *testing.T) {
	tests := []struct {
		input    []string
		expected []string
	}{
		{
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			input:    []string{},
			expected: []string{},
		},
		{
			input:    []string{"a"},
			expected: []string{"a"},
		},
		{
			input:    []string{"a", "a", "a"},
			expected: []string{"a"},
		},
	}

	for _, tt := range tests {
		result := Unique(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("Unique(%v) length = %d, expected %d", tt.input, len(result), len(tt.expected))
			continue
		}

		for i, expected := range tt.expected {
			if result[i] != expected {
				t.Errorf("Unique(%v)[%d] = %q, expected %q", tt.input, i, result[i], expected)
			}
		}
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"1s", time.Second, false},
		{"500ms", 500 * time.Millisecond, false},
		{"1m30s", 90 * time.Second, false},
		{"1h", time.Hour, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		result, err := ParseDuration(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("ParseDuration(%q) should return error", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseDuration(%q) returned error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("ParseDuration(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		}
	}
}

func TestRetry(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()

	// Test successful operation
	attempts := 0
	err := Retry(ctx, config, func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("Retry should succeed, got error: %v", err)
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}

	// Test failing operation
	attempts = 0
	err = Retry(ctx, config, func() error {
		attempts++
		return errors.New("test error")
	})

	if err == nil {
		t.Error("Retry should fail after max attempts")
	}

	if attempts != config.MaxAttempts {
		t.Errorf("Expected %d attempts, got %d", config.MaxAttempts, attempts)
	}

	// Test context cancellation
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately

	attempts = 0
	err = Retry(cancelCtx, config, func() error {
		attempts++
		return errors.New("test error")
	})

	if err == nil {
		t.Error("Retry should fail due to context cancellation")
	}

	if attempts != 0 {
		t.Errorf("Expected 0 attempts due to cancellation, got %d", attempts)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxAttempts <= 0 {
		t.Error("MaxAttempts should be positive")
	}

	if config.BaseDelay <= 0 {
		t.Error("BaseDelay should be positive")
	}

	if config.MaxDelay <= 0 {
		t.Error("MaxDelay should be positive")
	}

	if config.Multiplier <= 1.0 {
		t.Error("Multiplier should be greater than 1.0")
	}

	if config.BaseDelay >= config.MaxDelay {
		t.Error("BaseDelay should be less than MaxDelay")
	}
}
