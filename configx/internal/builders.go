// Package internal provides internal implementation details for configx.
//
// Overview:
//   - Responsibility: Provide easy-to-use functions for common configuration patterns
//   - Key Types: Builder functions for multi-source configurations
//   - Concurrency Model: All functions are safe for concurrent use
//   - Error Semantics: Functions return errors for invalid configurations
//   - Performance Notes: Optimized for common deployment patterns
//
// Usage:
//
//	sources, err := configx.BuildSources(ctx, logger)
//	manager, err := configx.NewManager(ctx, configx.Options{
//	  Logger: logger,
//	  Sources: sources,
//	})
package internal

import (
	"context"
	"os"
	"strings"

	"github.com/eggybyte-technology/egg/core/log"
)

// BuildSources builds configuration sources based on environment variables.
// This function implements the multi-ConfigMap pattern described in the guide.
//
// Environment variables recognized:
//   - APP_CONFIGMAP_NAME: Application-level dynamic configuration
//   - CACHE_CONFIGMAP_NAME: Cache-related configuration
//   - ACL_CONFIGMAP_NAME: Access control configuration
//   - Any *_CONFIGMAP_NAME: Additional ConfigMap sources
func BuildSources(ctx context.Context, logger log.Logger) ([]Source, error) {
	var sources []Source

	// Always start with environment variables as baseline
	envSource := NewEnvSource(EnvOptions{Prefix: ""})
	sources = append(sources, envSource)

	// Collect all ConfigMap names from environment variables
	configMapNames := collectConfigMapNames()

	// Add ConfigMap sources
	for _, name := range configMapNames {
		if name == "" {
			continue
		}

		k8sSource := NewK8sConfigMapSource(name, K8sOptions{
			Namespace: os.Getenv("NAMESPACE"),
			Logger:    logger,
		})
		sources = append(sources, k8sSource)
	}

	return sources, nil
}

// collectConfigMapNames collects ConfigMap names from environment variables.
func collectConfigMapNames() []string {
	var names []string

	// Check for explicit ConfigMap names
	explicitNames := []string{
		os.Getenv("APP_CONFIGMAP_NAME"),
		os.Getenv("CACHE_CONFIGMAP_NAME"),
		os.Getenv("ACL_CONFIGMAP_NAME"),
	}

	for _, name := range explicitNames {
		if name != "" {
			names = append(names, name)
		}
	}

	// Check for any *_CONFIGMAP_NAME pattern
	for _, env := range os.Environ() {
		if strings.HasSuffix(env, "_CONFIGMAP_NAME=") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 && parts[1] != "" {
				names = append(names, parts[1])
			}
		}
	}

	return names
}

// BuildEnvOnlySources builds sources for environment-only mode.
// This is useful for local development or simple deployments.
func BuildEnvOnlySources() []Source {
	return []Source{
		NewEnvSource(EnvOptions{Prefix: ""}),
	}
}

// BuildFileSources builds sources for file-based configuration.
// This is useful for containerized applications with mounted config files.
func BuildFileSources(configPaths []string, opts FileOptions) []Source {
	var sources []Source

	// Add environment variables as baseline
	sources = append(sources, NewEnvSource(EnvOptions{Prefix: ""}))

	// Add file sources
	for _, path := range configPaths {
		sources = append(sources, NewFileSource(path, opts))
	}

	return sources
}

// BuildHybridSources builds sources for hybrid configuration (Env + File + K8s).
// This combines multiple configuration sources for maximum flexibility.
func BuildHybridSources(ctx context.Context, logger log.Logger, configPaths []string, fileOpts FileOptions) ([]Source, error) {
	var sources []Source

	// Environment variables (baseline)
	sources = append(sources, NewEnvSource(EnvOptions{Prefix: ""}))

	// File sources (if any)
	for _, path := range configPaths {
		sources = append(sources, NewFileSource(path, fileOpts))
	}

	// Kubernetes ConfigMap sources (if any)
	configMapNames := collectConfigMapNames()
	for _, name := range configMapNames {
		if name == "" {
			continue
		}

		k8sSource := NewK8sConfigMapSource(name, K8sOptions{
			Namespace: os.Getenv("NAMESPACE"),
			Logger:    logger,
		})
		sources = append(sources, k8sSource)
	}

	return sources, nil
}

// Note: DefaultManager and QuickBind are now exposed at the package level (configx.go)
