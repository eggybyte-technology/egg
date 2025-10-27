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
	"time"

	"go.eggybyte.com/egg/configx/internal"
	"go.eggybyte.com/egg/core/log"
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
	LogLevel       string `env:"LOG_LEVEL" default:"info"` // Log level: debug, info, warn, error

	// Single port strategy: HTTP/Connect/gRPC-Web on one port
	HTTPPort    string `env:"HTTP_PORT" default:":8080"`
	HealthPort  string `env:"HEALTH_PORT" default:":8081"`
	MetricsPort string `env:"METRICS_PORT" default:":9091"`

	// Observability and dynamic configuration
	OTLPEndpoint   string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" default:""`
	ConfigMapName  string `env:"APP_CONFIGMAP_NAME" default:""` // Empty means Env-only mode
	DebounceMillis int    `env:"CONFIG_DEBOUNCE_MS" default:"200"`

	// Database configuration (optional, embedded)
	Database DatabaseConfig
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	Driver      string        `env:"DB_DRIVER" default:"mysql"`
	DSN         string        `env:"DB_DSN" default:""`
	MaxIdle     int           `env:"DB_MAX_IDLE" default:"10"`
	MaxOpen     int           `env:"DB_MAX_OPEN" default:"100"`
	MaxLifetime time.Duration `env:"DB_MAX_LIFETIME" default:"1h"`
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

// manager wraps the internal manager implementation.
type manager struct {
	impl *internal.ManagerImpl
}

// NewManager creates a new configuration manager.
//
// Parameters:
//   - ctx: context for manager initialization
//   - opts: manager configuration options
//
// Returns:
//   - Manager: initialized manager instance
//   - error: initialization error if any
//
// Concurrency:
//   - Safe to call from multiple goroutines
func NewManager(ctx context.Context, opts Options) (Manager, error) {
	if opts.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	if len(opts.Sources) == 0 {
		return nil, fmt.Errorf("at least one source is required")
	}

	// Convert sources to internal type
	internalSources := make([]internal.Source, len(opts.Sources))
	for i, src := range opts.Sources {
		internalSources[i] = src
	}

	// Set default debounce duration
	debounce := opts.Debounce
	if debounce == 0 {
		debounce = 200 * time.Millisecond
	}

	impl, err := internal.NewManager(opts.Logger, internalSources, debounce)
	if err != nil {
		return nil, err
	}

	// Initialize the manager
	if err := impl.Initialize(ctx); err != nil {
		return nil, err
	}

	return &manager{impl: impl}, nil
}

// Snapshot returns a copy of the current configuration.
func (m *manager) Snapshot() map[string]string {
	return m.impl.Snapshot()
}

// Value returns the value for a key and whether it exists.
func (m *manager) Value(key string) (string, bool) {
	return m.impl.Value(key)
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

	return m.impl.Bind(target, internal.BindConfig{
		OnUpdate: cfg.onUpdate,
	})
}

// OnUpdate subscribes to configuration update events.
func (m *manager) OnUpdate(fn func(snapshot map[string]string)) func() {
	return m.impl.OnUpdate(fn)
}

// --- Public wrappers for source constructors (delegating to internal) ---

// NewEnvSource creates an environment variable configuration source.
func NewEnvSource(opts EnvOptions) Source {
	return internal.NewEnvSource(internal.EnvOptions{
		Prefix:    opts.Prefix,
		Lowercase: opts.Lowercase,
		Uppercase: opts.Uppercase,
	})
}

// NewFileSource creates a file-based configuration source.
func NewFileSource(path string, opts FileOptions) Source {
	return internal.NewFileSource(path, internal.FileOptions{
		Watch:    opts.Watch,
		Format:   opts.Format,
		Interval: opts.Interval,
	})
}

// NewK8sConfigMapSource creates a Kubernetes ConfigMap configuration source.
func NewK8sConfigMapSource(name string, opts K8sOptions) Source {
	return internal.NewK8sConfigMapSource(name, internal.K8sOptions{
		Namespace: opts.Namespace,
		Logger:    opts.Logger,
	})
}

// DefaultManager creates a configuration manager with default sources (Env + optional K8s).
func DefaultManager(ctx context.Context, logger log.Logger) (Manager, error) {
	internalSources, err := internal.BuildSources(ctx, logger)
	if err != nil {
		return nil, err
	}
	// Convert internal.Source to configx.Source
	sources := make([]Source, len(internalSources))
	for i, s := range internalSources {
		sources[i] = s
	}
	return NewManager(ctx, Options{
		Logger:   logger,
		Sources:  sources,
		Debounce: 200 * time.Millisecond,
	})
}

// EnvOptions configures environment variable source behavior.
type EnvOptions struct {
	Prefix    string
	Lowercase bool
	Uppercase bool
}

// FileOptions configures file source behavior.
type FileOptions struct {
	Watch    bool
	Format   string
	Interval time.Duration
}

// K8sOptions configures Kubernetes ConfigMap source behavior.
type K8sOptions struct {
	Namespace string
	Logger    log.Logger
}
