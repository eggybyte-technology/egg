// Package k8sx provides Kubernetes ConfigMap watching and basic service
// resolution utilities.
//
// # Overview
//
// k8sx offers a small API surface to watch ConfigMaps for configuration
// updates and resolve service endpoints in cluster environments. It hides
// client-go complexity behind stable, testable interfaces.
//
// # Features
//
//   - Debounced ConfigMap watching with callback hooks
//   - Service endpoint resolution (headless and ClusterIP)
//   - Context cancellation and resource-safe stop semantics
//
// # Usage
//
//	err := k8sx.WatchConfigMap(ctx, "app-config", k8sx.WatchOptions{Namespace: "default", Logger: logger}, func(data map[string]string) {
//		// apply updates
//	})
//
// # Layer
//
// k8sx is an auxiliary module and depends on core/log minimally.
//
// # Stability
//
// Experimental until v0.1.0.
package k8sx
