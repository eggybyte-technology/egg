// Package k8sx provides Kubernetes ConfigMap watching and service discovery.
//
// Overview:
//   - Responsibility: Watch ConfigMaps for configuration updates and resolve service endpoints
//   - Key Types: WatchOptions for configuration, ServiceKind for service types
//   - Concurrency Model: All functions are safe for concurrent use
//   - Error Semantics: Functions return errors for failure cases
//   - Performance Notes: Uses Kubernetes informers for efficient resource watching
//
// Usage:
//
//	err := k8sx.WatchConfigMap(ctx, "my-config", k8sx.WatchOptions{
//	  Namespace: "default",
//	  Logger: logger,
//	}, func(data map[string]string) {
//	  // Handle configuration update
//	})
//	endpoints, err := k8sx.Resolve(ctx, "my-service", k8sx.ServiceKindHeadless)
package k8sx

import (
	"context"
	"fmt"
	"time"

	"github.com/eggybyte-technology/egg/core/log"
	"github.com/eggybyte-technology/egg/k8sx/internal"
)

// WatchOptions holds configuration for ConfigMap watching.
type WatchOptions struct {
	Namespace    string        // Kubernetes namespace (default: current namespace)
	ResyncPeriod time.Duration // Resync period for informer (default: 10 minutes)
	Logger       log.Logger    // Logger for watch operations
}

// ServiceKind represents the type of Kubernetes service.
type ServiceKind string

const (
	// ServiceKindHeadless represents a headless service (no ClusterIP).
	ServiceKindHeadless ServiceKind = "headless"
	// ServiceKindClusterIP represents a ClusterIP service.
	ServiceKindClusterIP ServiceKind = "clusterip"
)

// WatchConfigMap watches a ConfigMap for changes and calls the callback on updates.
// The callback function receives the ConfigMap data as a map of string key-value pairs.
// This function blocks until the context is cancelled or an error occurs.
func WatchConfigMap(ctx context.Context, name string, opts WatchOptions, onUpdate func(data map[string]string)) error {
	if opts.Logger == nil {
		return fmt.Errorf("logger is required")
	}

	// Set default namespace if not provided
	namespace := opts.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// Set default resync period
	resyncPeriod := opts.ResyncPeriod
	if resyncPeriod == 0 {
		resyncPeriod = 10 * time.Minute
	}

	// Create and start the watcher
	watcher := internal.NewConfigMapWatcher(name, namespace, opts.Logger, onUpdate)

	if err := watcher.Start(ctx); err != nil {
		return fmt.Errorf("failed to start ConfigMap watcher: %w", err)
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Stop the watcher
	if err := watcher.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop ConfigMap watcher: %w", err)
	}

	return nil
}

// Resolve resolves a Kubernetes service to its endpoints.
// For headless services, returns individual pod endpoints.
// For ClusterIP services, returns the service endpoint.
// Returns a slice of "host:port" strings.
func Resolve(ctx context.Context, service string, kind ServiceKind) ([]string, error) {
	if service == "" {
		return nil, fmt.Errorf("service name is required")
	}

	// Create service resolver
	resolver, err := internal.NewServiceResolver()
	if err != nil {
		return nil, fmt.Errorf("failed to create service resolver: %w", err)
	}

	// Resolve the service
	return resolver.ResolveService(ctx, service, string(kind))
}
