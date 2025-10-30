// Package internal provides internal implementation for the configx package.
package internal

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
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

		// Log each source's configuration at DEBUG level for debugging
		m.logSourceConfiguration(i, source, snapshot)

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

	// Log merged configuration details at DEBUG level
	m.logConfigurationDetails(merged)

	m.logger.Info("configuration loaded", log.Int("keys", len(merged)))
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

// logConfigurationDetails logs merged configuration details at DEBUG level
// with sensitive data masking. This helps debug configuration issues without exposing secrets.
//
// Logs:
//   - DEBUG level: All configuration variables (one per line) with masked sensitive values
func (m *ManagerImpl) logConfigurationDetails(config map[string]string) {
	if len(config) == 0 {
		m.logger.Debug("merged configuration", log.Str("status", "empty"))
		return
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(config))
	for k := range config {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Log all configuration variables at DEBUG level
	m.logger.Debug("merged configuration loaded",
		log.Int("total_keys", len(config)))

	for _, key := range keys {
		value := config[key]
		maskedValue := maskSensitiveValue(key, value)
		m.logger.Debug("configuration variable",
			log.Str("key", key),
			log.Str("value", maskedValue))
	}
}

// logSourceConfiguration logs configuration loaded from a specific source at DEBUG level.
// This helps debug which source provided which configuration values.
// For EnvSource, it also logs all environment variables (before filtering).
func (m *ManagerImpl) logSourceConfiguration(sourceIndex int, source Source, snapshot map[string]string) {
	// Get source type name for logging
	sourceType := m.getSourceTypeName(source)

	// Special handling for EnvSource: log all environment variables
	if sourceType == "EnvSource" {
		m.logAllEnvironmentVariables(sourceIndex)
	}

	if len(snapshot) == 0 {
		m.logger.Debug("source configuration loaded",
			log.Int("source_index", sourceIndex),
			log.Str("source_type", sourceType),
			log.Str("status", "empty"))
		return
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(snapshot))
	for k := range snapshot {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Log source summary
	m.logger.Debug("source configuration loaded",
		log.Int("source_index", sourceIndex),
		log.Str("source_type", sourceType),
		log.Int("key_count", len(snapshot)))

	// Log each configuration variable from this source
	for _, key := range keys {
		value := snapshot[key]
		maskedValue := maskSensitiveValue(key, value)
		m.logger.Debug("source configuration variable",
			log.Int("source_index", sourceIndex),
			log.Str("source_type", sourceType),
			log.Str("key", key),
			log.Str("value", maskedValue))
	}
}

// logAllEnvironmentVariables logs all environment variables at DEBUG level.
// This provides complete visibility into the runtime environment for debugging.
func (m *ManagerImpl) logAllEnvironmentVariables(sourceIndex int) {
	allEnvs := os.Environ()
	if len(allEnvs) == 0 {
		m.logger.Debug("environment variables", log.Int("source_index", sourceIndex), log.Str("status", "empty"))
		return
	}

	// Sort for consistent output
	sort.Strings(allEnvs)

	// Log summary
	m.logger.Debug("environment variables loaded",
		log.Int("source_index", sourceIndex),
		log.Int("total_count", len(allEnvs)))

	// Log each environment variable
	for _, env := range allEnvs {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]
		maskedValue := maskSensitiveValue(key, value)

		m.logger.Debug("environment variable",
			log.Int("source_index", sourceIndex),
			log.Str("key", key),
			log.Str("value", maskedValue))
	}
}

// getSourceTypeName returns a human-readable type name for the source.
func (m *ManagerImpl) getSourceTypeName(source Source) string {
	typeName := fmt.Sprintf("%T", source)
	// Remove package prefixes for cleaner output
	if idx := strings.LastIndex(typeName, "."); idx >= 0 {
		typeName = typeName[idx+1:]
	}
	return typeName
}

// maskSensitiveValue masks sensitive configuration values.
// Sensitive keys include: password, passwd, secret, key, token, dsn, auth, credential.
func maskSensitiveValue(key, value string) string {
	if value == "" {
		return "(empty)"
	}

	keyLower := strings.ToLower(key)

	// Check if key contains sensitive keywords
	sensitiveKeywords := []string{
		"password", "passwd", "pass",
		"secret", "key", "token",
		"dsn", "auth", "credential",
		"api_key", "apikey", "apisecret",
		"private", "private_key",
	}

	for _, keyword := range sensitiveKeywords {
		if strings.Contains(keyLower, keyword) {
			// Mask the value, showing only first 4 chars and last 4 chars if long enough
			if len(value) <= 8 {
				return "***"
			}
			return value[:4] + "***" + value[len(value)-4:]
		}
	}

	// For DSN-like values (containing @), mask the password part
	if strings.Contains(value, "@") && (strings.Contains(keyLower, "dsn") || strings.Contains(keyLower, "uri") || strings.Contains(keyLower, "url")) {
		// Format: user:password@host:port/database
		// Mask as: user:***@host:port/database
		parts := strings.SplitN(value, "@", 2)
		if len(parts) == 2 {
			userPass := parts[0]
			rest := parts[1]
			if strings.Contains(userPass, ":") {
				userParts := strings.SplitN(userPass, ":", 2)
				if len(userParts) == 2 {
					return userParts[0] + ":***@" + rest
				}
			}
			return "***@" + rest
		}
	}

	return value
}
