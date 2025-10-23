// Package configx provides unified configuration management with hot reloading.
//
// Overview:
//   - Responsibility: Manage configuration from multiple sources with hot updates
//   - Key Types: Source interface, Manager interface, Options for configuration
//   - Concurrency Model: Manager is safe for concurrent use, sources must be thread-safe
//   - Error Semantics: Functions return errors for initialization and binding failures
//   - Performance Notes: Supports debouncing and efficient configuration merging
//
// Usage:
//
//	sources := []configx.Source{
//	  configx.NewEnvSource(configx.EnvOptions{}),
//	  configx.NewK8sConfigMapSource("app-config", configx.K8sOptions{}),
//	}
//	manager, err := configx.NewManager(ctx, configx.Options{
//	  Logger: logger,
//	  Sources: sources,
//	  Debounce: 200 * time.Millisecond,
//	})
//	var cfg AppConfig
//	err = manager.Bind(&cfg)
package configx

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/eggybyte-technology/egg/core/log"
)

// Source describes a configuration source that can load and watch for updates.
// Implementations must be thread-safe and honor context cancellation.
type Source interface {
	// Load reads the current configuration snapshot for initial merge.
	// Returns a map of key-value pairs.
	Load(ctx context.Context) (map[string]string, error)

	// Watch starts monitoring for updates and publishes snapshots via the returned channel.
	// The channel should be closed when the context is cancelled to avoid goroutine leaks.
	Watch(ctx context.Context) (<-chan map[string]string, error)
}

// Manager manages multiple configuration sources and provides unified access.
// The manager merges configurations with later sources taking precedence.
type Manager interface {
	// Snapshot returns a copy of the current merged configuration.
	Snapshot() map[string]string

	// Value returns the value for a key and whether it exists.
	Value(key string) (string, bool)

	// Bind decodes the configuration into a struct with env tags and default values.
	// Supports hot reloading via callback when configuration changes.
	Bind(target any, opts ...BindOption) error

	// OnUpdate subscribes to configuration update events.
	// Returns an unsubscribe function.
	OnUpdate(fn func(snapshot map[string]string)) (unsubscribe func())
}

// Options holds configuration for the manager.
type Options struct {
	Logger   log.Logger    // Logger for configuration operations
	Sources  []Source      // Configuration sources (later sources override earlier ones)
	Debounce time.Duration // Debounce duration for updates (default: 200ms)
}

// BindOption configures binding behavior.
type BindOption interface {
	apply(*bindConfig)
}

type bindConfig struct {
	onUpdate func()
}

type bindOptionFunc func(*bindConfig)

func (f bindOptionFunc) apply(cfg *bindConfig) {
	f(cfg)
}

// WithUpdateCallback sets a callback to be invoked when configuration changes.
func WithUpdateCallback(fn func()) BindOption {
	return bindOptionFunc(func(cfg *bindConfig) {
		cfg.onUpdate = fn
	})
}

// BaseConfig provides common configuration fields for all services.
// All fields are read from environment variables only.
type BaseConfig struct {
	ServiceName    string `env:"SERVICE_NAME" default:"app"`
	ServiceVersion string `env:"SERVICE_VERSION" default:"0.0.0"`
	Env            string `env:"ENV" default:"dev"`

	// Single port strategy: HTTP/Connect/gRPC-Web on one port
	HTTPPort    string `env:"HTTP_PORT" default:":8080"`
	HealthPort  string `env:"HEALTH_PORT" default:":8081"`
	MetricsPort string `env:"METRICS_PORT" default:":9091"`

	// Observability and dynamic configuration
	OTLPEndpoint   string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" default:""`
	ConfigMapName  string `env:"APP_CONFIGMAP_NAME" default:""` // Empty means Env-only mode
	DebounceMillis int    `env:"CONFIG_DEBOUNCE_MS" default:"200"`
}

// GetHTTPPort returns the HTTP server port.
func (c *BaseConfig) GetHTTPPort() string {
	return c.HTTPPort
}

// GetHealthPort returns the health check server port.
func (c *BaseConfig) GetHealthPort() string {
	return c.HealthPort
}

// GetMetricsPort returns the metrics server port.
func (c *BaseConfig) GetMetricsPort() string {
	return c.MetricsPort
}

// manager implements the Manager interface.
type manager struct {
	logger     log.Logger
	sources    []Source
	debounce   time.Duration
	snapshot   map[string]string
	mu         sync.RWMutex
	updateSubs map[int]func(map[string]string)
	subsMu     sync.RWMutex
	nextSubID  int
}

// NewManager creates a new configuration manager.
func NewManager(ctx context.Context, opts Options) (Manager, error) {
	if opts.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	if len(opts.Sources) == 0 {
		return nil, fmt.Errorf("at least one source is required")
	}

	// Set default debounce duration
	debounce := opts.Debounce
	if debounce == 0 {
		debounce = 200 * time.Millisecond
	}

	m := &manager{
		logger:     opts.Logger,
		sources:    opts.Sources,
		debounce:   debounce,
		snapshot:   make(map[string]string),
		updateSubs: make(map[int]func(map[string]string)),
	}

	// Load initial configuration
	if err := m.loadInitial(ctx); err != nil {
		return nil, fmt.Errorf("failed to load initial configuration: %w", err)
	}

	// Start watching for updates
	if err := m.startWatching(ctx); err != nil {
		return nil, fmt.Errorf("failed to start watching: %w", err)
	}

	return m, nil
}

// loadInitial loads configuration from all sources and merges them.
func (m *manager) loadInitial(ctx context.Context) error {
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
func (m *manager) startWatching(ctx context.Context) error {
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
func (m *manager) watchSource(ctx context.Context, sourceIndex int, updateChan <-chan map[string]string) {
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
func (m *manager) applyUpdate(sourceIndex int, update map[string]string) {
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
func (m *manager) notifySubscribers(snapshot map[string]string) {
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
func (m *manager) Snapshot() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := make(map[string]string, len(m.snapshot))
	for k, v := range m.snapshot {
		snapshot[k] = v
	}
	return snapshot
}

// Value returns the value for a key and whether it exists.
func (m *manager) Value(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, exists := m.snapshot[key]
	return value, exists
}

// Bind decodes the configuration into a struct.
func (m *manager) Bind(target any, opts ...BindOption) error {
	if target == nil {
		return fmt.Errorf("target cannot be nil")
	}

	var cfg bindConfig
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	snapshot := m.Snapshot()
	return bindToStruct(snapshot, target, cfg.onUpdate)
}

// OnUpdate subscribes to configuration update events.
func (m *manager) OnUpdate(fn func(snapshot map[string]string)) func() {
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

// bindToStruct binds configuration values to struct fields using env tags.
func bindToStruct(snapshot map[string]string, target any, onUpdate func()) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr || targetValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}

	return bindStructFields(snapshot, targetValue.Elem())
}

// bindStructFields recursively binds configuration values to struct fields.
func bindStructFields(snapshot map[string]string, structValue reflect.Value) error {
	structType := structValue.Type()

	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Field(i)
		fieldType := structType.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Handle nested structs (embedded or regular)
		if field.Kind() == reflect.Struct {
			if err := bindStructFields(snapshot, field); err != nil {
				return fmt.Errorf("failed to bind nested struct %s: %w", fieldType.Name, err)
			}
			continue
		}

		// Get env tag
		envTag := fieldType.Tag.Get("env")
		if envTag == "" {
			continue
		}

		// Get default value
		defaultValue := fieldType.Tag.Get("default")

		// Get value from snapshot or use default
		value, exists := snapshot[envTag]
		if !exists {
			value = defaultValue
		}

		// Set field value
		if err := setFieldValue(field, value); err != nil {
			return fmt.Errorf("failed to set field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

// setFieldValue sets a field value from a string.
func setFieldValue(field reflect.Value, value string) error {
	if value == "" {
		return nil // Keep zero value
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			duration, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			field.SetInt(int64(duration))
		} else {
			intValue, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			field.SetInt(intValue)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatValue)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}
