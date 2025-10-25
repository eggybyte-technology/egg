// Package internal contains the runtime implementation.
package internal

import (
	"context"
	"sync"
)

// HealthChecker defines the interface for health checks.
// Implementations should perform quick checks and honor context deadlines.
type HealthChecker interface {
	// Name returns the name of the health check.
	Name() string
	// Check performs the health check and returns an error if unhealthy.
	Check(ctx context.Context) error
}

var (
	healthCheckers   []HealthChecker
	healthCheckersMu sync.RWMutex
)

// RegisterHealthChecker registers a global health checker.
func RegisterHealthChecker(checker HealthChecker) {
	healthCheckersMu.Lock()
	defer healthCheckersMu.Unlock()
	healthCheckers = append(healthCheckers, checker)
}

// CheckHealth runs all registered health checkers.
// Returns nil if all checks pass, otherwise returns the first error.
func CheckHealth(ctx context.Context) error {
	healthCheckersMu.RLock()
	checkers := make([]HealthChecker, len(healthCheckers))
	copy(checkers, healthCheckers)
	healthCheckersMu.RUnlock()

	for _, checker := range checkers {
		if err := checker.Check(ctx); err != nil {
			return err
		}
	}

	return nil
}

// ClearHealthCheckers clears all registered health checkers (intended for testing).
func ClearHealthCheckers() {
	healthCheckersMu.Lock()
	defer healthCheckersMu.Unlock()
	healthCheckers = nil
}

