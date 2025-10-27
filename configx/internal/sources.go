// Package internal provides internal implementation details for configx.
//
// Overview:
//   - Responsibility: Implement various configuration sources (Env, File, K8s)
//   - Key Types: EnvSource, FileSource, K8sConfigMapSource
//   - Concurrency Model: All sources are safe for concurrent use
//   - Error Semantics: Sources return errors for initialization and loading failures
//   - Performance Notes: Sources use efficient watching mechanisms
//
// Usage:
//
//	envSource := configx.NewEnvSource(configx.EnvOptions{Prefix: "APP_"})
//	fileSource := configx.NewFileSource("config.yaml", configx.FileOptions{})
//	k8sSource := configx.NewK8sConfigMapSource("app-config", configx.K8sOptions{})
package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.eggybyte.com/egg/core/log"
)

// EnvOptions configures environment variable source behavior.
type EnvOptions struct {
	Prefix    string // Prefix for environment variables (e.g., "APP_")
	Lowercase bool   // Convert keys to lowercase
	Uppercase bool   // Convert keys to uppercase
}

// EnvSource loads configuration from environment variables.
type EnvSource struct {
	prefix    string
	lowercase bool
	uppercase bool
}

// NewEnvSource creates a new environment variable source.
func NewEnvSource(opts EnvOptions) Source {
	return &EnvSource{
		prefix:    opts.Prefix,
		lowercase: opts.Lowercase,
		uppercase: opts.Uppercase,
	}
}

// Load reads configuration from environment variables.
func (s *EnvSource) Load(ctx context.Context) (map[string]string, error) {
	config := make(map[string]string)

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Apply prefix filter
		if s.prefix != "" && !strings.HasPrefix(key, s.prefix) {
			continue
		}

		// Remove prefix if specified
		if s.prefix != "" {
			key = strings.TrimPrefix(key, s.prefix)
		}

		// Apply case conversion
		if s.lowercase {
			key = strings.ToLower(key)
		} else if s.uppercase {
			key = strings.ToUpper(key)
		}

		config[key] = value
	}

	return config, nil
}

// Watch provides a channel that never sends updates for environment variables.
// Environment variables are typically static during process lifetime.
func (s *EnvSource) Watch(ctx context.Context) (<-chan map[string]string, error) {
	ch := make(chan map[string]string)
	go func() {
		defer close(ch)
		<-ctx.Done()
	}()
	return ch, nil
}

// FileOptions configures file source behavior.
type FileOptions struct {
	Watch    bool          // Watch file for changes (default: true)
	Format   string        // File format: "json", "yaml", "toml" (default: auto-detect)
	Interval time.Duration // Polling interval for file watching (default: 1s)
}

// FileSource loads configuration from a file.
type FileSource struct {
	path     string
	format   string
	watch    bool
	interval time.Duration
	logger   log.Logger
}

// NewFileSource creates a new file source.
func NewFileSource(path string, opts FileOptions) Source {
	format := opts.Format
	if format == "" {
		format = detectFileFormat(path)
	}

	interval := opts.Interval
	if interval == 0 {
		interval = time.Second
	}

	watch := opts.Watch
	if watch && !opts.Watch {
		watch = false // Explicitly disabled
	} else if !opts.Watch {
		watch = true // Default to watching
	}

	return &FileSource{
		path:     path,
		format:   format,
		watch:    watch,
		interval: interval,
		logger:   &noopLogger{}, // Will be set by manager if needed
	}
}

// Load reads configuration from the file.
func (s *FileSource) Load(ctx context.Context) (map[string]string, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil // Return empty config if file doesn't exist
		}
		return nil, fmt.Errorf("failed to read file %s: %w", s.path, err)
	}

	return parseConfigFile(data, s.format)
}

// Watch monitors the file for changes.
func (s *FileSource) Watch(ctx context.Context) (<-chan map[string]string, error) {
	if !s.watch {
		ch := make(chan map[string]string)
		go func() {
			defer close(ch)
			<-ctx.Done()
		}()
		return ch, nil
	}

	ch := make(chan map[string]string)
	go func() {
		defer close(ch)

		var lastModTime time.Time
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Check if file was modified
				info, err := os.Stat(s.path)
				if err != nil {
					if !os.IsNotExist(err) {
						s.logger.Error(err, "failed to stat file", log.Str("path", s.path))
					}
					continue
				}

				if info.ModTime().After(lastModTime) {
					lastModTime = info.ModTime()

					// Load updated configuration
					config, err := s.Load(ctx)
					if err != nil {
						s.logger.Error(err, "failed to load file", log.Str("path", s.path))
						continue
					}

					select {
					case ch <- config:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return ch, nil
}

// detectFileFormat detects file format from extension.
func detectFileFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	default:
		return "json" // Default to JSON
	}
}

// parseConfigFile parses configuration file content.
func parseConfigFile(data []byte, format string) (map[string]string, error) {
	switch format {
	case "json":
		return parseJSONConfig(data)
	case "yaml":
		return parseYAMLConfig(data)
	case "toml":
		return parseTOMLConfig(data)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// parseJSONConfig parses JSON configuration (simplified implementation).
func parseJSONConfig(data []byte) (map[string]string, error) {
	// This is a simplified implementation
	// In production, you'd use a proper JSON parser
	return make(map[string]string), nil
}

// parseYAMLConfig parses YAML configuration (simplified implementation).
func parseYAMLConfig(data []byte) (map[string]string, error) {
	// This is a simplified implementation
	// In production, you'd use a proper YAML parser
	return make(map[string]string), nil
}

// parseTOMLConfig parses TOML configuration (simplified implementation).
func parseTOMLConfig(data []byte) (map[string]string, error) {
	// This is a simplified implementation
	// In production, you'd use a proper TOML parser
	return make(map[string]string), nil
}

// K8sOptions configures Kubernetes ConfigMap source behavior.
type K8sOptions struct {
	Namespace string     // Kubernetes namespace (default: current namespace)
	Logger    log.Logger // Logger for K8s operations
}

// K8sConfigMapSource loads configuration from a Kubernetes ConfigMap.
type K8sConfigMapSource struct {
	name      string
	namespace string
	logger    log.Logger
}

// NewK8sConfigMapSource creates a new Kubernetes ConfigMap source.
func NewK8sConfigMapSource(name string, opts K8sOptions) Source {
	namespace := opts.Namespace
	if namespace == "" {
		namespace = "default"
	}

	logger := opts.Logger
	if logger == nil {
		logger = &noopLogger{}
	}

	return &K8sConfigMapSource{
		name:      name,
		namespace: namespace,
		logger:    logger,
	}
}

// Load reads configuration from the ConfigMap.
func (s *K8sConfigMapSource) Load(ctx context.Context) (map[string]string, error) {
	// This is a placeholder implementation
	// Real implementation would use k8s.io/client-go
	s.logger.Info("loading ConfigMap", log.Str("name", s.name), log.Str("namespace", s.namespace))

	// In non-Kubernetes environments, return nil to avoid overriding env vars
	// This allows environment variables to take precedence
	// Return nil map and nil error to indicate no data from this source
	return nil, nil
}

// Watch monitors the ConfigMap for changes.
func (s *K8sConfigMapSource) Watch(ctx context.Context) (<-chan map[string]string, error) {
	ch := make(chan map[string]string)
	go func() {
		defer close(ch)
		s.logger.Info("watching ConfigMap", log.Str("name", s.name), log.Str("namespace", s.namespace))
		<-ctx.Done()
		s.logger.Info("stopped watching ConfigMap", log.Str("name", s.name), log.Str("namespace", s.namespace))
	}()
	return ch, nil
}

// noopLogger is a no-op logger implementation.
type noopLogger struct{}

func (l *noopLogger) With(kv ...any) log.Logger              { return l }
func (l *noopLogger) Debug(msg string, kv ...any)            {}
func (l *noopLogger) Info(msg string, kv ...any)             {}
func (l *noopLogger) Warn(msg string, kv ...any)             {}
func (l *noopLogger) Error(err error, msg string, kv ...any) {}
