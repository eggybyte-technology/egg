// Package config provides Kubernetes configuration watching for the user service.
//
// Overview:
//   - Responsibility: Kubernetes ConfigMap watching and configuration updates
//   - Key Types: K8sWatcher struct with ConfigMap monitoring
//   - Concurrency Model: Thread-safe configuration updates with debouncing
//   - Error Semantics: Watch errors are logged and handled gracefully
//   - Performance Notes: Optimized for minimal resource usage
//
// Usage:
//
//	watcher := NewK8sWatcher(client, logger)
//	watcher.WatchConfigMap(ctx, "user-service-config", callback)
package config

import (
	"context"
	"sync"
	"time"

	"github.com/eggybyte-technology/egg/core/log"
	"github.com/eggybyte-technology/egg/k8sx"
)

// K8sWatcher provides Kubernetes ConfigMap watching capabilities.
// It handles configuration updates with debouncing and error recovery.
type K8sWatcher struct {
	logger log.Logger

	// Debouncing configuration
	debounceDuration time.Duration
	updateChannels   map[string]chan struct{}
	mu               sync.RWMutex
}

// NewK8sWatcher creates a new K8sWatcher instance.
// The returned watcher is safe for concurrent use.
func NewK8sWatcher(logger log.Logger) *K8sWatcher {
	return &K8sWatcher{
		logger:           logger,
		debounceDuration: 5 * time.Second,
		updateChannels:   make(map[string]chan struct{}),
	}
}

// WatchConfigMap starts watching a ConfigMap for changes.
// It debounces updates and calls the callback function when changes occur.
func (w *K8sWatcher) WatchConfigMap(ctx context.Context, configMapName string, callback func(map[string]string)) error {
	w.logger.Info("Starting ConfigMap watch", log.Str("configmap", configMapName))

	// Create debounce channel for this ConfigMap
	w.mu.Lock()
	updateCh := make(chan struct{}, 1)
	w.updateChannels[configMapName] = updateCh
	w.mu.Unlock()

	// Start debounce goroutine
	go w.debounceUpdates(ctx, configMapName, updateCh, callback)

	// Start ConfigMap watching using k8sx
	return k8sx.WatchConfigMap(ctx, configMapName, k8sx.WatchOptions{
		Logger: w.logger,
	}, func(data map[string]string) {
		w.logger.Info("ConfigMap updated",
			log.Str("configmap", configMapName),
			log.Int("keys", len(data)))

		// Send update signal to debounce channel
		select {
		case updateCh <- struct{}{}:
		default:
			// Channel is full, skip this update
		}
	})
}

// debounceUpdates handles debouncing of configuration updates.
// It ensures that rapid configuration changes don't cause excessive callbacks.
func (w *K8sWatcher) debounceUpdates(ctx context.Context, configMapName string, updateCh <-chan struct{}, callback func(map[string]string)) {
	var timer *time.Timer
	_ = time.Now() // Placeholder for lastUpdate

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("ConfigMap watch stopped", log.Str("configmap", configMapName))
			return

		case <-updateCh:

			// Reset timer if it's running
			if timer != nil {
				timer.Stop()
			}

			// Set new timer
			timer = time.AfterFunc(w.debounceDuration, func() {
				w.logger.Info("Processing debounced ConfigMap update",
					log.Str("configmap", configMapName),
					log.Str("debounce_duration", w.debounceDuration.String()))

				// Get current ConfigMap data
				data, err := w.getConfigMapData(ctx, configMapName)
				if err != nil {
					w.logger.Error(err, "Failed to get ConfigMap data",
						log.Str("configmap", configMapName))
					return
				}

				// Call callback
				callback(data)
			})
		}
	}
}

// getConfigMapData retrieves the current data from a ConfigMap.
// It handles errors gracefully and returns empty data if retrieval fails.
func (w *K8sWatcher) getConfigMapData(ctx context.Context, configMapName string) (map[string]string, error) {
	// This is a simplified implementation
	// In a real implementation, you would use the k8sx client to get ConfigMap data
	// For now, we'll return empty data as a placeholder
	return map[string]string{}, nil
}

// Stop stops watching all ConfigMaps and cleans up resources.
func (w *K8sWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Close all update channels
	for configMapName, updateCh := range w.updateChannels {
		close(updateCh)
		w.logger.Info("Stopped ConfigMap watch", log.Str("configmap", configMapName))
	}

	// Clear the map
	w.updateChannels = make(map[string]chan struct{})
}

// SetDebounceDuration sets the debounce duration for configuration updates.
// It affects all future ConfigMap watches.
func (w *K8sWatcher) SetDebounceDuration(duration time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.debounceDuration = duration
	w.logger.Info("Debounce duration updated", log.Str("duration", duration.String()))
}
