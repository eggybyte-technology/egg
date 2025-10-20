// Package utils provides minimal utility functions for common operations.
//
// Overview:
//   - Responsibility: Provide lightweight utility functions for common patterns
//   - Key Types: Retry configuration, time helpers, slice utilities
//   - Concurrency Model: All functions are safe for concurrent use
//   - Error Semantics: Functions return errors for failure cases
//   - Performance Notes: Minimal allocations, designed for high-throughput scenarios
//
// Usage:
//
//	err := utils.Retry(ctx, 3, func() error { return doSomething() })
//	duration := utils.ParseDuration("1s")
package utils

import (
	"context"
	"fmt"
	"time"
)

// RetryConfig holds configuration for retry operations.
type RetryConfig struct {
	MaxAttempts int           // Maximum number of attempts
	BaseDelay   time.Duration // Base delay between attempts
	MaxDelay    time.Duration // Maximum delay between attempts
	Multiplier  float64       // Delay multiplier for exponential backoff
}

// DefaultRetryConfig returns a sensible default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		Multiplier:  2.0,
	}
}

// Retry executes a function with retry logic using exponential backoff.
// The function will be retried up to config.MaxAttempts times.
// Returns the last error if all attempts fail.
func Retry(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error
	delay := config.BaseDelay

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := fn(); err != nil {
			lastErr = err
			if attempt == config.MaxAttempts-1 {
				break // Last attempt, don't sleep
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}

			delay = time.Duration(float64(delay) * config.Multiplier)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		} else {
			return nil // Success
		}
	}

	return fmt.Errorf("retry failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// ParseDuration parses a duration string with common abbreviations.
// Supports formats like "1s", "500ms", "1m30s", etc.
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// Min returns the minimum of two integers.
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max returns the maximum of two integers.
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Contains checks if a slice contains a specific string.
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Unique removes duplicate strings from a slice.
func Unique(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}
