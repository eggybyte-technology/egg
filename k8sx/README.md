# egg/k8sx

## Overview

`k8sx` provides Kubernetes integration for ConfigMap watching and service discovery.
It enables dynamic configuration updates and service endpoint resolution in
Kubernetes environments.

## Key Features

- ConfigMap watching with change notifications
- Service discovery (ClusterIP and Headless)
- Automatic reconnection on failures
- Clean interface abstraction
- Resync support for consistency

## Dependencies

Layer: **Auxiliary (Kubernetes Layer)**  
Depends on: `core/log`, `k8s.io/client-go`

## Installation

```bash
go get github.com/eggybyte-technology/egg/k8sx@latest
```

## Basic Usage

### ConfigMap Watching

```go
import (
    "context"
    "github.com/eggybyte-technology/egg/k8sx"
)

func main() {
    ctx := context.Background()
    
    // Watch ConfigMap for changes
    err := k8sx.WatchConfigMap(ctx, "my-config", k8sx.WatchOptions{
        Namespace: "default",
        Logger:    logger,
    }, func(data map[string]string) {
        // Called when ConfigMap changes
        logger.Info("config updated", "keys", len(data))
        
        // Update application configuration
        updateConfig(data)
    })
    
    if err != nil {
        log.Fatal(err)
    }
}
```

### Service Discovery

```go
// Resolve headless service endpoints
endpoints, err := k8sx.Resolve(ctx, "my-service", k8sx.ServiceKindHeadless)
if err != nil {
    log.Fatal(err)
}

// endpoints = ["10.0.1.5:8080", "10.0.1.6:8080", "10.0.1.7:8080"]

// Resolve ClusterIP service
endpoints, err := k8sx.Resolve(ctx, "my-service", k8sx.ServiceKindClusterIP)
// endpoints = ["my-service.default.svc.cluster.local:8080"]
```

## API Reference

### ConfigMap Watching

```go
// WatchConfigMap watches a ConfigMap for changes and calls the callback on updates
func WatchConfigMap(
    ctx context.Context,
    name string,
    opts WatchOptions,
    onUpdate func(data map[string]string),
) error

type WatchOptions struct {
    Namespace    string        // Kubernetes namespace (default: current namespace)
    ResyncPeriod time.Duration // Resync period for informer (default: 10 minutes)
    Logger       log.Logger    // Logger for watch operations
}
```

### Service Discovery

```go
// Resolve resolves a Kubernetes service to its endpoints
func Resolve(ctx context.Context, service string, kind ServiceKind) ([]string, error)

type ServiceKind string

const (
    // ServiceKindHeadless represents a headless service (no ClusterIP)
    ServiceKindHeadless ServiceKind = "headless"
    
    // ServiceKindClusterIP represents a ClusterIP service
    ServiceKindClusterIP ServiceKind = "clusterip"
)
```

## Architecture

The k8sx module provides Kubernetes integration:

```
k8sx/
├── k8sx.go              # Public API (~103 lines)
│   ├── WatchConfigMap() # ConfigMap watching
│   ├── Resolve()        # Service discovery
│   └── Types            # WatchOptions, ServiceKind
└── internal/
    ├── watcher.go       # ConfigMap watcher implementation
    │   └── Start()      # Start watching
    │   └── Stop()       # Stop watching
    └── resolver.go      # Service resolver implementation
        └── ResolveService()  # Resolve endpoints
```

**Design Highlights:**
- Public interface is simple and focused
- Informer-based watching for efficiency
- Automatic reconnection on failures
- Clean shutdown support

## Example: Dynamic Configuration

```go
package main

import (
    "context"
    "sync"
    
    "github.com/eggybyte-technology/egg/k8sx"
)

type AppConfig struct {
    DatabaseURL string
    MaxConns    int
    Debug       bool
}

var (
    config *AppConfig
    mu     sync.RWMutex
)

func main() {
    ctx := context.Background()
    
    // Watch ConfigMap
    err := k8sx.WatchConfigMap(ctx, "app-config", k8sx.WatchOptions{
        Namespace: "production",
        Logger:    logger,
    }, func(data map[string]string) {
        // Parse config
        newConfig := &AppConfig{
            DatabaseURL: data["DATABASE_URL"],
            MaxConns:    parseInt(data["MAX_CONNS"]),
            Debug:       parseBool(data["DEBUG"]),
        }
        
        // Update config atomically
        mu.Lock()
        config = newConfig
        mu.Unlock()
        
        logger.Info("configuration reloaded",
            "database_url", newConfig.DatabaseURL,
            "max_conns", newConfig.MaxConns,
        )
    })
    
    if err != nil {
        log.Fatal(err)
    }
}

func getConfig() *AppConfig {
    mu.RLock()
    defer mu.RUnlock()
    return config
}
```

## Example: Service Discovery

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/eggybyte-technology/egg/k8sx"
)

func main() {
    ctx := context.Background()
    
    // Discover headless service pods
    endpoints, err := k8sx.Resolve(ctx, "backend-service", k8sx.ServiceKindHeadless)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Backend service endpoints:\n")
    for _, ep := range endpoints {
        fmt.Printf("  - %s\n", ep)
    }
    
    // Load balance requests across pods
    for _, endpoint := range endpoints {
        resp, err := http.Get(fmt.Sprintf("http://%s/health", endpoint))
        if err != nil {
            log.Printf("Failed to connect to %s: %v", endpoint, err)
            continue
        }
        resp.Body.Close()
        fmt.Printf("%s is healthy\n", endpoint)
    }
}
```

## Example: Integration with configx

```go
import (
    "github.com/eggybyte-technology/egg/configx"
    "github.com/eggybyte-technology/egg/k8sx"
)

func main() {
    ctx := context.Background()
    
    // Create ConfigMap source
    k8sSource := configx.NewK8sConfigMapSource("app-config", configx.K8sOptions{
        Namespace: "default",
        Logger:    logger,
    })
    
    // Create config manager with multiple sources
    manager, err := configx.NewManager(ctx, configx.Options{
        Logger: logger,
        Sources: []configx.Source{
            configx.NewEnvSource(configx.EnvOptions{}),
            k8sSource,  // ConfigMap overrides env vars
        },
    })
    
    // Bind configuration
    var cfg AppConfig
    manager.Bind(&cfg)
    
    // Configuration auto-reloads when ConfigMap changes
}
```

## Kubernetes Resources

### ConfigMap Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: default
data:
  DATABASE_URL: "postgres://db:5432/myapp"
  MAX_CONNS: "50"
  DEBUG: "false"
  CACHE_TTL: "300"
```

### Headless Service Example

```yaml
apiVersion: v1
kind: Service
metadata:
  name: backend-service
  namespace: default
spec:
  clusterIP: None  # Headless service
  selector:
    app: backend
  ports:
  - port: 8080
    targetPort: 8080
```

## In-Cluster vs Out-of-Cluster

### In-Cluster (Running in Kubernetes)

```go
// Uses in-cluster config automatically
err := k8sx.WatchConfigMap(ctx, "app-config", k8sx.WatchOptions{
    Namespace: "default",  // Or get from downward API
    Logger:    logger,
}, onUpdate)
```

### Out-of-Cluster (Local Development)

```bash
# Set KUBECONFIG environment variable
export KUBECONFIG=~/.kube/config
```

```go
// Will use ~/.kube/config
err := k8sx.WatchConfigMap(ctx, "app-config", k8sx.WatchOptions{
    Namespace: "default",
    Logger:    logger,
}, onUpdate)
```

## RBAC Requirements

Your service account needs appropriate permissions:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-app
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: configmap-reader
  namespace: default
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "watch", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: my-app-configmap-reader
  namespace: default
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: configmap-reader
subjects:
- kind: ServiceAccount
  name: my-app
  namespace: default
```

## Best Practices

1. **Use service accounts** - Configure RBAC properly
2. **Handle watch failures** - Implement retry logic
3. **Validate configuration** - Validate data before applying
4. **Log config changes** - Track when configuration updates
5. **Graceful degradation** - Continue running if watch fails
6. **Namespace isolation** - Use appropriate namespace
7. **Test locally** - Use kubeconfig for local development

## Error Handling

```go
err := k8sx.WatchConfigMap(ctx, "app-config", opts, func(data map[string]string) {
    if err := validateConfig(data); err != nil {
        logger.Error(err, "invalid configuration, keeping previous config")
        return
    }
    
    if err := applyConfig(data); err != nil {
        logger.Error(err, "failed to apply configuration")
        return
    }
    
    logger.Info("configuration applied successfully")
})

if err != nil {
    logger.Error(err, "failed to start config watcher")
    // Fallback to environment variables or default config
}
```

## Testing

```go
func TestConfigWatcher(t *testing.T) {
    // For testing, use fake Kubernetes client
    // or test with environment variables instead
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    updateCalled := false
    
    err := k8sx.WatchConfigMap(ctx, "test-config", k8sx.WatchOptions{
        Namespace: "default",
        Logger:    testLogger,
    }, func(data map[string]string) {
        updateCalled = true
        assert.Contains(t, data, "DATABASE_URL")
    })
    
    // In real cluster, this would work
    // For tests, use mock or integration tests
}
```

## Stability

**Status**: Stable  
**Layer**: Auxiliary (Kubernetes)  
**API Guarantees**: Backward-compatible changes only

The k8sx module is production-ready and follows semantic versioning.

## License

This package is part of the egg framework and is licensed under the MIT License.
See the root LICENSE file for details.
