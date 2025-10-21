# ☸️ K8sX Package

The `k8sx` package provides Kubernetes integration for the EggyByte framework.

## Overview

This package offers Kubernetes-native functionality including ConfigMap monitoring, service discovery, and resource management. It's designed to be production-ready with proper error handling and resource management.

## Features

- **ConfigMap monitoring** - Watch ConfigMap changes and hot updates
- **Service discovery** - Kubernetes service discovery
- **Resource management** - Proper resource lifecycle management
- **Error handling** - Robust error handling and retry logic
- **Production ready** - Optimized for production Kubernetes environments

## Quick Start

```go
import "github.com/eggybyte-technology/egg/k8sx"

func main() {
    // Create Kubernetes client
    client, err := k8sx.NewClient(ctx, logger)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Watch ConfigMap
    watcher, err := client.WatchConfigMap(ctx, "app-config", "default")
    if err != nil {
        log.Fatal(err)
    }
    
    // Handle ConfigMap changes
    go func() {
        for event := range watcher.Events() {
            switch event.Type {
            case k8sx.EventAdded, k8sx.EventModified:
                logger.Info("ConfigMap updated", log.Str("name", event.Name))
                // Reload configuration
            case k8sx.EventDeleted:
                logger.Info("ConfigMap deleted", log.Str("name", event.Name))
            }
        }
    }()
}
```

## API Reference

### Types

#### Client

```go
type Client struct {
    clientset kubernetes.Interface
    logger    log.Logger
}

// Close closes the client
func (c *Client) Close() error

// WatchConfigMap watches a ConfigMap for changes
func (c *Client) WatchConfigMap(ctx context.Context, name, namespace string) (*Watcher, error)

// GetConfigMap gets a ConfigMap
func (c *Client) GetConfigMap(ctx context.Context, name, namespace string) (*corev1.ConfigMap, error)

// ListServices lists services in a namespace
func (c *Client) ListServices(ctx context.Context, namespace string) ([]corev1.Service, error)
```

#### Watcher

```go
type Watcher struct {
    events chan Event
    stop   chan struct{}
}

// Events returns a channel of events
func (w *Watcher) Events() <-chan Event

// Stop stops the watcher
func (w *Watcher) Stop()
```

#### Event

```go
type Event struct {
    Type      EventType
    Name      string
    Namespace string
    Data      map[string]string
    Error     error
}

type EventType int

const (
    EventAdded EventType = iota
    EventModified
    EventDeleted
    EventError
)
```

### Functions

```go
// NewClient creates a new Kubernetes client
func NewClient(ctx context.Context, logger log.Logger) (*Client, error)

// NewClientFromConfig creates a new Kubernetes client from config
func NewClientFromConfig(ctx context.Context, logger log.Logger, config *rest.Config) (*Client, error)
```

## Usage Examples

### Basic ConfigMap Watching

```go
func main() {
    // Create Kubernetes client
    client, err := k8sx.NewClient(ctx, logger)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Watch ConfigMap
    watcher, err := client.WatchConfigMap(ctx, "app-config", "default")
    if err != nil {
        log.Fatal(err)
    }
    defer watcher.Stop()
    
    // Handle ConfigMap changes
    go func() {
        for event := range watcher.Events() {
            switch event.Type {
            case k8sx.EventAdded, k8sx.EventModified:
                logger.Info("ConfigMap updated",
                    log.Str("name", event.Name),
                    log.Str("namespace", event.Namespace),
                )
                
                // Reload configuration
                if err := reloadConfiguration(event.Data); err != nil {
                    logger.Error(err, "Failed to reload configuration")
                }
                
            case k8sx.EventDeleted:
                logger.Info("ConfigMap deleted",
                    log.Str("name", event.Name),
                    log.Str("namespace", event.Namespace),
                )
                
            case k8sx.EventError:
                logger.Error(event.Error, "ConfigMap watch error")
            }
        }
    }()
    
    // Keep running
    select {}
}
```

### Service Discovery

```go
func main() {
    // Create Kubernetes client
    client, err := k8sx.NewClient(ctx, logger)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // List services
    services, err := client.ListServices(ctx, "default")
    if err != nil {
        log.Fatal(err)
    }
    
    // Process services
    for _, service := range services {
        logger.Info("Found service",
            log.Str("name", service.Name),
            log.Str("namespace", service.Namespace),
            log.Str("type", string(service.Spec.Type)),
        )
        
        // Extract service endpoints
        if service.Spec.Type == corev1.ServiceTypeClusterIP {
            endpoint := fmt.Sprintf("%s.%s.svc.cluster.local:%d",
                service.Name,
                service.Namespace,
                service.Spec.Ports[0].Port,
            )
            logger.Info("Service endpoint", log.Str("endpoint", endpoint))
        }
    }
}
```

### Configuration Management Integration

```go
type K8sConfigManager struct {
    client    *k8sx.Client
    logger    log.Logger
    configMap string
    namespace string
    data      map[string]string
    watcher   *k8sx.Watcher
}

func NewK8sConfigManager(ctx context.Context, logger log.Logger, configMap, namespace string) (*K8sConfigManager, error) {
    // Create Kubernetes client
    client, err := k8sx.NewClient(ctx, logger)
    if err != nil {
        return nil, err
    }
    
    manager := &K8sConfigManager{
        client:    client,
        logger:    logger,
        configMap: configMap,
        namespace: namespace,
        data:      make(map[string]string),
    }
    
    // Load initial configuration
    if err := manager.loadConfig(ctx); err != nil {
        return nil, err
    }
    
    // Start watching for changes
    if err := manager.startWatching(ctx); err != nil {
        return nil, err
    }
    
    return manager, nil
}

func (m *K8sConfigManager) loadConfig(ctx context.Context) error {
    configMap, err := m.client.GetConfigMap(ctx, m.configMap, m.namespace)
    if err != nil {
        return err
    }
    
    m.data = configMap.Data
    m.logger.Info("Configuration loaded from ConfigMap",
        log.Str("name", m.configMap),
        log.Str("namespace", m.namespace),
    )
    
    return nil
}

func (m *K8sConfigManager) startWatching(ctx context.Context) error {
    watcher, err := m.client.WatchConfigMap(ctx, m.configMap, m.namespace)
    if err != nil {
        return err
    }
    
    m.watcher = watcher
    
    go func() {
        for event := range watcher.Events() {
            switch event.Type {
            case k8sx.EventAdded, k8sx.EventModified:
                m.data = event.Data
                m.logger.Info("Configuration updated from ConfigMap")
                
            case k8sx.EventDeleted:
                m.logger.Warn("ConfigMap deleted, using cached configuration")
                
            case k8sx.EventError:
                m.logger.Error(event.Error, "ConfigMap watch error")
            }
        }
    }()
    
    return nil
}

func (m *K8sConfigManager) Get(key string) (string, bool) {
    value, exists := m.data[key]
    return value, exists
}

func (m *K8sConfigManager) Close() error {
    if m.watcher != nil {
        m.watcher.Stop()
    }
    return m.client.Close()
}
```

### Health Check Integration

```go
func main() {
    // Create Kubernetes client
    client, err := k8sx.NewClient(ctx, logger)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Create health check handler
    mux := http.NewServeMux()
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        // Check Kubernetes connectivity
        if err := checkK8sConnectivity(client); err != nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            w.Write([]byte("Kubernetes connectivity failed"))
            return
        }
        
        // Check ConfigMap availability
        if err := checkConfigMapAvailability(client); err != nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            w.Write([]byte("ConfigMap unavailable"))
            return
        }
        
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })
    
    // Start server
    server := &http.Server{
        Addr:    ":8081",
        Handler: mux,
    }
    
    go func() {
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()
    
    // Graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        log.Fatal(err)
    }
}

func checkK8sConnectivity(client *k8sx.Client) error {
    // Try to list services to check connectivity
    _, err := client.ListServices(context.Background(), "default")
    return err
}

func checkConfigMapAvailability(client *k8sx.Client) error {
    // Try to get ConfigMap to check availability
    _, err := client.GetConfigMap(context.Background(), "app-config", "default")
    return err
}
```

## Configuration

### Environment Variables

```bash
# Kubernetes configuration
KUBECONFIG=/path/to/kubeconfig
KUBERNETES_SERVICE_HOST=kubernetes.default.svc.cluster.local
KUBERNETES_SERVICE_PORT=443

# ConfigMap configuration
CONFIGMAP_NAME=app-config
CONFIGMAP_NAMESPACE=default
```

### Kubernetes RBAC

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: app-service-account
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: app-config-reader
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["services"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: app-config-reader-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: app-config-reader
subjects:
- kind: ServiceAccount
  name: app-service-account
  namespace: default
```

## Testing

```go
func TestK8sClient(t *testing.T) {
    // Create test client
    client, err := k8sx.NewClient(context.Background(), &TestLogger{})
    assert.NoError(t, err)
    defer client.Close()
    
    // Test ConfigMap operations
    configMap, err := client.GetConfigMap(context.Background(), "test-config", "default")
    if err != nil {
        // ConfigMap might not exist in test environment
        t.Logf("ConfigMap not found: %v", err)
    } else {
        assert.NotNil(t, configMap)
    }
    
    // Test service listing
    services, err := client.ListServices(context.Background(), "default")
    assert.NoError(t, err)
    assert.NotNil(t, services)
}

func TestConfigMapWatcher(t *testing.T) {
    // Create test client
    client, err := k8sx.NewClient(context.Background(), &TestLogger{})
    assert.NoError(t, err)
    defer client.Close()
    
    // Create watcher
    watcher, err := client.WatchConfigMap(context.Background(), "test-config", "default")
    if err != nil {
        // Watcher might not work in test environment
        t.Logf("Watcher creation failed: %v", err)
        return
    }
    defer watcher.Stop()
    
    // Test watcher events channel
    assert.NotNil(t, watcher.Events())
}

type TestLogger struct{}

func (l *TestLogger) With(kv ...any) log.Logger { return l }
func (l *TestLogger) Debug(msg string, kv ...any) {}
func (l *TestLogger) Info(msg string, kv ...any) {}
func (l *TestLogger) Warn(msg string, kv ...any) {}
func (l *TestLogger) Error(err error, msg string, kv ...any) {}
```

## Best Practices

### 1. Proper Resource Management

```go
func main() {
    // Create Kubernetes client
    client, err := k8sx.NewClient(ctx, logger)
    if err != nil {
        log.Fatal(err)
    }
    
    // Ensure proper cleanup
    defer func() {
        if err := client.Close(); err != nil {
            logger.Error(err, "Failed to close Kubernetes client")
        }
    }()
    
    // Use client
    useClient(client)
}
```

### 2. Error Handling

```go
func watchConfigMap(ctx context.Context, client *k8sx.Client) error {
    watcher, err := client.WatchConfigMap(ctx, "app-config", "default")
    if err != nil {
        return errors.Wrap(err, "K8S_ERROR", "failed to watch ConfigMap")
    }
    defer watcher.Stop()
    
    for event := range watcher.Events() {
        switch event.Type {
        case k8sx.EventError:
            logger.Error(event.Error, "ConfigMap watch error")
            // Implement retry logic if needed
            
        case k8sx.EventAdded, k8sx.EventModified:
            if err := handleConfigUpdate(event.Data); err != nil {
                logger.Error(err, "Failed to handle config update")
            }
            
        case k8sx.EventDeleted:
            logger.Warn("ConfigMap deleted, using cached configuration")
        }
    }
    
    return nil
}
```

### 3. Context Usage

```go
func getConfigMap(ctx context.Context, client *k8sx.Client) (*corev1.ConfigMap, error) {
    // Use context for cancellation and timeouts
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    configMap, err := client.GetConfigMap(ctx, "app-config", "default")
    if err != nil {
        return nil, errors.Wrap(err, "K8S_ERROR", "failed to get ConfigMap")
    }
    
    return configMap, nil
}
```

### 4. Graceful Shutdown

```go
func main() {
    // Create Kubernetes client
    client, err := k8sx.NewClient(ctx, logger)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start watcher
    watcher, err := client.WatchConfigMap(ctx, "app-config", "default")
    if err != nil {
        log.Fatal(err)
    }
    
    // Handle shutdown signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        logger.Info("Shutting down...")
        
        // Stop watcher
        watcher.Stop()
        
        // Close client
        if err := client.Close(); err != nil {
            logger.Error(err, "Failed to close Kubernetes client")
        }
        
        os.Exit(0)
    }()
    
    // Keep running
    select {}
}
```

## Thread Safety

All functions in this package are safe for concurrent use. The Kubernetes client is designed to handle concurrent access safely.

## Dependencies

- **Go 1.21+** required
- **Kubernetes client-go** - Kubernetes API client
- **Standard library** - Core functionality

## Version Compatibility

- **Go 1.21+** required
- **API Stability**: Evolving (L3 module)
- **Breaking Changes**: Possible in minor versions

## Contributing

Contributions are welcome! Please see the main project [Contributing Guide](../CONTRIBUTING.md) for details.

## License

This package is part of the EggyByte framework and is licensed under the MIT License.