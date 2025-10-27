// Package internal contains Kubernetes ConfigMap watcher implementation.
package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"go.eggybyte.com/egg/core/log"
)

// ConfigMapWatcher watches a ConfigMap for changes using Kubernetes client-go.
type ConfigMapWatcher struct {
	name      string
	namespace string
	logger    log.Logger
	onUpdate  func(data map[string]string)
	client    kubernetes.Interface
	stopCh    chan struct{}
	mu        sync.RWMutex
	isRunning bool
}

// NewConfigMapWatcher creates a new ConfigMap watcher.
func NewConfigMapWatcher(name, namespace string, logger log.Logger, onUpdate func(data map[string]string)) *ConfigMapWatcher {
	return &ConfigMapWatcher{
		name:      name,
		namespace: namespace,
		logger:    logger,
		onUpdate:  onUpdate,
		stopCh:    make(chan struct{}),
	}
}

// Start starts watching the ConfigMap.
func (w *ConfigMapWatcher) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isRunning {
		return fmt.Errorf("watcher is already running")
	}

	// Create Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig for local development
		config, err = rest.InClusterConfig()
		if err != nil {
			return fmt.Errorf("failed to create Kubernetes config: %w", err)
		}
	}

	w.client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	w.logger.Info("starting ConfigMap watcher",
		log.Str("name", w.name),
		log.Str("namespace", w.namespace))

	// Start watching in a goroutine
	go w.watch(ctx)

	w.isRunning = true
	return nil
}

// Stop stops watching the ConfigMap.
func (w *ConfigMapWatcher) Stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isRunning {
		return nil
	}

	w.logger.Info("stopping ConfigMap watcher",
		log.Str("name", w.name),
		log.Str("namespace", w.namespace))

	close(w.stopCh)
	w.isRunning = false
	return nil
}

// watch performs the actual watching of the ConfigMap.
func (w *ConfigMapWatcher) watch(ctx context.Context) {
	defer func() {
		w.logger.Info("ConfigMap watcher stopped",
			log.Str("name", w.name),
			log.Str("namespace", w.namespace))
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		default:
		}

		// Create watcher
		watcher, err := w.client.CoreV1().ConfigMaps(w.namespace).Watch(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", w.name),
		})
		if err != nil {
			w.logger.Error(err, "failed to create ConfigMap watcher",
				log.Str("name", w.name),
				log.Str("namespace", w.namespace))

			// Wait before retrying
			select {
			case <-ctx.Done():
				return
			case <-w.stopCh:
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		// Process events
		func() {
			defer watcher.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-w.stopCh:
					return
				case event, ok := <-watcher.ResultChan():
					if !ok {
						w.logger.Warn("ConfigMap watcher channel closed",
							log.Str("name", w.name),
							log.Str("namespace", w.namespace))
						return
					}

					switch event.Type {
					case watch.Added, watch.Modified:
						if cm, ok := event.Object.(*corev1.ConfigMap); ok {
							w.logger.Info("ConfigMap updated",
								log.Str("name", cm.Name),
								log.Str("namespace", cm.Namespace),
								log.Int("data_keys", len(cm.Data)))

							if w.onUpdate != nil {
								w.onUpdate(cm.Data)
							}
						}
					case watch.Deleted:
						w.logger.Info("ConfigMap deleted",
							log.Str("name", w.name),
							log.Str("namespace", w.namespace))

						if w.onUpdate != nil {
							w.onUpdate(make(map[string]string))
						}
					case watch.Error:
						w.logger.Error(nil, "ConfigMap watcher error",
							log.Str("name", w.name),
							log.Str("namespace", w.namespace))
					}
				}
			}
		}()

		// Wait before recreating watcher
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-time.After(1 * time.Second):
		}
	}
}
