// Package internal provides internal implementation for the configx package.
package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.eggybyte.com/egg/core/log"
)

// ManagerImpl implements the manager for configuration sources.
type ManagerImpl struct {
	logger     log.Logger
	sources    []Source
	debounce   time.Duration
	snapshot   map[string]string
	mu         sync.RWMutex
	updateSubs map[int]func(map[string]string)
	subsMu     sync.RWMutex
	nextSubID  int
}

// BindConfig holds bind configuration options.
type BindConfig struct {
	OnUpdate func()
}

// NewManager creates a new configuration manager.
func NewManager(logger log.Logger, sources []Source, debounce time.Duration) (*ManagerImpl, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	if len(sources) == 0 {
		return nil, fmt.Errorf("at least one source is required")
	}

	// Set default debounce duration
	if debounce == 0 {
		debounce = 200 * time.Millisecond
	}

	m := &ManagerImpl{
		logger:     logger,
		sources:    sources,
		debounce:   debounce,
		snapshot:   make(map[string]string),
		updateSubs: make(map[int]func(map[string]string)),
	}

	return m, nil
}

// Initialize loads initial configuration and starts watching.
func (m *ManagerImpl) Initialize(ctx context.Context) error {
	// Load initial configuration
	if err := m.loadInitial(ctx); err != nil {
		return fmt.Errorf("failed to load initial configuration: %w", err)
	}

	// Start watching for updates
	if err := m.startWatching(ctx); err != nil {
		return fmt.Errorf("failed to start watching: %w", err)
	}

	return nil
}

// loadInitial loads configuration from all sources and merges them.
func (m *ManagerImpl) loadInitial(ctx context.Context) error {
	merged := make(map[string]string)

	for i, source := range m.sources {
		snapshot, err := source.Load(ctx)
		if err != nil {
			return fmt.Errorf("source %d load failed: %w", i, err)
		}

		// Merge with later sources taking precedence
		// Only set values that are non-empty to avoid overriding env vars with empty ConfigMap values
		for k, v := range snapshot {
			if v != "" {
				merged[k] = v
			}
		}
	}

	m.mu.Lock()
	m.snapshot = merged
	m.mu.Unlock()

	m.logger.Info("configuration loaded", "keys", len(merged))
	return nil
}

// startWatching starts watching all sources for updates.
func (m *ManagerImpl) startWatching(ctx context.Context) error {
	for i, source := range m.sources {
		updateChan, err := source.Watch(ctx)
		if err != nil {
			return fmt.Errorf("source %d watch failed: %w", i, err)
		}

		go m.watchSource(ctx, i, updateChan)
	}

	return nil
}

// watchSource watches a single source for updates.
func (m *ManagerImpl) watchSource(ctx context.Context, sourceIndex int, updateChan <-chan map[string]string) {
	var debounceTimer *time.Timer
	var pendingUpdate map[string]string

	for {
		select {
		case <-ctx.Done():
			return
		case snapshot, ok := <-updateChan:
			if !ok {
				return // Channel closed
			}

			// Debounce updates
			if debounceTimer != nil {
				debounceTimer.Stop()
			}

			pendingUpdate = snapshot
			debounceTimer = time.AfterFunc(m.debounce, func() {
				m.applyUpdate(sourceIndex, pendingUpdate)
			})
		}
	}
}

// applyUpdate applies a configuration update from a specific source.
func (m *ManagerImpl) applyUpdate(sourceIndex int, update map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Re-merge all sources with the updated one
	merged := make(map[string]string)

	for i, source := range m.sources {
		var snapshot map[string]string
		if i == sourceIndex {
			snapshot = update
		} else {
			// For other sources, we'd need to cache their last snapshot
			// For simplicity, we'll reload all sources (this could be optimized)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			snap, err := source.Load(ctx)
			cancel()
			if err != nil {
				m.logger.Error(err, "failed to reload source for update", log.Int("source", i))
				continue
			}
			snapshot = snap
		}

		for k, v := range snapshot {
			if v != "" {
				merged[k] = v
			}
		}
	}

	m.snapshot = merged
	m.logger.Info("configuration updated", log.Int("keys", len(merged)))

	// Notify subscribers
	m.notifySubscribers(merged)
}

// notifySubscribers notifies all subscribers of configuration updates.
func (m *ManagerImpl) notifySubscribers(snapshot map[string]string) {
	m.subsMu.RLock()
	subs := make(map[int]func(map[string]string))
	for k, v := range m.updateSubs {
		subs[k] = v
	}
	m.subsMu.RUnlock()

	for _, sub := range subs {
		go sub(snapshot)
	}
}

// Snapshot returns a copy of the current configuration.
func (m *ManagerImpl) Snapshot() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := make(map[string]string, len(m.snapshot))
	for k, v := range m.snapshot {
		snapshot[k] = v
	}
	return snapshot
}

// Value returns the value for a key and whether it exists.
func (m *ManagerImpl) Value(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, exists := m.snapshot[key]
	return value, exists
}

// Bind decodes the configuration into a struct.
func (m *ManagerImpl) Bind(target any, cfg BindConfig) error {
	if target == nil {
		return fmt.Errorf("target cannot be nil")
	}

	snapshot := m.Snapshot()
	return BindToStruct(snapshot, target, cfg.OnUpdate)
}

// OnUpdate subscribes to configuration update events.
func (m *ManagerImpl) OnUpdate(fn func(snapshot map[string]string)) func() {
	m.subsMu.Lock()
	defer m.subsMu.Unlock()

	// Assign unique ID to this subscription
	subID := m.nextSubID
	m.nextSubID++
	m.updateSubs[subID] = fn

	// Return unsubscribe function
	return func() {
		m.subsMu.Lock()
		defer m.subsMu.Unlock()
		delete(m.updateSubs, subID)
	}
}
